package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

func main() {
	measurements := make(chan *httpMeasurement)
	urls := make(chan *URLEntry)
	workers := &sync.WaitGroup{}
	for idx := 0; idx < 20; idx++ {
		workers.Add(1)
		go measureAsync(idx, workers, urls, measurements)
	}
	collector := make(chan interface{})
	go collectResults("results.jsonl", measurements, collector)
	readers := &sync.WaitGroup{}
	for _, name := range os.Args[1:] {
		if strings.Contains(name, "00-LEGEND-") {
			continue
		}
		readers.Add(1)
		go generateURLs(readers, name, urls)
	}
	readers.Wait()
	close(urls)
	workers.Wait()
	close(measurements)
	<-collector
}

// collectResults collects the results and writes them to
// the output file as a sequence of JSONL files.
func collectResults(filepath string, measurements <-chan *httpMeasurement,
	collector chan<- interface{}) {
	filep, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	// the parent closes measurements when all goroutines terminate
	for m := range measurements {
		data, err := json.Marshal(m)
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, '\n')
		if _, err := filep.Write(data); err != nil {
			log.Fatal(err)
		}
	}
	if err := filep.Close(); err != nil {
		log.Fatal(err)
	}
	close(collector)
}
