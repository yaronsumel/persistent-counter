# persistent-counter

### Description

It is small coding challenge for company `x`.
The task is to count the total number of requests in specific time window (default is 60 seconds), It should also work when restarting the server.

### Concept

Once started it will try to open journal file, if it is not located it should create one. The journal file is basically binary encoded of `time.Time` entries, each entry is in 15 bytes long. At the load time it will iterate over the binary data and try to decode it into `time.Time`, If that went fine it will append that entry to `sync.map` if the entry fits the time window. Each entry is created with random `id` and `chan time.Time` which will trigger listeners when it needs to be wiped out.

### Prerequisites

Go > 1.9 (making use of sync.map)

### Install

```bash 
$ go get github.com/yaronsumel/persistent-counter
```

### Run 

```bash 
$ persistent-counter --debug
```

### Usage 
```bash
$ persistent-counter --help
 Usage of persistent-counter:
  -debug
        show some information
  -journal string
        path to your journal file [no file will create one] (default "journal.data")
```        
