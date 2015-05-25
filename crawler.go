// gowebcrawler is a concurrent Web Crawler that generates a JSON sitemap for a given root URL
package gowebcrawler

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"strings"
)

// A Page represents a web page's relation to other pages and the
// data needed to make a site map showing assets it depends on
type Page struct {
	Url      string
	Assets   []string
	Links    []string
	Children map[string]*Page
	parent   *Page
}

type Parser interface {
	Parse(string) (links []string, assets []string, err error)
}

// UrlParser implements Parser to extract relevant data from a page at a given URL
type UrlParser struct{}

type Crawler interface {
	Crawl(string, parser Parser) ([]byte, error)
}

// WebCrawler implements Crawler and generates a JSON site map from
// a starting domain and path. It takes care to not crawl other domains or
// get the same page more than once. Also supports a FetchLimit to limit
// total fetches made.
type WebCrawler struct {
	Parser     *UrlParser
	RootUrl    string
	FetchLimit int
}

type PageMessage struct {
	Page  *Page
	Error error
	Url   string
}

// Starts crawling from a given URL or path.
func (w WebCrawler) Crawl(url string) ([]byte, error) {
	c := make(chan *PageMessage)

	// Make a slice of errors to append errors to
	// TODO: Make use of these or get rid of them
	var errors []error

	url = getAbsoluteUrl(w.RootUrl, url)
	page, err := w.fetchPage(nil, url)

	if err != nil {
		return nil, fmt.Errorf("%v: %v", err, url)
	}

	// Mark root url as requested and set the root page
	requestedUrls := make(map[string]bool)
	requestedUrls[url] = true
	rootPage := page

	go func() {
		c <- &PageMessage{Page: page, Url: url}
	}()

	for waiting := 1; waiting > 0; waiting-- {
		pageMsg := <-c

		if pageMsg.Error != nil {
			errors = append(errors, fmt.Errorf("%v: %v", pageMsg.Error, pageMsg.Url))
			continue
		}

		page := pageMsg.Page

		if page.parent != nil {
			page.parent.Children[page.Url] = page
		}

		// We've hit the fetch limit, don't fetch any more but finish processing the ones in flight
		if w.FetchLimit != 0 && len(requestedUrls) >= w.FetchLimit {
			continue
		}

		// Fetch pages in goroutines without repeating any
		for _, l := range page.Links {
			l = getAbsoluteUrl(w.RootUrl, l)
			if requestedUrls[l] != true {
				// Mark as requested, and let the loop know to wait for one more
				requestedUrls[l] = true
				waiting++
				go func(link string) {
					result, err := w.fetchPage(page, link)
					c <- &PageMessage{Page: result, Error: err, Url: link}
				}(l)
			}
		}
	}

	b, jErr := json.MarshalIndent(rootPage, "", "  ")
	if jErr != nil {
		return nil, fmt.Errorf("Error generating JSON Site Map: %s", jErr)
	}

	return b, nil
}

func getAbsoluteUrl(rootUrl string, url string) string {
	if strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "//") {
		return fmt.Sprint(rootUrl, url)
	}
	return url
}

// Fetches a page from it's parent and an absolute URL
func (w WebCrawler) fetchPage(parent *Page, url string) (*Page, error) {
	if !strings.HasPrefix(url, w.RootUrl) {
		return nil, fmt.Errorf("%s", "Url invalid or outside of allowed domain")
	}

	links, assets, err := w.Parser.Parse(url)
	if err != nil {
		return nil, err
	}

	page := Page{
		Url:      url,
		Assets:   assets,
		Links:    links,
		Children: make(map[string]*Page),
		parent:   parent,
	}

	return &page, nil
}

// Gets slices of links and assets from a goquery.Document
func GetAttributesFromDocument(doc *goquery.Document) (links []string, assets []string) {
	// Links
	links = doc.Find("a[href]").Map(func(_ int, s *goquery.Selection) string {
		href, _ := s.Attr("href")
		return href
	})

	// CSS and other "link" elements
	assets = doc.Find("link[href]").Map(func(i int, s *goquery.Selection) string {
		href, _ := s.Attr("href")
		return href
	})

	//Anything with the "src" attribute (media or scripts)
	assets = append(
		assets,
		doc.Find("[src]").Map(func(i int, s *goquery.Selection) string {
			src, _ := s.Attr("src")
			return src
		})...)

	return links, assets
}

// Grabs links and assets from a page at a URL
func (u UrlParser) Parse(url string) (links []string, assets []string, err error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("Got a %d status code when getting URL [%s]", res.StatusCode, url)
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return nil, nil, err
	}

	links, assets = GetAttributesFromDocument(doc)
	return links, assets, nil
}
