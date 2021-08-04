package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
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

	// BodySize is the size of the response body.
	BodySize int

	// Idx is the index of this response in the redirect chain. The
	// first response in the chain has index zero.
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
	newMeasurer(m).doAll(ctx, URL.URL)
	return m
}

// measurer performs the real measurement. All fields are
// mandatory. Use newMeasurer to construct a measurer.
type measurer struct {
	// clnt is the underlying client.
	clnt *http.Client

	// m is the measurement we're filling.
	m *httpMeasurement
}

// newMeasurer creates a new instance of measurer.
func newMeasurer(m *httpMeasurement) *measurer {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err) // this should not happen.
	}
	return &measurer{
		clnt: &http.Client{
			Transport: &http.Transport{},
			Jar:       jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		m: m,
	}
}

// do performs the measurement and saves the results into m.
func (m *measurer) doAll(ctx context.Context, URL string) {
	defer m.clnt.CloseIdleConnections()
	for URL != "" {
		if len(m.m.Responses) > 10 {
			f := "too many redirections"
			m.m.Failure = &f
			return
		}
		URL = m.doOne(ctx, URL)
	}
}

// doOne performs a single measurement and saves the results into m. Return
// the next URL in the redirection chain or an empty string.
func (m *measurer) doOne(ctx context.Context, URL string) string {
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		f := err.Error()
		m.m.Failure = &f
		return ""
	}
	resp, err := m.clnt.Do(req)
	if err != nil {
		f := err.Error()
		m.m.Failure = &f
		return ""
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		f := err.Error()
		m.m.Failure = &f
		return ""
	}
	m.m.Responses = append(m.m.Responses, httpResponse{
		Code:     resp.StatusCode,
		Idx:      len(m.m.Responses),
		BodySize: len(data),
		Request: httpRequest{
			URL: resp.Request.URL.String(),
		},
	})
	location, err := resp.Location()
	if err != nil && !errors.Is(err, http.ErrNoLocation) {
		f := err.Error()
		m.m.Failure = &f
		return ""
	}
	if errors.Is(err, http.ErrNoLocation) {
		return ""
	}
	return location.String()
}
