package main

import (
	"flag"
	"fmt"
	"github.com/cgenuity/gowebcrawler"
)

func main() {
	var rootUrl = flag.String("rootUrl", "https://www.golang.org", "Root Url for crawling")
	var rootPath = flag.String("path", "/", "Path after Root Url to start the crawl")
	flag.Parse()

	parser := gowebcrawler.UrlParser{}

	crawler := gowebcrawler.WebCrawler{
		Parser:      &parser,
		ParsedPages: make(map[string]*gowebcrawler.Page),
		RootUrl:     *rootUrl,
	}

	json, err := crawler.Crawl(*rootPath)
	if err != nil {
		fmt.Println("Crawl error: ", err)
	} else {
		fmt.Println(string(json))
	}
}
