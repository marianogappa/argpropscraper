package main

import (
	"encoding/csv"
	"io"
	"log"
	"sync"
)

type csvWriter struct {
	ch   chan map[string]string
	wg   *sync.WaitGroup
	file io.Writer
}

func newCSVWriter(ch chan map[string]string, wg *sync.WaitGroup, file io.Writer) csvWriter {
	return csvWriter{ch, wg, file}
}

func (c csvWriter) WriteResults(fieldOrder []string) {
	w := csv.NewWriter(c.file)
	defer w.Flush()

	if err := w.Write(fieldOrder); err != nil {
		log.Fatalln("error writing record to csv:", err)
	}
	for result := range c.ch {
		row := make([]string, len(fieldOrder))
		for i, key := range fieldOrder {
			row[i] = result[key]
		}
		if err := w.Write(row); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}
}
