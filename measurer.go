package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// httpMeasurement is an HTTP measurement.
type httpMeasurement struct {
	// Failure is the error that occurred.
	Failure *string

	// OrigURL is the original URL inside the test list.
	OrigURL string

	// Filename is the original file name.
	Filename string

	// Responses contains the responses. The first response is
	// the final one and subsequent ones are redirects.
	Responses []httpResponse
}

// httpResponse is an HTTP response.
type httpResponse struct {
	// Code is the status code.
	Code int

	// Idx is the index of this response in the redirect chain. The
	// last response in the chain has index zero.
	Idx int

	// Request is the corresponding request.
	Request httpRequest
}

// httpRequest is an HTTP request.
type httpRequest struct {
	// URL is the request URL.
	URL string
}

// measureAsync reads URLs from a channel and writes measurements onto a channel
// until the channel from which we read URLs is closed. We call wg.Done when
// we're about to exit because the input channel has been closed.
func measureAsync(idx int, wg *sync.WaitGroup, urls <-chan *URLEntry,
	measurements chan<- *httpMeasurement) {
	for URL := range urls {
		log.Printf("[%d] measuring %s...", idx, URL)
		measurements <- measure(URL)
		log.Printf("[%d] measuring %s... done", idx, URL)
	}
	wg.Done()
}

// measure measures the given URL.
func measure(URL *URLEntry) *httpMeasurement {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return measureWithContext(ctx, URL)
}

// measureWithContext is like measure but with a context.
func measureWithContext(ctx context.Context, URL *URLEntry) *httpMeasurement {
	m := &httpMeasurement{OrigURL: URL.URL, Filename: URL.Filename}
	req, err := http.NewRequestWithContext(ctx, "GET", URL.URL, nil)
	if err != nil {
		f := err.Error()
		m.Failure = &f
		return m
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		f := err.Error()
		m.Failure = &f
		return m
	}
	for idx := 0; resp != nil; idx++ {
		m.Responses = append(m.Responses, httpResponse{
			Code: resp.StatusCode,
			Idx:  idx,
			Request: httpRequest{
				URL: resp.Request.URL.String(),
			},
		})
		resp = resp.Request.Response
	}
	return m
}
