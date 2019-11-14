package scraper

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/api/gmail/v1"
)

// Scrape will extract attachments contained in mails sent by a specific email.
func Scrape(service *gmail.Service) {
	var email = flag.String("email", "", "we are to query mesages against this email")

	flag.Parse()

	messages := getMessages(service, email)

	for _, msg := range messages {
		msgContent, _ := getMessageContent(msg.Id, service)
		tm := time.Unix(0, msgContent.InternalDate*1e6)
		for _, part := range msgContent.Payload.Parts {

			if len(part.Filename) != 0 {
				newFileName := tm.Format("Jan-02-2006") + "-" + part.Filename
				attachment, _ := getAttachment(service, msgContent.Id, part.Body.AttachmentId)

				decoded, _ := base64.URLEncoding.DecodeString(attachment.Data)
				const path = "./attachments/"
				if _, err := os.Stat(path); os.IsNotExist(err) {
					os.Mkdir(path, os.ModePerm)
				}
				f, err := os.Create(path + newFileName)
				if err != nil {
					panic(err)
				}
				defer f.Close()

				if _, err := f.Write(decoded); err != nil {
					panic(err)
				}
				if err := f.Sync(); err != nil {
					panic(err)
				}

			}

		}
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

func getAttachment(service *gmail.Service, messageID string, attachmentID string) (*gmail.MessagePartBody, error) {
	return service.Users.Messages.Attachments.Get("me", messageID, attachmentID).Do()
}
