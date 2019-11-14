package main

import "github.com/collinewait/gmail-scraper-go/credentials"

func main() {
	service := credentials.GetService()
	println("Service", service)
}
