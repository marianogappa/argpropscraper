package main

import (
	"os"
	"strings"
	"sync"
)

func main() {
	var (
		hoodsToScrape = strings.Split(os.Args[1], ",")
		ch            = make(chan map[string]string)
		wg            = &sync.WaitGroup{}
		csvFieldOrder = []string{
			"image1", "price", "expenses", "hood", "address", "title", "details1", "details2", "details3",
			"details4", "details5", "url", "image2", "image3", "image4", "image5", "image6", "image7", "image8", "image9", "content"}
		argenpropHoodToURL = map[string]string{
			"Almagro":       "https://www.argenprop.com/departamento-alquiler-barrio-almagro%v",
			"San Cristobal": "https://www.argenprop.com/departamento-alquiler-barrio-san-cristobal%v",
		}
		zonapropHoodToURL = map[string]string{
			"Almagro":       "https://www.zonaprop.com.ar/departamentos-alquiler-almagro%v.html",
			"San Cristobal": "https://www.zonaprop.com.ar/departamentos-alquiler-san-cristobal%v.html",
		}
	)

	go newArgenpropScraper(ch, wg, argenpropHoodToURL).Scrape(hoodsToScrape)
	go newZonapropScraper(ch, wg, zonapropHoodToURL).Scrape(hoodsToScrape)
	go closeChannelWhenScrapersAreDone(ch, wg)

	newCSVWriter(ch, wg, os.Stdout).WriteResults(csvFieldOrder)
}

func closeChannelWhenScrapersAreDone(ch chan map[string]string, wg *sync.WaitGroup) {
	// Technically, putting Add here is a race condition. You'd expect scrapers will take time to finish, though.
	// The correct place for Add is right before the first go statement.
	wg.Add(2)
	wg.Wait()
	close(ch)
}
