package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/config"
	"github.com/vildapavlicek/GoLang/youtubeCrawler/models"
)

// Manager manages data storing
type Manager struct {
	StorePipe        chan models.NextLink // chan to receive data to store from
	StoreDestination Storer               // destination where to store data, DB or file
	Shutdown         chan bool
	log              *logrus.Logger
}

type Storer interface {
	Store(link models.NextLink) error
	Close()
}

// DbStore holds DB configuration
type DbStore struct {
	User               string
	Pwd                string
	DbURL              string
	DbName             string
	DbPool             *sql.DB
	insertYoutubeLinks *sql.Stmt
	log                *logrus.Logger
}

type FileStore struct {
	destFile *os.File
	log      *logrus.Logger
}

// New returns new *Manager
func New(config config.StoreConfig, log *logrus.Logger) *Manager {
	storeDestination, err := decideStoreTarget(config, log)
	if err != nil {
		fmt.Printf("Failed to resolve store destination. Reason: %s", err)
		panic(err)
	} else {
		return &Manager{
			StorePipe:        make(chan models.NextLink, 500),
			StoreDestination: storeDestination,
			Shutdown:         make(chan bool, 1),
			log:              log,
		}
	}

}

// Decides target to store data to. If opening connection to DB fails, saves data to file links.dat
func decideStoreTarget(c config.StoreConfig, log *logrus.Logger) (Storer, error) {
	db := DbStore{
		User:   c.DbUser,
		Pwd:    c.DbPwd,
		DbURL:  "tcp(" + c.DbURL + ")",
		DbName: c.DbName,
		log:    log,
	}

	err := db.OpenConnection()

	if err == nil {
		fmt.Println("Connected to DB")
		log.WithFields(logrus.Fields{
			"method": "decideStoreTarget",
			"user":   db.User,
			"DBURL":  db.DbURL,
			"DBName": db.DbName,
		}).Debug("Connected to DB!")

		db.insertYoutubeLinks, err = db.DbPool.Prepare("insert into testdb.links (id, title, link, link_id, number) values (0,?,?,?,?)")
		if err != nil {
			fmt.Printf("Failed to prepare stmt %s", err)
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Warn("Failed to prepare insert statement")

		}
		return db, nil
	} else {
		fmt.Printf("Connection to DB failed, reason '%s'\n", err)

		log.WithFields(logrus.Fields{
			"method": "decideStoreTarget",
			"DBUrl":  db.DbURL,
			"DBName": db.DbName,
			"err":    err.Error(),
		}).Warn("Failed to connect to DB!")
	}

	file, err := os.Create(c.FilePath)

	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Warn("Failed to create file for data storing")

		return nil, err
	}

	path, err := filepath.Abs(filepath.Dir(file.Name()))
	fmt.Printf("Created file at '%v'\n", path)
	log.WithFields(logrus.Fields{
		"path": path,
	}).Trace("Created file at path")

	return FileStore{destFile: file, log: log}, nil
}

// OpenConnection opens connection to db
func (db *DbStore) OpenConnection() error {
	var err error
	connectionString := db.User + ":" + db.Pwd + "@" + db.DbURL + "/" + db.DbName
	db.DbPool, err = sql.Open("mysql", connectionString)
	if err != nil {
		return err
	}
	if err = db.DbPool.Ping(); err != nil {
		return err
	}
	return nil

}

//Store stores data to DB
func (db DbStore) Store(link models.NextLink) error {
	_, err := db.insertYoutubeLinks.Exec(link.Title, link.Link, link.ID, link.Number)

	if err != nil {
		log.Printf("Insert failed: %s", err)
		db.log.WithFields(logrus.Fields{
			"err":            err.Error(),
			"nextLinkID":     link.ID,
			"nextLinkTitle":  link.Title,
			"nextLinkLink":   link.Link,
			"nextLinkNumber": link.Number,
		}).Warn("Failed to insert data to DB")
	}
	return nil
}

func (db DbStore) Close() {
	db.insertYoutubeLinks.Close()
	db.DbPool.Close()

}

//Store store data to file
func (f FileStore) Store(link models.NextLink) error {
	s := "[ID: '" + link.ID + "', Link: '" + link.Link + "', Title: '" + link.Title + "', no.: '" + strconv.Itoa(link.Number) + "']\n"
	_, err := f.destFile.Write([]byte(s))
	if err != nil {
		return err
	}
	return nil
}

// StoreData stores data to configured destination
func (m *Manager) StoreData() {

	for {
		select {
		case data, ok := <-m.StorePipe:
			if !ok {
				fmt.Println("Store channel closed, shutting down")
				m.log.Info("storePipe chan closed, shutting down")
				m.StoreDestination.Close()
				m.Shutdown <- true
				close(m.Shutdown)
				return
			}
			err := m.StoreDestination.Store(data)
			if err != nil {
				fmt.Printf("Failed to store data [ID: %v], iteration %v, reason: %s", data.ID, data.Number, err)
				m.log.WithFields(logrus.Fields{
					"err":            err.Error(),
					"nextLinkID":     data.ID,
					"nextLinkTitle":  data.Title,
					"nextLinkLink":   data.Link,
					"nextLinkNumber": data.Number,
				}).Fatal("Failed to store data")
			}
		default:

		}

	}
	fmt.Println("Storing data finished")
}

func (f FileStore) Close() {
	f.destFile.Close()
}
