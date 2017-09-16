package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
	"os"
	"io"
	"flag"
	"sync"
	// it is just better than the std log
	log "github.com/sirupsen/logrus"
	"crypto/rand"
)

const (
	basePath = "/"
	window   = time.Second * 60
)

// journal is the journal file path
var journalPath = flag.String("journal", "journal.data", "path to your journal file [no file will create one]")
var debug = flag.Bool("debug", false, "show some information")

func main() {
	// parse flags
	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Debug("loading journal file:", *journalPath)
	// load journal
	j, err := Load(*journalPath)
	if err != nil {
		log.Fatal(err)
	}
	defer j.Close()
	log.Debug("journal file was loaded")
	// new http mux with simple handler to catch basePath requests
	mux := http.NewServeMux()
	mux.HandleFunc(basePath, func(w http.ResponseWriter, req *http.Request) {
		// serve just our path
		if req.URL.Path != basePath {
			http.NotFound(w, req)
			return
		}
		// append entry to journal
		j.Append(time.Now(), true)
		// return the latest counter
		fmt.Fprintf(w, "Counter: %o", j.Counter())
	})
	log.Info("listening on :8080")
	// serve on :8080
	log.Fatal(http.ListenAndServe(":8080", mux))
}

type entry struct {
	id string
	t  time.Time
	ch <-chan time.Time
}

type journal struct {
	f          *os.File
	entriesMap sync.Map
	counter    uint64
	mtx        sync.Mutex
}

// Load loads the journal file into memory and process it
func Load(path string) (*journal, error) {
	// open the journal file or create it with 777 permissions :P
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 777)
	if err != nil {
		return nil, err
	}
	// new journal
	j := &journal{
		f:          f,
		counter:    0,
		mtx:        sync.Mutex{},
		entriesMap: sync.Map{},
		//entries: []entry{},
	}
	// 15 bytes is the siz of binary encoded time struct
	timeByte := make([]byte, 15)
	t := time.Time{}
	// read entries from file till eof or error reached
	for {
		_, err := f.Read(timeByte)
		// break loop at eof
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Panic(err)
		}
		if err := t.GobDecode(timeByte); err != nil {
			continue
		}
		// if time within the window
		// inc the counter
		if time.Now().Sub(t) <= window {
			j.Append(t, false)
		}
	}
	// journal file is loaded up
	// truncate file and write updated entries
	// to keep fresh journal file
	log.Debug("truncating journal")
	if err := f.Truncate(0); err != nil {
		return nil, err
	}
	// seek to beginning
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	// write current slice
	log.Debug("writing new entries to journal file")
	j.entriesMap.Range(func(key, value interface{}) bool {
		if t, ok := value.(time.Time); ok {
			b, err := t.GobEncode()
			if err != nil {
				log.Fatal(err)
			}
			if _, err := j.f.Write(b); err != nil {
				log.Fatal(err)
			}
		}
		return true
	})
	return j, nil
}

// GetCounter returns the counter value
func (j *journal) Counter() uint64 {
	x := atomic.LoadUint64(&j.counter)
	log.Debugf("load journal counter %d", x)
	return x
}

// Close close journal file
func (j *journal) Close() error {
	log.Debug("closing journal file")
	return j.f.Close()
}

// Append append time entry to journal
func (j *journal) Append(t time.Time, writeToFile bool) {
	log.Debugf("append %s to journal", t.String())
	if writeToFile {
		// encode time to []byte
		b, err := t.GobEncode()
		if err != nil {
			log.Fatal(err)
		}
		if _, err := j.f.Write(b); err != nil {
			log.Fatal(err)
		}
	}
	// increment the counter in different goroutine
	atomic.AddUint64(&j.counter, 1)
	// append the entry to entries
	je := entry{
		id: randId(),
		t:  t,
		ch: time.After(window),
	}
	j.mtx.Lock()
	j.entriesMap.Store(je.id, je)
	//j.entries = append(j.entries, je)
	j.mtx.Unlock()
	go j.Listen(je.id, je.ch)
}

// Sync sync journal entries every t (duration)
func (j *journal) Listen(id string, tCh <-chan time.Time) {
	for {
		select {
		case <-tCh:
			log.Debug("Got time.after chan message... Remove from journal")
			if val, ok := j.entriesMap.Load(id); ok {
				if e, ok := val.(entry); ok {
					if time.Now().Sub(e.t) > window {
						// remove from map
						j.entriesMap.Delete(id)
						// decrement the counter
						atomic.AddUint64(&j.counter, ^uint64(0))
					}
				}
			}
			log.Debugf("current counter:%d", j.Counter())
			return
		}
	}
}

func randId() (string) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Panic(err)
	}
	return string(b)
}
