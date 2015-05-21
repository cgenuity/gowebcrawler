package gowebcrawler

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
)

const (
	BasePath = "./test_data/"
)

// Test server that fetches pages from a local directory
func createTestServer() *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, e := os.Open(path.Join(BasePath, r.URL.Path))

		if e != nil {
			//404 when file doesn't exist or other error
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "Not Found")
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, f)
		}
	}))

	return ts
}

func jsonToMap(j []byte) map[string]interface{} {
	var f interface{}
	json.Unmarshal(j, &f)
	return f.(map[string]interface{})
}

func getCrawler(rootUrl string) WebCrawler {
	crawler := WebCrawler{
		Parser:      &UrlParser{},
		ParsedPages: make(map[string]*Page),
		RootUrl:     rootUrl,
	}

	return crawler
}

func TestCrawlOnePageNoLinksOrAssets(t *testing.T) {
	ts := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/example.com.html"
	j, err := crawler.Crawl("/example.com.html")

	assert.Nil(t, err, "Got an error from Crawl")
	m := jsonToMap(j)

	expectedUrl := fmt.Sprint(ts.URL, path)
	assert.Equal(t, expectedUrl, m["Url"], "Did not get the expected URL")
	assert.Len(t, m["Links"], 0, "Links length is not empty")
	assert.Nil(t, m["Assets"], "Assets is not nil")
}
