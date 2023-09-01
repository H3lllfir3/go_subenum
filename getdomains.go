package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func getSubdomains(url string) ([]string, error) {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
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

func main() {
	fmt.Println("geting subdomains started...")

	url := "https://h3llfir3.xyz/domains.txt"
	subdomains, err := getSubdomains(url)

	if err != nil {
		log.Fatal(err)
	}
	for i, subdomain := range subdomains {
		fmt.Println(i, subdomain)
	}
}
