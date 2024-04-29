package logs

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

// NewClient returns a new client to connect to elasticsearch
func NewClient() *elasticsearch.Client {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	password := os.Getenv("ELASTIC_PASSWORD")
	cfg := elasticsearch.Config{
		Addresses: []string{
			"https://localhost:9200",
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Username: "k8s",
		Password: password,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Println(err)
	}
	return es
}

// IndexEvent indexes a document by creating an elastic client
func IndexEvent(client elasticsearch.Client, username string, message string, eventType string) {
	document := types.EventLog{
		Username:  username,
		Message:   message,
		Timestamp: time.Now(),
		Type:      eventType,
	}
	jsonTest, _ := json.Marshal(document)
	res, err := client.Index("twitch", strings.NewReader(string(jsonTest)))
	if err != nil {
		log.Println(err)
	}
	if res.StatusCode != 201 {
		log.Println("Error indexing document")
	}
}