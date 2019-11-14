package scraper

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/api/gmail/v1"
)

// Scrape will extract attachments contained in mails sent by a specific email.
func Scrape(service *gmail.Service) {
	var email = flag.String("email", "", "we are to query mesages against this email")

	flag.Parse()
	messagesChannel := make(chan *gmail.Message)
	messageContentChannel := make(chan *gmail.Message)
	attachmentChannel := make(chan *attachment)
	doneChannel := make(chan bool)

	go getMessages(service, email, messagesChannel)
	go getMessageContent(messagesChannel, messageContentChannel, service)
	go getAttachment(service, messageContentChannel, attachmentChannel)
	go saveAttachment(attachmentChannel, doneChannel)

	<-doneChannel
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

	var wg sync.WaitGroup
	for _, msg := range msgs {
		wg.Add(1)
		go func(msg *gmail.Message) {
			defer wg.Done()
			msgsCh <- msg
		}(msg)
	}
	wg.Wait()
	close(msgsCh)
}

func getMessageContent(msgsCh, msgCh chan *gmail.Message, service *gmail.Service) {
	var wg sync.WaitGroup
	for msg := range msgsCh {
		wg.Add(1)
		go func(msg *gmail.Message) {
			defer wg.Done()
			msgContent, err := service.Users.Messages.Get("me", msg.Id).Do()
			if err != nil {
				log.Fatalf("Unable to retrieve Message Contents: %v", err)
			}
			msgCh <- msgContent
		}(msg)
	}
	wg.Wait()
	close(msgCh)
}

func getAttachment(service *gmail.Service, msgContentCh chan *gmail.Message, attachCh chan *attachment) {
	var wg sync.WaitGroup
	for msgContent := range msgContentCh {
		wg.Add(1)
		go func(msgContent *gmail.Message) {
			defer wg.Done()
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
		}(msgContent)
	}
	wg.Wait()
	close(attachCh)
}

func saveAttachment(attachCh chan *attachment, doneCh chan bool) {
	var wg sync.WaitGroup
	for attach := range attachCh {
		wg.Add(1)
		go func(attach *attachment) {
			defer wg.Done()
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
		}(attach)
	}
	wg.Wait()
	doneCh <- true
}
