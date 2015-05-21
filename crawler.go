// gowebcrawler is a concurrent Web Crawler that generates a JSON sitemap for a given root URL
package gowebcrawler

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

// A Page represents a web page's relation to other pages and the
// data needed to make a site map showing assets it depends on
type Page struct {
	Url      string
	Assets   []string
	Children map[string]*Page `json:"Links"`
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
// a starting domain. It takes care to not crawl other domains or
// get the same pages multiple times.
type WebCrawler struct {
	Parser      *UrlParser
	ParsedPages map[string]*Page
	RootUrl     string
	rootPage    *Page
}

// Starts crawling from a given URL
func (w WebCrawler) Crawl(url string) ([]byte, error) {
	pages := make(chan *Page)
	done := make(chan bool)

	go w.crawlWorker(nil, url, pages, done)

	for {
		select {
		case page := <-pages:
			w.ParsedPages[page.Url] = page

			if page.parent == nil {
				w.rootPage = page
			} else {
				page.parent.Children[page.Url] = page
			}
		case <-done:
			b, err := json.MarshalIndent(w.rootPage, "", "  ")
			if err == nil {
				return b, nil
			} else {
				return nil, fmt.Errorf("Error generating JSON Site Map: %s\n", err)
			}
		}
	}
}

// Helper function for Crawl that calls itself recursively
func (w WebCrawler) crawlWorker(parent *Page, link string, pages chan *Page, done chan bool) {
	// Fix links that start with "/", but not with "//"
	if strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "//") {
		link = fmt.Sprint(w.RootUrl, link)
	}

	// Link invalid or outside of allowed domain
	if !strings.HasPrefix(link, w.RootUrl) {
		return
	}

	if w.ParsedPages[link] != nil {
		return
	}

	links, assets, err := w.Parser.Parse(link)

	if err != nil {
		fmt.Printf("Error parsing resource at URL [ %s ]: err\n", link, err)

		if parent == nil {
			// Finish only if there is an error with the root page
			done <- true
		}
		return
	}

	page := Page{
		Url:      link,
		Assets:   assets,
		Children: make(map[string]*Page),
		parent:   parent,
	}

	pages <- &page

	go func() {
		for _, l := range links {
			w.crawlWorker(&page, l, pages, done)
		}
		done <- true
	}()

}

// Gets links and assets from a goquery Document
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

	//Anything with the "src" attribute (media or js)
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
	doc, err := goquery.NewDocument(url)

	if err != nil {
		return nil, nil, fmt.Errorf("Error generating document from url [ %s ]: %v", url, err)
	}

	links, assets = GetAttributesFromDocument(doc)

	return links, assets, nil
}
