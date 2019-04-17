package crawler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vildapavlicek/GoLang/youtubeCrawler/config"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/models"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/store"
)

type countParser struct {
}

func (cp countParser) ParseData(response *http.Response) (link, title string, err error) {
	return "", "", nil
}

type fakeStore struct {
	data    []models.NextLink
	counter *int32
}

func (fs fakeStore) Store(link models.NextLink) error {
	fs.data = append(fs.data, link)
	atomic.AddInt32(fs.counter, 1)
	return nil
}

func (fs fakeStore) Close() {

}

func TestGetResponse(t *testing.T) {
	t.Run("OK Response", func(t *testing.T) {
		status := http.StatusOK
		want := http.StatusOK
		server := makeHTTPServer(status)
		defer server.Close()
		got := getResponse("GET", server.URL, "", myClient)
		defer got.Body.Close()
		assertStatusEquals(t, want, got.StatusCode)
	})
}

func TestCrawl(t *testing.T) {

	t.Run("Test 30 iterations - single thread", func(t *testing.T) {
		counter := int32(0)
		testStore := fakeStore{
			data:    make([]models.NextLink, 30),
			counter: &counter,
		}
		testStoreManager := &store.Manager{
			StorePipe:        make(chan models.NextLink, 10),
			StoreDestination: testStore,
			Shutdown:         make(chan bool, 1),
		}

		server := makeHTTPServer(200)
		defer server.Close()

		firstLink := models.NextLink{
			BaseURL:       server.URL,
			Link:          "",
			NOfIterations: 30,
			Number:        0,
		}

		cp := countParser{}

		crawler := Crawler{
			data:         make(chan models.NextLink, 5),
			parser:       cp,
			wg:           sync.WaitGroup{},
			stopSignal:   make(chan bool),
			StoreManager: testStoreManager,
			printTarget:  ioutil.Discard,
		}
		crawler.Add(firstLink)
		crawler.wg.Add(1)
		go crawler.crawl(1, crawler.parser)
		go crawler.StoreManager.StoreData()
		time.Sleep(3 * time.Second)
		crawler.Stop()

		wantIterations := int32(30)
		gotIterations := atomic.LoadInt32(testStore.counter) - 1

		assertCountEquals(t, wantIterations, gotIterations)

	})

	t.Run("Test 20 iterations - single thread", func(t *testing.T) {
		counter := int32(0)
		testStore := fakeStore{
			data:    make([]models.NextLink, 20),
			counter: &counter,
		}
		testStoreManager := &store.Manager{
			StorePipe:        make(chan models.NextLink, 10),
			StoreDestination: testStore,
			Shutdown:         make(chan bool, 1),
		}

		server := makeHTTPServer(200)
		defer server.Close()

		firstLink := models.NextLink{
			BaseURL:       server.URL,
			Link:          "",
			NOfIterations: 20,
			Number:        0,
		}

		cp := countParser{}

		crawler := Crawler{
			data:         make(chan models.NextLink, 5),
			parser:       cp,
			wg:           sync.WaitGroup{},
			stopSignal:   make(chan bool),
			StoreManager: testStoreManager,
			printTarget:  ioutil.Discard,
		}
		crawler.Add(firstLink)
		crawler.wg.Add(1)
		go crawler.crawl(1, crawler.parser)
		go crawler.StoreManager.StoreData()
		time.Sleep(3 * time.Second)
		crawler.Stop()

		wantIterations := int32(20)
		gotIterations := atomic.LoadInt32(testStore.counter) - 1

		assertCountEquals(t, wantIterations, gotIterations)

	})

	t.Run("Test 30 iterations - multiple threads", func(t *testing.T) {
		counter := int32(0)
		testStore := fakeStore{
			data:    make([]models.NextLink, 30),
			counter: &counter,
		}
		testStoreManager := &store.Manager{
			StorePipe:        make(chan models.NextLink, 10),
			StoreDestination: testStore,
			Shutdown:         make(chan bool, 1),
		}

		server := makeHTTPServer(200)
		defer server.Close()

		firstLink := models.NextLink{
			BaseURL:       server.URL,
			Link:          "",
			NOfIterations: 30,
			Number:        0,
		}

		cp := countParser{}

		crawler := Crawler{
			data:         make(chan models.NextLink, 5),
			parser:       cp,
			wg:           sync.WaitGroup{},
			stopSignal:   make(chan bool),
			StoreManager: testStoreManager,
			Configuration: config.CrawlerConfig{
				NumOfGoroutines: 5,
			},
			printTarget: ioutil.Discard,
		}

		crawler.Add(firstLink)
		crawler.Add(firstLink)
		crawler.Add(firstLink)
		crawler.Add(firstLink)
		crawler.Add(firstLink)
		crawler.wg.Add(5)

		for i := 0; i < crawler.Configuration.NumOfGoroutines; i++ {
			fmt.Fprintf(crawler.printTarget, "Starting routine no. %v\n", i+1)
			go crawler.crawl(i, crawler.parser)
		}

		go crawler.StoreManager.StoreData()
		time.Sleep(3 * time.Second)
		crawler.Stop()

		wantIterations := int32(30 * 5)
		gotIterations := atomic.LoadInt32(testStore.counter) - 5

		assertCountEquals(t, wantIterations, gotIterations)
	})
}

func TestRun(t *testing.T) {
	t.Run("Multiple Threads - 30 iterations", func(t *testing.T) {
		counter := int32(0)
		lock := &sync.Mutex{}

		testStore := fakeStore{
			data:    make([]models.NextLink, 30),
			counter: &counter,
		}
		testStoreManager := &store.Manager{
			StorePipe:        make(chan models.NextLink, 10),
			StoreDestination: testStore,
			Shutdown:         make(chan bool, 1),
		}

		server := makeHTTPServer(200)
		defer server.Close()

		firstLink := models.NextLink{
			BaseURL:       server.URL,
			Link:          "",
			NOfIterations: 30,
			Number:        0,
		}

		cp := countParser{}

		conf := config.CrawlerConfig{
			NumOfGoroutines: 5,
			NumOfCrawls:     30,
		}

		c := New(testStoreManager, conf, cp, ioutil.Discard)

		go c.Run()
		c.Add(firstLink)
		c.Add(firstLink)
		c.Add(firstLink)

		time.Sleep(3 * time.Second)

		wantIterations := int32(30 * 3)
		lock.Lock()
		gotIterations := atomic.LoadInt32(testStore.counter) - 3
		lock.Unlock()
		assertCountEquals(t, wantIterations, gotIterations)

	})

	t.Run("DataRace", func(t *testing.T) {

		counter := int32(0)

		testStore := fakeStore{
			data:    make([]models.NextLink, 30),
			counter: &counter,
		}
		testStoreManager := &store.Manager{
			StorePipe:        make(chan models.NextLink, 10),
			StoreDestination: testStore,
			Shutdown:         make(chan bool, 1),
		}

		server := makeHTTPServer(200)
		defer server.Close()

		firstLink := models.NextLink{
			BaseURL:       server.URL,
			Link:          "",
			NOfIterations: 30,
			Number:        0,
		}

		cp := countParser{}

		conf := config.CrawlerConfig{
			NumOfGoroutines: 5,
			NumOfCrawls:     30,
		}

		c := New(testStoreManager, conf, cp, ioutil.Discard)

		go c.Run()
		c.Add(firstLink)
		c.Add(firstLink)
		c.Add(firstLink)

		time.Sleep(3 * time.Second)
	})
}

func makeHTTPServer(status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)

	}))
}

func assertStatusEquals(t *testing.T, want, got int) {
	t.Helper()
	if want != got {
		t.Errorf("Got '%v', want: '%v'", got, want)
	}
}

func assertCountEquals(t *testing.T, want, got int32) {
	t.Helper()
	if want != got {
		t.Errorf("Got '%v', want: '%v'", got, want)
	}
}
