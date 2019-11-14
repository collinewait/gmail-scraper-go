package scraper

import (
	"flag"
	"fmt"
	"log"

	"google.golang.org/api/gmail/v1"
)

// Scrape will extract attachments contained in mails sent by a specific email.
func Scrape(service *gmail.Service) {
	var email = flag.String("email", "", "we are to query mesages against this email")

	flag.Parse()

	messages := getMessages(service, email)

	for _, msg := range messages {
		msgContent, _ := getMessageContent(msg.Id, service)
		println("msgContent.Id", msgContent.Id)
	}

}

func getMessages(service *gmail.Service, email *string) []*gmail.Message {
	query := fmt.Sprintf("from:%s", *email)

	msgs := []*gmail.Message{}

	r, err := service.Users.Messages.List("me").Q(query).Do()
	msgs = append(msgs, r.Messages...)

	for len(r.NextPageToken) != 0 {
		r, err = service.Users.Messages.List("me").Q(query).PageToken(r.NextPageToken).Do()
		msgs = append(msgs, r.Messages...)
	}

	if err != nil {
		log.Fatalf("Unable to retrieve Messages: %v", err)
	}
	if len(r.Messages) == 0 {
		fmt.Println("No messages found.")
	}

	return msgs
}

func getMessageContent(messageID string, service *gmail.Service) (*gmail.Message, error) {
	msg, err := service.Users.Messages.Get("me", messageID).Do()
	return msg, err
}
