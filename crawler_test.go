package gowebcrawler

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
)

const (
	BasePath = "./test_data/"
)

func TestCrawlExampleCom(t *testing.T) {
	ts, _ := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/example.com.html"
	j, err := crawler.Crawl(path)

	assert.Nil(t, err, "Got an error from Crawl")

	m := jsonToMap(j)

	expectedUrl := fmt.Sprint(ts.URL, path)
	assert.Equal(t, expectedUrl, m["Url"], "Did not get the expected URL")
	assert.Len(t, m["Links"], 1, "Links length is not 1")
	assert.Nil(t, m["Assets"], "Assets is not nil")
	assert.Len(t, m["Children"], 0, "Children is not nil")
}

func TestCrawlRootPageNotFound(t *testing.T) {
	ts, _ := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/404"
	_, err := crawler.Crawl(path)

	assert.Error(t, err, "Did not get an error")
}

func TestCrawlThreeLevels(t *testing.T) {
	ts, _ := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/three/1.html"
	j, err := crawler.Crawl(path)

	assert.Nil(t, err, "Got an error from Crawl")

	m := jsonToMap(j)

	//First level
	expectedUrl := fmt.Sprint(ts.URL, path)
	assert.Equal(t, expectedUrl, m["Url"], "Did not get the expected URL")
	oneChildren := m["Children"].(map[string]interface{})

	//Second level
	twoUrl := fmt.Sprint(ts.URL, "/three/2.html")
	two := oneChildren[twoUrl].(map[string]interface{})
	assert.Equal(t, two["Url"], twoUrl)
	twoChildren := two["Children"].(map[string]interface{})

	//Third level
	threeUrl := fmt.Sprint(ts.URL, "/three/3.html")
	three := twoChildren[threeUrl].(map[string]interface{})
	threeAssets := three["Assets"].([]interface{})
	assert.Equal(t, threeAssets[0], "theend.jpg")

}

func TestCrawlDoesntFollowExternalLinks(t *testing.T) {
	ts, _ := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/external_links.html"
	j, err := crawler.Crawl(path)

	assert.Nil(t, err, "Got an error from Crawl")

	m := jsonToMap(j)

	expectedUrl := fmt.Sprint(ts.URL, path)
	assert.Equal(t, expectedUrl, m["Url"], "Did not get the expected URL")
	assert.Len(t, m["Links"], 2, "Didn't find 2 links")
	assert.Len(t, m["Children"], 0, "Children is not nil")
}

func TestCrawlFindsAssets(t *testing.T) {
	ts, _ := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/assets.html"
	j, err := crawler.Crawl(path)

	assert.Nil(t, err, "Got an error from Crawl")

	m := jsonToMap(j)

	expectedUrl := fmt.Sprint(ts.URL, path)
	assert.Equal(t, expectedUrl, m["Url"], "Did not get the expected URL")
	assert.Len(t, m["Assets"], 3, "Didn't find 3 assets")
}

func TestCrawlDoesntRepeatRequests(t *testing.T) {
	ts, requestCount := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/circular/1.html"
	_, err := crawler.Crawl(path)

	assert.Nil(t, err, "Got an error from Crawl")

	assert.Equal(t, 3, *requestCount, "Didn't make the right amount of requests")
}

func TestCrawlRespectsFetchLimit(t *testing.T) {
	ts, requestCount := createTestServer()
	defer ts.Close()

	crawler := WebCrawler{
		Parser:     &UrlParser{},
		RootUrl:    ts.URL,
		FetchLimit: 2,
	}

	path := "/three/1.html"
	_, err := crawler.Crawl(path)

	assert.Nil(t, err, "Got an error from Crawl")

	assert.Equal(t, 2, *requestCount, "Didn't make the right amount of requests")
}

func TestCrawlDoesntIncludeInvalidLinks(t *testing.T) {
	ts, _ := createTestServer()
	defer ts.Close()

	crawler := getCrawler(ts.URL)
	path := "/invalid_links.html"
	j, err := crawler.Crawl(path)
	m := jsonToMap(j)

	assert.Nil(t, err, "Got an error from Crawl")

	assert.Nil(t, m["Links"], "Found links when it shouldn't have.")
}

// Test server that fetches pages from a local directory
func createTestServer() (*httptest.Server, *int) {
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		body, err := ioutil.ReadFile(path.Join(BasePath, r.URL.Path))

		if err != nil {
			//404 when file doesn't exist or other error
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.Write(body)
		}
	}))

	return ts, &requestCount
}

func jsonToMap(j []byte) map[string]interface{} {
	var f interface{}
	json.Unmarshal(j, &f)
	return f.(map[string]interface{})
}

func getCrawler(rootUrl string) WebCrawler {
	crawler := WebCrawler{
		Parser:  &UrlParser{},
		RootUrl: rootUrl,
	}

	return crawler
}
