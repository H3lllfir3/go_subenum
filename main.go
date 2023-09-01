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
	// insertData()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Access a collection
	collection := client.Database("testdb").Collection("subdomains")

	// fetch subdomains
	fmt.Println("Fetching subdomains ...")
	subdomainsUrl := "https://h3llfir3.xyz/domains.txt"
	subdomains, err := getSubdomains(subdomainsUrl)
	if err != nil {
		log.Fatal(err)
	}

	for _, subdomain := range subdomains {
		if subdomain != "" {
			fmt.Println("Processing subdomain: ", subdomain)

			amassData, err := amass(subdomain)
			if err != nil {
				log.Println("Amass: ", err)
			}

			subfinderData, err := subfinder(subdomain)
			if err != nil {
				log.Println("Subfinder:", err)
			}
			combinedData := amassData + subfinderData

			subdomains := strings.Split(combinedData, "\n")
			for _, subdomain := range subdomains {
				if subdomain != "" {
					data := Subdomain{
						URL:      subdomain,
						Resolved: resolver(subdomain),
					}

					_, err := collection.InsertOne(context.Background(), data)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}
