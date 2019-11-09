package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type argenpropScraper struct {
	ch        chan map[string]string
	wg        *sync.WaitGroup
	hoodToURL map[string]string
}

func newArgenpropScraper(ch chan map[string]string, wg *sync.WaitGroup, hoodToURL map[string]string) argenpropScraper {
	return argenpropScraper{ch, wg, hoodToURL}
}

func (s argenpropScraper) Scrape(hoodsToScrape []string) {
	defer s.wg.Done()
	for _, hoodToScrape := range hoodsToScrape {
		s.scrapeHood(hoodToScrape)
	}
}

func (s argenpropScraper) scrapeHood(hoodToScrape string) {
	if _, ok := s.hoodToURL[hoodToScrape]; !ok {
		return
	}
	pageNumber := 1
	for {
		url := s.calculatePageURL(hoodToScrape, pageNumber)
		document, err := requestPage(url)
		if err != nil {
			log.Printf("argenpropScraper: stopping due to error requesting page: %v\n", err)
			break
		}

		ok := s.scrapeDocument(document, hoodToScrape)
		if !ok {
			log.Println("argenpropScraper: stopping due to no properties found in browse page; probably done?")
			break
		}
		pageNumber++
	}
}

func (s argenpropScraper) scrapeDocument(doc *goquery.Document, hoodToScrape string) bool {
	ok := false
	doc.Find(".listing__item").Each(func(index int, element *goquery.Selection) {
		result := map[string]string{}

		result["hood"] = hoodToScrape
		result["address"] = scrapeSelectorAsText(element, ".card__address")
		result["title"] = scrapeSelectorAsText(element, ".card__title")
		result["content"] = scrapeSelectorAsText(element, ".card__info")
		result["price"] = filterOnlyNumbers(scrapeSelectorAsText(element, ".card__price"))
		result["expenses"] = filterOnlyNumbers(scrapeSelectorAsText(element, ".card__expenses"))
		result["url"] = fmt.Sprintf("https://www.argenprop.com%v", scrapeSelectorAttr(element, "a", "href"))

		// Images
		imageUrls := []string{}
		element.Find(".card__photos li img").Each(func(index int, element *goquery.Selection) {
			src, exists := element.Attr("data-src")
			if exists && src != "" {
				imageUrls = append(imageUrls, src)
			}
		})
		for i := 0; i < 10; i++ {
			result[fmt.Sprintf("image%v", i+1)] = ""
			if i < len(imageUrls) {
				result[fmt.Sprintf("image%v", i+1)] = fmt.Sprintf(`=image("%v")`, imageUrls[i])
			}
		}

		// Details
		maybeDetails := strings.Split(scrapeSelectorAsText(element, ".card__common-data"), "•")
		details := []string{}
		for i := range maybeDetails {
			maybeDetail := strings.TrimSpace(maybeDetails[i])
			if maybeDetail != "" {
				details = append(details, maybeDetail)
			}
		}
		for i := 0; i < 10; i++ {
			result[fmt.Sprintf("details%v", i+1)] = ""
			if i < len(details) {
				result[fmt.Sprintf("details%v", i)] = details[i]
			}
		}

		s.ch <- result
		ok = true
	})
	return ok
}

func (s argenpropScraper) calculatePageURL(hood string, pageNumber int) string {
	if pageNumber <= 1 {
		return fmt.Sprintf(s.hoodToURL[hood], "")
	}
	return fmt.Sprintf(s.hoodToURL[hood], fmt.Sprintf("-pagina-%v", pageNumber))
}
