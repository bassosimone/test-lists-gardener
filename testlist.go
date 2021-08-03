package main

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
)

// testListEntry is an entry of a test list.
type testListEntry struct {
	// URL is the URL.
	URL string

	// CategoryCode is the category code.
	CategoryCode string

	// CategoryDescription describes the category.
	CategoryDescription string

	// DateAdded is when the entry was added.
	DateAdded string

	// Source is who added the entry.
	Source string

	// Notes contains free-form textual notes.
	Notes string
}

// readTestList reads a test-list file.
func readTestList(filepath string) ([]testListEntry, error) {
	filep, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(filep)
	var all []testListEntry
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) != 6 {
			// this record seems malformed but in theory this
			// cannot happen because the csv library should return
			// an error in case we see a short record.
			panic("should not happen")
		}
		all = append(all, testListEntry{
			URL:                 record[0],
			CategoryCode:        record[1],
			CategoryDescription: record[2],
			DateAdded:           record[3],
			Source:              record[4],
			Notes:               record[5],
		})
	}
	return all, nil
}
