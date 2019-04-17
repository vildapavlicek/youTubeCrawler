package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/config"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/crawler"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/handlers"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/parsers"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/store"

	_ "net/http/pprof"

	"github.com/joho/godotenv"
)

var firstLink = "/watch?v=DT61L8hbbJ4"
var secondLink = "/watch?v=Q3oItpVa9fs"

var log = logrus.New()

func init() {
	file, err := os.OpenFile("logs", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	log.Out = file
	//log.SetReportCaller(true)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.TraceLevel) //change to be set by ENV

	err = godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load '.env' config file. All values will be set to default if not set as system environment variable")
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Info(".env couldn't be open")
	}
}

func main() {

	stop := make(chan os.Signal, 1)
	go catchSignal(stop)

	conf := config.New()
	m := http.NewServeMux()
	server := &http.Server{
		Addr:         ":8080",
		Handler:      m,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	storeManager := store.New(conf.StoreConfig, log)
	defer storeManager.StoreDestination.Close()

	monster := crawler.New(storeManager, conf.CrawlerConfig, parsers.YoutubeParser{Log: log}, os.Stdout, log)
	go monster.Run()

	handlers.SetHandlers(m, monster)
	go startServer(server)

	for {
		select {
		case <-storeManager.Shutdown:
			fmt.Println("Server shutting down")
			server.Shutdown(context.TODO())
			os.Exit(1)
		case <-stop:
			monster.Stop()
		default:
		}
	}
}

func startServer(s *http.Server) {
	fmt.Printf("Starting server at addr: %s", s.Addr)
	log.WithFields(logrus.Fields{
		"Addr": s.Addr,
	}).Debug("Server listening")
	err := s.ListenAndServe()
	if err != nil {
	}

}

func catchSignal(stopChan chan os.Signal) {
	signal.Notify(stopChan, os.Interrupt)
}
