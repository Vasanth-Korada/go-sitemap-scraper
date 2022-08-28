package main

import (
	"log"
	"math/rand"
	"net/http"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Vasanth-Korada/sitemap-crawler/helpers"
	"github.com/Vasanth-Korada/sitemap-crawler/models"
	"github.com/joho/godotenv"
)

var userAgents []string = helpers.GetUserAgents()

/*DefaultParser is en empty struct for implmenting default parser*/
type DefaultParser struct {
}

/*Parser defines the parsing interface*/
type Parser interface {
	GetSEOData(resp *http.Response) (models.SEOData, error)
}

/*Function which return a radom user agent*/
func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}

/*Http helper function to make network request*/
func makeRequest(url string) (*http.Response, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetSeoData concrete implementation of the default parser
func (d DefaultParser) GetSEOData(resp *http.Response) (models.SEOData, error) {
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return models.SEOData{}, err
	}
	seoData := models.SEOData{}
	seoData.URL = resp.Request.URL.String()
	seoData.StatusCode = resp.StatusCode
	seoData.Title = doc.Find("title").First().Text()
	seoData.H1 = doc.Find("h1").First().Text()
	seoData.MetaDescription, _ = doc.Find("meta[name^=description]").Attr("content")

	return seoData, nil
}

/*Crawls a given page*/
func crawlPage(url string, tokens chan struct{}) (*http.Response, error) {
	tokens <- struct{}{}
	resp, err := makeRequest(url)
	<-tokens
	if err != nil {
		return nil, err
	}
	return resp, err
}

/*Scraps a given page*/
func scrapePage(url string, token chan struct{}, parser Parser) (models.SEOData, error) {
	response, err := crawlPage(url, token)
	if err != nil {
		return models.SEOData{}, err
	}
	seoData, err := parser.GetSEOData(response)
	if err != nil {
		return models.SEOData{}, err
	}
	return seoData, nil
}

/*Function to extract urls in a sitemap response document (xml)*/
func extractUrls(response *http.Response) ([]string, error) {
	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}
	extractedUrls := []string{}
	sel := doc.Find("loc")
	for i := range sel.Nodes {
		loc := sel.Eq(i)
		url := loc.Text()
		extractedUrls = append(extractedUrls, url)
	}
	return extractedUrls, nil
}

/*Function to recursively extract urls in the given sitemap*/
func extractSiteMapURLs(startURL string) []string {
	workList := make(chan []string)
	toCrawl := []string{}
	var itr int
	itr++

	go func() { workList <- []string{startURL} }()

	for ; itr > 0; itr-- {
		list := <-workList
		for _, link := range list {
			itr++
			go func(link string) {
				response, err := makeRequest(link)
				if err != nil {
					log.Printf("Error retrieving URL:%s", link)
				}
				urls, _ := extractUrls(response)
				if err != nil {
					log.Printf("Error extracting document form response, URL:%s", link)
				}
				siteMapFiles, pages := helpers.IsSitemap(urls)
				if siteMapFiles != nil {
					workList <- siteMapFiles
				}
				toCrawl = append(toCrawl, pages...)
			}(link)
		}
	}
	return toCrawl
}

/*Function to scrape the given urls*/
func scrapeURLs(urls []string, parser Parser, concurreny int) []models.SEOData {
	tokens := make(chan struct{}, concurreny)
	var itr int
	itr++
	workList := make(chan []string)
	scrapedData := []models.SEOData{}

	go func() { workList <- urls }()
	for ; itr > 0; itr-- {
		urlList := <-workList
		for _, url := range urlList {
			if url != "" {
				itr++
				go func(url string, token chan struct{}) {
					log.Printf("Requesting URL:%s", url)
					res, err := scrapePage(url, tokens, parser)
					if err != nil {
						log.Printf("Encountered error, URL:%s", url)
					} else {
						scrapedData = append(scrapedData, res)
					}
					workList <- []string{}
				}(url, tokens)
			}
		}
	}
	return scrapedData
}

// ScrapeSitemap scrapes a given sitemap
func scrapeSiteMap(url string, parser Parser, concurreny int) []models.SEOData {
	results := extractSiteMapURLs(url)
	scrapedDataSlice := scrapeURLs(results, parser, concurreny)
	return scrapedDataSlice
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	parser := DefaultParser{}
	scrapedData := scrapeSiteMap("https://www.confirmtkt.com/sitemap.xml", parser, 10)
	helpers.GenerateExcelFile(scrapedData)
}
