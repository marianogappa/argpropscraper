package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var hoodToURL = map[string]string{
	"Almagro":       "https://www.argenprop.com/departamento-alquiler-barrio-almagro%v",
	"San Cristobal": "https://www.argenprop.com/departamento-alquiler-barrio-san-cristobal%v",
}

func calculatePageURL(hood string, pageNumber int) string {
	if pageNumber <= 1 {
		return fmt.Sprintf(hoodToURL[hood], "")
	}
	return fmt.Sprintf(hoodToURL[hood], fmt.Sprintf("-pagina-%v", pageNumber))
}

func requestPage(url string) (*goquery.Document, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Create and modify HTTP request before sending
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Not Firefox")

	// Make request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return goquery.NewDocumentFromReader(response.Body)
}

func scrapeSelectorAsText(sel *goquery.Selection, selector string) string {
	result := ""
	sel.Find(selector).Each(func(index int, element *goquery.Selection) {
		result = element.Text()
	})
	return strings.TrimSpace(result)
}

func scrapeSelectorAttr(sel *goquery.Selection, selector, attr string) string {
	result := ""
	sel.Find(selector).Each(func(index int, element *goquery.Selection) {
		res, exists := element.Attr(attr)
		if exists && res != "" {
			result = res
		}
	})
	return strings.TrimSpace(result)
}

func scrapeDocument(doc *goquery.Document, hoodToScrape string) ([]map[string]string, error) {
	results := []map[string]string{}
	doc.Find(".listing__item").Each(func(index int, element *goquery.Selection) {
		result := map[string]string{}

		result["hood"] = hoodToScrape
		result["address"] = scrapeSelectorAsText(element, ".card__address")
		result["location"] = scrapeSelectorAsText(element, ".card__location")
		result["title"] = scrapeSelectorAsText(element, ".card__title")
		result["content"] = scrapeSelectorAsText(element, ".card__info")
		result["price"] = scrapeSelectorAsText(element, ".card__price")
		result["expenses"] = strings.TrimSpace(strings.Replace(strings.Replace(scrapeSelectorAsText(element, ".card__expenses"), "+", "", -1), " expensas", "", -1))
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

		results = append(results, result)
	})
	return results, nil
}

func scrapeHood(hoodToScrape string) ([]map[string]string, error) {
	if _, ok := hoodToURL[hoodToScrape]; !ok {
		return []map[string]string{}, nil
	}
	pageNumber := 1
	properties := []map[string]string{}
	for {
		url := calculatePageURL(hoodToScrape, pageNumber)
		document, err := requestPage(url)
		if err != nil {
			break
		}

		documentProperties, err := scrapeDocument(document, hoodToScrape)
		if err != nil || len(documentProperties) == 0 {
			break
		}
		for _, documentProperty := range documentProperties {
			properties = append(properties, documentProperty)
		}
		pageNumber++
	}
	return properties, nil
}

func mustWriteResultsAsCSV(results []map[string]string, file io.Writer) {
	w := csv.NewWriter(file)
	// fields := []string{}
	// for key := range results[0] {
	// 	fields = append(fields, key)

	// }
	fields := []string{
		"image1", "price", "expenses", "hood", "address", "title", "details1", "details2", "details3",
		"details4", "details5", "url", "image2", "image3", "image4", "image5", "image6", "image7", "image8", "image9", "content"}
	if err := w.Write(fields); err != nil {
		log.Fatalln("error writing record to csv:", err)
	}
	for _, result := range results {
		row := make([]string, len(fields))
		for i, key := range fields {
			row[i] = result[key]
		}
		if err := w.Write(row); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}
	w.Flush()
}

func main() {
	hoodsToScrape := strings.Split(os.Args[1], ",")

	results := []map[string]string{}
	for _, hoodToScrape := range hoodsToScrape {
		hoodProperties, _ := scrapeHood(hoodToScrape)
		results = append(results, hoodProperties...)
	}

	if len(results) == 0 {
		log.Fatal("Couldn't scrape any data :(")
	}
	mustWriteResultsAsCSV(results, os.Stdout)
}
