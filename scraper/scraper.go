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
	start := time.Now()
	messagesChannel := make(chan *gmail.Message)
	messageContentChannel := make(chan *gmail.Message)
	attachmentChannel := make(chan *attachment)
	doneChannel := make(chan bool)

	go getMessages(service, email, messagesChannel)
	go getMessageContent(messagesChannel, messageContentChannel, service)
	go getAttachment(service, messageContentChannel, attachmentChannel)
	go saveAttachment(attachmentChannel, doneChannel)

	<-doneChannel
	fmt.Println("Elapsed: ", time.Since(start))
}

type attachment struct {
	data     string
	fileName string
}

func getMessages(service *gmail.Service, email *string, msgsCh chan *gmail.Message) {
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

	for _, msg := range msgs {
		msgsCh <- msg
	}
	close(msgsCh)
}

func getMessageContent(msgsCh, msgCh chan *gmail.Message, service *gmail.Service) {
	for msg := range msgsCh {
		msgContent, err := service.Users.Messages.Get("me", msg.Id).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve Message Contents: %v", err)
		}
		msgCh <- msgContent
	}
	close(msgCh)
}

func getAttachment(service *gmail.Service, msgContentCh chan *gmail.Message, attachCh chan *attachment) {
	for msgContent := range msgContentCh {
		attach := new(attachment)
		tm := time.Unix(0, msgContent.InternalDate*1e6)
		for _, part := range msgContent.Payload.Parts {
			if len(part.Filename) != 0 {
				newFileName := tm.Format("Jan-02-2006") + "-" + part.Filename
				msgPartBody, err := service.Users.Messages.Attachments.Get("me", msgContent.Id, part.Body.AttachmentId).Do()
				if err != nil {
					log.Fatalf("Unable to retrieve Attachment: %v", err)
				}
				attach.data = msgPartBody.Data
				attach.fileName = newFileName
				attachCh <- attach
			}
		}
	}
	close(attachCh)
}

func saveAttachment(attachCh chan *attachment, doneCh chan bool) {
	for attach := range attachCh {
		decoded, _ := base64.URLEncoding.DecodeString(attach.data)
		const path = "./attachments/"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}
		f, err := os.Create(path + attach.fileName)
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
	doneCh <- true
}
