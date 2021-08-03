package main

import (
	"log"
	"path/filepath"
	"sync"
)

// URLEntry is an entry in the test list.
type URLEntry struct {
	// URL is the URL itself.
	URL string

	// Filename is the test list file name.
	Filename string
}

// generateURLs emits the URLs onto the urls channel and closes
// such a channel when all the URLs have been emitted.
func generateURLs(wg *sync.WaitGroup, path string, urls chan<- *URLEntry) {
	basename := filepath.Base(path)
	testList, err := readTestList(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range testList {
		urls <- &URLEntry{
			URL:      entry.URL,
			Filename: basename,
		}
	}
	wg.Done()
}
