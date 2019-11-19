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

const userID = "me"

// Scrape will extract attachments contained in mails sent by a specific email.
func Scrape(service *gmail.Service) {
	start := time.Now()
	var email = flag.String("email", "",
		"we are to query mesages against this email")

	flag.Parse()

	var ms messageSevice
	ms = &message{}
	messagesChannel := getIDs(service, email, ms)
	messageContentChannel := make(chan *gmail.Message)
	attachmentChannel := make(chan *attachment)
	doneChannel := make(chan bool)

	go getMessageContent(messagesChannel, messageContentChannel, service)
	go getAttachment(service, messageContentChannel, attachmentChannel)
	go saveAttachment(attachmentChannel, doneChannel)

	<-doneChannel
	fmt.Println(time.Since(start))
}

type messageSevice interface {
	fetchMessages(service *gmail.Service,
		query string) (*gmail.ListMessagesResponse, error)
	fetchNextPage(service *gmail.Service,
		query string,
		NextPageToken string) (*gmail.ListMessagesResponse, error)
}

type attachment struct {
	data     string
	fileName string
}

type message struct {
}

func getIDs(service *gmail.Service,
	email *string,
	m messageSevice) <-chan string {
	query := fmt.Sprintf("from:%s", *email)

	msgs := []*gmail.Message{}

	r, err := m.fetchMessages(service, query)
	if err != nil {
		log.Fatalf("Unable to retrieve Messages: %v", err)
	}
	msgs = append(msgs, r.Messages...)

	for len(r.NextPageToken) != 0 {
		r, err = m.fetchNextPage(service, query, r.NextPageToken)
		if err != nil {
			log.Fatalf("Unable to retrieve Messages on the next page: %v", err)
		}
		msgs = append(msgs, r.Messages...)
	}

	if len(r.Messages) == 0 {
		fmt.Println("No messages found.")
	}

	ids := make(chan string)

	var wg sync.WaitGroup
	for _, msg := range msgs {
		wg.Add(1)
		go func(msg *gmail.Message) {
			defer wg.Done()
			ids <- msg.Id
		}(msg)
	}

	go func() {
		wg.Wait()
		close(ids)
	}()

	return ids
}

func (m *message) fetchMessages(
	service *gmail.Service,
	query string) (*gmail.ListMessagesResponse, error) {
	r, err := service.Users.Messages.List(userID).Q(query).Do()
	return r, err
}

func (m *message) fetchNextPage(
	service *gmail.Service,
	query string,
	NextPageToken string) (*gmail.ListMessagesResponse, error) {
	r, err := service.Users.Messages.List(userID).Q(query).
		PageToken(NextPageToken).Do()
	return r, err
}

func getMessageContent(
	ids <-chan string,
	msgCh chan *gmail.Message,
	service *gmail.Service) {
	var wg sync.WaitGroup
	for id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			msgContent, err := service.Users.Messages.Get(userID, id).Do()
			if err != nil {
				log.Fatalf("Unable to retrieve Message Contents: %v", err)
			}
			msgCh <- msgContent
		}(id)
	}
	wg.Wait()
	close(msgCh)
}

func getAttachment(
	service *gmail.Service,
	msgContentCh chan *gmail.Message,
	attachCh chan *attachment) {
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
					msgPartBody, err := service.Users.Messages.Attachments.
						Get(userID, msgContent.Id, part.Body.AttachmentId).Do()
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
