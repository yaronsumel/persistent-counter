package main

import (
	"fmt"
	"net/http"
	"time"
	"flag"
	// it is just better than the std log
	log "github.com/sirupsen/logrus"
	"github.com/yaronsumel/persistent-counter/journal"
)

const basePath = "/"

// journal is the journal file path
var journalPath = flag.String("journal", "/tmp/journal.data", "path to your journal file [no file will create one]")
var port = flag.String("port", ":8080", "port to bind")
var window = flag.Int64("window", 60, "time window in seconds")
var debug = flag.Bool("debug", false, "show some information")

func main() {

	// parse flags
	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Debug("loading journal file:", *journalPath)
	// load journal
	j, err := journal.Load(*journalPath, time.Duration(*window)*time.Second)
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
		fmt.Fprintf(w, "Counter: %d", j.Counter())
	})

	log.Infof("listening on %s", *port)
	// serve on :8080
	log.Fatal(http.ListenAndServe(*port, mux))

}
