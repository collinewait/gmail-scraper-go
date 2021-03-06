package main

import (
	"github.com/collinewait/gmail-scraper-go/credentials"
	"github.com/collinewait/gmail-scraper-go/scraper"
	"google.golang.org/api/gmail/v1"
)

// GmailService is the interface that wraps the GetService method.
// GetService method must return a gmail service
type GmailService interface {
	GetService() *gmail.Service
}

func main() {
	var gmailService GmailService
	gmailService = &credentials.Credentials{}
	service := gmailService.GetService()

	scraper.Scrape(service)
}
