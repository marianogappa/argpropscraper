package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

func filterOnlyNumbers(s string) string {
	reg := regexp.MustCompile("[^0-9]+")
	return reg.ReplaceAllString(s, "")
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
