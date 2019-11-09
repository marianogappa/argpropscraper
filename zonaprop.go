package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type zonapropScraper struct {
	ch        chan map[string]string
	wg        *sync.WaitGroup
	hoodToURL map[string]string
}

func newZonapropScraper(ch chan map[string]string, wg *sync.WaitGroup, hoodToURL map[string]string) zonapropScraper {
	return zonapropScraper{ch, wg, hoodToURL}
}

func (s zonapropScraper) Scrape(hoodsToScrape []string) {
	defer s.wg.Done()
	for _, hoodToScrape := range hoodsToScrape {
		s.scrapeHood(hoodToScrape)
	}
}

func (s zonapropScraper) scrapeHood(hoodToScrape string) {
	if _, ok := s.hoodToURL[hoodToScrape]; !ok {
		return
	}
	pageNumber := 1
	for {
		url := s.calculatePageURL(hoodToScrape, pageNumber)
		document, err := requestPage(url)
		if err != nil {
			log.Printf("zonapropScraper: stopping due to error requesting page: %v\n", err)
			break
		}

		// Zonaprop redirects you to the larger valid page number when you exceed it. Therefore, the naive strategy of
		// looping forever with incrementing page numbers doesn't work. The simple solution chosen here is to read the
		// active page number according to the HTML, and if it's lower than expected, to break.
		currentPageAccordingToDocument := atoi(document.Find("li.active a").Text())
		if currentPageAccordingToDocument < pageNumber {
			log.Printf("zonapropScraper: stopping due to pageNumber mismatch (probably done): %v vs %v\n", pageNumber, currentPageAccordingToDocument)
			break
		}

		ok := s.scrapeDocument(document, hoodToScrape)
		if !ok {
			log.Println("zonapropScraper: stopping due to no properties found in browse page; probably bug/throttle?")
			break
		}
		pageNumber++
	}
	return
}

func (s zonapropScraper) scrapeDocument(doc *goquery.Document, hoodToScrape string) bool {
	ok := false
	doc.Find(".general-content").Each(func(index int, element *goquery.Selection) {
		result := map[string]string{}

		result["hood"] = hoodToScrape
		result["address"] = scrapeSelectorAsText(element, ".posting-location")
		result["title"] = scrapeSelectorAsText(element, ".posting-title")
		result["content"] = scrapeSelectorAsText(element, ".posting-description")
		result["price"] = filterOnlyNumbers(scrapeSelectorAsText(element, ".first-price"))
		result["expenses"] = filterOnlyNumbers(scrapeSelectorAsText(element, ".expenses"))
		result["url"] = fmt.Sprintf("https://www.zonaprop.com%v", scrapeSelectorAttr(element, "a", "href"))

		// Images
		imageUrls := []string{}
		element.Find(".posting-gallery-slider").Each(func(index int, element *goquery.Selection) {
			html, err := element.Html()
			if err != nil || html == "" {
				return
			}
			for _, match := range regexp.MustCompile(`url730x532: '(.+)', url360x266`).FindAllStringSubmatch(html, -1) {
				if len(match) == 2 {
					imageUrls = append(imageUrls, strings.TrimSpace(match[1]))
				}
			}
		})
		for i := 0; i < 10; i++ {
			result[fmt.Sprintf("image%v", i+1)] = ""
			if i < len(imageUrls) {
				result[fmt.Sprintf("image%v", i+1)] = fmt.Sprintf(`=image("%v")`, imageUrls[i])
			}
		}

		// Details
		details := []string{}
		element.Find(".main-features li b").Each(func(index int, element *goquery.Selection) {
			maybeDetail := element.Text()
			if maybeDetail != "" {
				details = append(details, maybeDetail)
			}
		})
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

func (s zonapropScraper) calculatePageURL(hood string, pageNumber int) string {
	if pageNumber <= 1 {
		return fmt.Sprintf(s.hoodToURL[hood], "")
	}
	return fmt.Sprintf(s.hoodToURL[hood], fmt.Sprintf("-pagina-%v", pageNumber))
}
