package config

import (
	"fmt"
	"os"
	"strconv"
)

const defaultNoOfGoroutines = 5
const defaultNoOfCrawlsPerLink = 10
const defaultDbUser = "guest"
const defaultDbPwd = ""
const defaultDbURL = "127.0.0.1:3306"
const defaultDbName = ""
const defaultFilePath = "defaultFile.dat"

//Config main config struct
type Config struct {
	CrawlerConfig CrawlerConfig
	StoreConfig   StoreConfig
}

//CrawlerConfig crawler config struct
type CrawlerConfig struct {
	NumOfGoroutines int
	NumOfCrawls     int
}

//StoreConfig configuration for data storing, db connection settings, file path
type StoreConfig struct {
	DbUser   string
	DbPwd    string
	DbURL    string
	DbName   string
	FilePath string
}

// New returns pointer to new config struct
func New() *Config {
	return &Config{
		CrawlerConfig: CrawlerConfig{
			NumOfGoroutines: getEnvAsInt("GOROUTINES", defaultNoOfGoroutines),
			NumOfCrawls:     getEnvAsInt("NUMOFCRAWLS", defaultNoOfCrawlsPerLink),
		},
		StoreConfig: StoreConfig{
			DbUser:   getEnv("DBUSER", defaultDbUser),
			DbPwd:    getEnv("DBPWD", defaultDbPwd),
			DbURL:    getEnv("DBURL", defaultDbURL),
			DbName:   getEnv("DBNAME", defaultDbName),
			FilePath: getEnv("FILESTORE", defaultFilePath),
		},
	}
}

// looks up environment by name, returns default value if not found
func getEnv(envName string, defaultValue string) string {
	value, exists := os.LookupEnv(envName)
	if !exists {
		fmt.Printf("Didn't find env '%s'. Setting default value '%v'\n", envName, defaultValue)
		return defaultValue
	}
	return value
}

// looks up environment by name and converts it to string, if not found returns default value
func getEnvAsInt(envName string, defaultValue int) int {
	value := getEnv(envName, "")
	if value == "" {
		fmt.Printf("Env \"%s\" not found. Setting default value '%v\n'", envName, defaultValue)
		return defaultValue
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		fmt.Printf("Failed to convert env \"%s\" value '%v' to int. Setting default value '%v'\n", envName, value, defaultValue)
		return defaultValue
	}

	return n
}
