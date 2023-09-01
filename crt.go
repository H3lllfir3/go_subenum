package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	resp, err := http.Get(
		fmt.Sprintf("https://crt.sh/?q=%s&output=json", "sokanacademy.com"),
	)
	if err != nil {
		log.Fatal(err)
	}

}
