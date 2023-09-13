package main

import (
	"bytes"
	"context"
	"fmt"

	// "fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/projectdiscovery/dnsx/libs/dnsx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Subdomain struct {
	URL      string
	Resolved bool
}

func getSubdomains(url string) ([]string, error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	bodyStr := string(body)
	subdomains := strings.Fields(bodyStr)
	return subdomains, nil
}

func getAllRecords(collection *mongo.Collection) ([]Subdomain, error) {

	var subdomains []Subdomain

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var subdomain Subdomain
		if err := cursor.Decode(&subdomain); err != nil {
			return nil, err
		}
		subdomains = append(subdomains, subdomain)
	}
	return subdomains, nil
}

func subfinder(domain string) (string, error) {
	cmd := exec.Command("subfinder", "-d", domain)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func amass(domain string) (string, error) {
	cmd := exec.Command("amass", "enum", "-passive", "-d", domain)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func resolver(subdomain string) bool {
	dnsClient, err := dnsx.New(dnsx.DefaultOptions)
	if err != nil {
		log.Fatal(err)
	}
	_, err = dnsClient.Lookup(subdomain)
	return err == nil
}

func main() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("testdb").Collection("subdomains")

	subdomainsURL := "https://h3llfir3.xyz/domains.txt"
	subdomains, err := getSubdomains(subdomainsURL)
	if err != nil {
		log.Fatal(err)
	}

	var documents []interface{}

	existingRecords, err := getAllRecords(collection)
	if err != nil {
		log.Fatal(err)
	}

	existingURLs := make(map[string]bool)
	for _, record := range existingRecords {
		existingURLs[record.URL] = true
	}

	for _, subdomain := range subdomains {
		subdomain = strings.TrimSpace(subdomain)
		if subdomain == "" {
			continue
		}

		fmt.Println("Processing subdomain:", subdomain)

		amassData, err := amass(subdomain)
		if err != nil {
			log.Println("Amass:", err)
		}

		subfinderData, err := subfinder(subdomain)
		if err != nil {
			log.Println("Subfinder:", err)
		}

		combinedData := amassData + subfinderData
		subdomains := strings.Split(combinedData, "\n")

		for _, subdomain := range subdomains {
			subdomain = strings.TrimSpace(subdomain)
			if subdomain == "" {
				continue
			}

			if !existingURLs[subdomain] {
				data := Subdomain{
					URL:      subdomain,
					Resolved: resolver(subdomain),
				}

				documents = append(documents, data)
				existingURLs[subdomain] = true

				fmt.Println("Inserted subdomain:", subdomain)
			}
		}
	}

	// Insert all documents at once
	if len(documents) > 0 {
		insertResult, err := collection.InsertMany(context.Background(), documents)
		if err != nil {
			log.Println("MongoDB InsertMany:", err)
		} else {
			fmt.Println("Inserted", len(insertResult.InsertedIDs), "documents")
		}
	}
}
