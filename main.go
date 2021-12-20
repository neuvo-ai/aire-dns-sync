package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	// Cloudflare
	"github.com/cloudflare/cloudflare-go"

	// MongoDB
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// ENV loading
	"github.com/joho/godotenv"

	// Log
	log "github.com/sirupsen/logrus"
)

type EntryStruct struct {
	Name    string
	Content string
}

func main() {

	// Load ENV from .env file
	godotenv.Load()

	// Set the logger defaults
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}
	log.SetFormatter(&log.TextFormatter{})
	var nameValidate = regexp.MustCompile(os.Getenv("DNS_FILTER"))

	// Construct a new API object for Cloudflare
	api, err := cloudflare.NewWithAPIToken(os.Getenv("CF_API_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Most API calls require a Context
	//ctx := context.Background()

	// Fetch the zone ID
	zoneID, err := api.ZoneIDByName(os.Getenv("CF_DOMAIN"))
	if err != nil {
		log.Panic(err)
	}

	// TODO: remove these
	log.Debug("Cloudflare Zone ID: " + zoneID)

	records, err := api.DNSRecords(context.Background(), zoneID, cloudflare.DNSRecord{Type: "A"})

	// for _, record := range records {
	// 	log.Info("record")
	// //	log.Debug(fmt.Sprintf("%+v", record))
	// }

	// Connect to the databavse
	cursor := getEntryDocuments()
	for cursor.Next(context.Background()) {
		log.Debug("Decoding new DNS entry from the database")

		entry := EntryStruct{}

		err := cursor.Decode(&entry)
		if err != nil {
			log.Panic(err)
		}

		log.Debug(fmt.Sprintf("%+v", entry))
		log.Info("Verifying entries for: " + entry.Name)

		// Verify that the name matches a filer REGEX
		if !nameValidate.MatchString(entry.Name) {
			log.Error("DNS name doesn't match the filter")
		} else {

			resultIndex, findErr := findRecordIndex(entry.Name, records)

			if findErr == nil {
				// The record already exists, we just make sure it is the same one
				if records[resultIndex].Content != entry.Content {
					// The content differs, updating...
					log.Info("Updating record")
					records[resultIndex].Content = entry.Content
					api.UpdateDNSRecord(context.Background(), zoneID, records[resultIndex].ID, records[resultIndex])
				} else {
					log.Debug("The record is already up to date")
				}
			} else {
				// New record
				log.Info("Creating a new record")

				proxied := new(bool)
				*proxied = true

				record := cloudflare.DNSRecord{
					Type:    "A",
					Name:    entry.Name,
					Content: entry.Content,
					Proxied: proxied,
				}

				api.CreateDNSRecord(context.Background(), zoneID, record)
			}

		}
	}
	defer cursor.Close(context.Background())

}

func getEntryDocuments() *mongo.Cursor {
	/*
	   Connect to my cluster
	*/
	log.Info("Starting database")
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Panic(err)
	}
	defer client.Disconnect(ctx)
	defer cancel()
	database := client.Database(os.Getenv("MONGODB_DATABASE"))
	collection := database.Collection(os.Getenv(("MONGODB_COLLECTION")))
	cursor, err := collection.Find(context.Background(), bson.D{})

	if err != nil {
		log.Fatal("Failed to find entries in the collection")
	}

	return cursor
}

func findRecordIndex(name string, records []cloudflare.DNSRecord) (int, error) {
	for i := range records {
		if records[i].Name == name {
			return i, nil
		}
	}
	return -1, errors.New("record not found")
}
