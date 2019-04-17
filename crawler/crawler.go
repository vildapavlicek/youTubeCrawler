package crawler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/config"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/models"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/parsers"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/store"
	"golang.org/x/net/publicsuffix"
)

var cjar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

//myClient is a custom http client
var myClient = &http.Client{
	Timeout: 30 * time.Second,
	Jar:     cjar,
	Transport: &http.Transport{
		MaxIdleConns:    15,
		IdleConnTimeout: 30 * time.Second,
	},
}

// Crawler struct holds all data needed for crawling
type Crawler struct {
	data          chan models.NextLink //chan used for crawling
	stopSignal    chan bool            //chan to stop all crawling threads
	wg            sync.WaitGroup       //crawling threads waitGroup
	StoreManager  *store.Manager       // manager for data storing
	Configuration config.CrawlerConfig
	parser        parsers.DataParser
	printTarget   io.Writer //used to set output for message printing (not logging)
	log           *logrus.Logger
}

// New returns *Crawler
func New(storeManager *store.Manager, config config.CrawlerConfig, parser parsers.DataParser, output io.Writer, log *logrus.Logger) *Crawler {

	return &Crawler{
		data:          make(chan models.NextLink, 500),
		wg:            sync.WaitGroup{},
		stopSignal:    make(chan bool, config.NumOfGoroutines),
		StoreManager:  storeManager,
		Configuration: config,
		parser:        parser,
		printTarget:   output,
		log:           log,
	}
}

// GetHTTPRequest returns *Request to do Do method with
func (c *Crawler) getHTTPRequest(method, uri string) (*http.Request, error) {
	httpMethod := method
	req, err := http.NewRequest(httpMethod, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/html; charset=utf-8")
	return req, nil
}

// getResponse does GET request to specified URI
func (c *Crawler) getResponse(httpMethod, baseURL, urlSuffix string, customHTTPClient *http.Client) (res *http.Response, err error) {
	uri := baseURL + urlSuffix
	req, err := c.getHTTPRequest(httpMethod, uri)
	if err != nil {
		c.log.WithFields(logrus.Fields{
			"method:": "getHTTPRequest",
			"err":     err.Error(),
		}).Fatal("Failed to get request")
	}

	youTube, err := url.Parse(baseURL)
	if err != nil {
		c.log.WithFields(logrus.Fields{
			"method": "url.Parse",
			"value":  baseURL,
			"err":    err.Error(),
		}).Warn("Failed to parse baseURL, wont set cookies")
	}

	for _, v := range myClient.Jar.Cookies(youTube) {
		req.AddCookie(v)
	}

	res, err = customHTTPClient.Do(req)
	if err != nil {
		c.log.WithFields(logrus.Fields{
			"requestURI": req.URL.String(),
			"method":     "customHTTPClient.Do",
			"value":      "req",
			"err":        err.Error(),
		}).Error("Failed to get response")
		return nil, err
	}

	if res.StatusCode == http.StatusOK {
		c.log.WithFields(logrus.Fields{
			"requestURI":     req.URL.String(),
			"responseStatus": res.Status,
		}).Debug("Received response 200 OK")
	} else {
		c.log.WithFields(logrus.Fields{
			"requestURI":     req.URL.String(),
			"responseStatus": res.Status,
		}).Warn("ResponseCode <> 200 OK")
		return nil, errors.New("Failed to get response 200 OK, received " + res.Status)
	}

	return res, nil
}

// Crawl crawls through youTube
// takes data from Crawler.Data chan in form of nextLink struct
// checks if enough iterations has been done
// sends copy to Crawler.StoreManager.StorePipe to store data
// calls getResponse to get *http.Body used to call parseNextVideoData to get urlSuffix and title
// makes new NextLink struct and sends it to Crawler.Data chan to keep crawling
// if receives stopSignal, crawling for that given thread stops
func (c *Crawler) crawl(id int) {
	var title string
	var urlSuffix string
	for {
		select {
		case nextLink := <-c.data:
			fmt.Fprintf(c.printTarget, "Thread ID-%v Got Link from channel: [ID: %v], [title: %s], [link: %s], [number: %v]\n", id, nextLink.ID, nextLink.Title, nextLink.Link, nextLink.Number)
			c.log.WithFields(logrus.Fields{
				"threadID":       id,
				"nextLinkID":     nextLink.ID,
				"nextLinkTitle":  nextLink.Title,
				"nextLinkLink":   nextLink.Link,
				"nextLinkNumber": nextLink.Number,
			}).Trace("Got nextLink from Chan")

			if nextLink.Number > nextLink.NOfIterations {
				fmt.Fprintf(c.printTarget, "Stopped crawling for [ID: %v]; reached max iteration '%v' of '%v' on thread ID-%v\n", nextLink.ID, nextLink.Number, nextLink.NOfIterations, id)
				c.log.WithFields(logrus.Fields{
					"threadID":              id,
					"nextLinkID":            nextLink.ID,
					"nextLinkNumber":        nextLink.Number,
					"nextLinkNofIterations": nextLink.NOfIterations,
				}).Debug("Finished crawling")
				break
			}

			c.StoreManager.StorePipe <- nextLink

			res, err := c.getResponse("GET", nextLink.BaseURL, nextLink.Link, myClient)

			if err != nil {
				c.log.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Fatal("Failed to get correct response")
			}

			title, urlSuffix, err = c.parser.ParseData(res)
			res.Body.Close()
			if err != nil {
				fmt.Fprintf(c.printTarget, "Failed parseNextVideoData, reason: %s\n", err)
				c.log.WithFields(logrus.Fields{
					"method": "parser.ParseData",
					"err":    err.Error(),
				}).Fatal("Failed to parseData from response")
			}

			c.data <- models.NextLink{ID: nextLink.ID, NOfIterations: nextLink.NOfIterations, Title: title, Link: urlSuffix, Number: nextLink.Number + 1, BaseURL: nextLink.BaseURL}

		case <-c.stopSignal:
			c.wg.Done()
			fmt.Fprintf(c.printTarget, "Thread ID-%v received stop signal and stopped\n", id)
			c.log.WithFields(logrus.Fields{
				"threadID": id,
			}).Trace("Thread received stop signal and stopped")
			return
		default:

		}
	}
}

// Run starts crawling
func (c *Crawler) Run() {
	c.wg.Add(c.Configuration.NumOfGoroutines)

	for i := 0; i < c.Configuration.NumOfGoroutines; i++ {
		fmt.Fprintf(c.printTarget, "Starting routine no. %v\n", i+1)
		c.log.WithFields(logrus.Fields{
			"threadID": i + 1,
		}).Debug("Starting Go routine")
		go c.crawl(i)
	}
	go c.StoreManager.StoreData()

	c.wg.Wait()

	close(c.data)
	fmt.Fprintf(c.printTarget, "c.data Closed\n")
	c.log.Debug("c.Data chan closed")

	close(c.StoreManager.StorePipe)
	fmt.Fprintf(c.printTarget, "c.StoreManager.StorePipe closed\n")
	c.log.Debug("c.StoreManager.StorePipe chan closed")

	fmt.Fprintf(c.printTarget, "All channels closed\n")
	c.log.Info("All channels closed")
}

// Stop stops all crawling threads
func (c *Crawler) Stop() {
	for i := 0; i < c.Configuration.NumOfGoroutines; i++ {
		fmt.Fprintf(c.printTarget, "Sending stop signal to thread ID-%v\n", i)
		c.log.WithFields(logrus.Fields{
			"threadID": i,
		}).Trace("Sending stop signal to thread")
		c.stopSignal <- true
	}
}

//Add link to the Crawler.Data chan to crawl
func (c *Crawler) Add(firstLink models.NextLink) {
	c.data <- firstLink
}

/*
COOKIES:
YSC:"kF2TfpztVpw"
CreationTime:"Sat, 13 Apr 2019 10:32:41 GMT"
Domain:".youtube.com"
Expires:"Session"
HostOnly:false
HttpOnly:true
LastAccessed:"Sat, 13 Apr 2019 10:33:53 GMT"
Path:"/"
Secure:false
sameSite:"Unset"

VISITOR_INFO1_LIVE:"NBp52m1jYME"
CreationTime:"Sat, 13 Apr 2019 10:32:41 GMT"
Domain:".youtube.com"
Expires:"Thu, 10 Oct 2019 10:32:43 GMT"
HostOnly:false
HttpOnly:true
LastAccessed:"Sat, 13 Apr 2019 10:33:53 GMT"
Path:"/"
Secure:false
sameSite:"Unset"

*/
