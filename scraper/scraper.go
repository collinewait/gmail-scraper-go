package scraper

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/api/gmail/v1"
)

const userID = "me"

// Scrape will extract attachments contained in mails sent by a specific email.
func Scrape(service *gmail.Service) {
	osf := os.Stdin
	emailThatSentAttach := getEmailThatSentAttachment(osf)

	var ms messageSevice
	var cont content
	var as attachmentService

	ms = &message{}
	cont = &messageContent{}
	as = &attachment{}

	messagesChannel, errorChannel := getIDs(service, emailThatSentAttach, ms)
	messageContentChannel, errorChannel := getMessageContent(messagesChannel, service, cont)
	attachmentChannel, errorChannel := getAttachment(messageContentChannel, service, as)
	doneChannel := make(chan bool)

	go saveAttachment(attachmentChannel, doneChannel)
	go exitOnError(errorChannel)

	<-doneChannel
}

func getEmailThatSentAttachment(in io.Reader) string {
	scanner := bufio.NewScanner(in)
	fmt.Print("Enter email that sent attachments: ")
	scanner.Scan()
	emailThatSent := scanner.Text()

	return emailThatSent
}

type messageSevice interface {
	fetchMessages(service *gmail.Service,
		query string) (*gmail.ListMessagesResponse, error)
	fetchNextPage(service *gmail.Service,
		query string,
		NextPageToken string) (*gmail.ListMessagesResponse, error)
}
type content interface {
	getContent(
		service *gmail.Service, id string) (*gmail.Message, error)
}
type attachmentService interface {
	fetchAttachment(
		service *gmail.Service,
		msgID string, attachID string) (*gmail.MessagePartBody, error)
}

type attachment struct {
	data     string
	fileName string
}
type message struct{}
type messageError struct {
	err error
	msg string
}
type messageContent struct{}

func getIDs(service *gmail.Service,
	email string,
	m messageSevice) (<-chan string, <-chan *messageError) {
	errs := new(messageError)
	errorsCh := make(chan *messageError, 1)
	defer close(errorsCh)
	query := fmt.Sprintf("from:%s", email)

	msgs := []*gmail.Message{}

	r, err := m.fetchMessages(service, query)
	if err != nil {
		errs.msg = "Unable to retrieve Messages"
		errs.err = err
		errorsCh <- errs
		return nil, errorsCh
	}
	msgs = append(msgs, r.Messages...)

	for len(r.NextPageToken) != 0 {
		r, err = m.fetchNextPage(service, query, r.NextPageToken)
		if err != nil {
			errs.msg = "Unable to retrieve Messages on the next page"
			errs.err = err
			errorsCh <- errs
			return nil, errorsCh
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

	return ids, nil
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
	service *gmail.Service,
	c content) (<-chan *gmail.Message, <-chan *messageError) {

	msgCh := make(chan *gmail.Message)
	errs := new(messageError)
	errorsCh := make(chan *messageError, 1)
	var wg sync.WaitGroup
	for id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			msgContent, err := c.getContent(service, id)
			if err != nil {
				errs.msg = "Unable to retrieve Message Contents"
				errs.err = err
				errorsCh <- errs
				close(errorsCh)
			}
			msgCh <- msgContent
		}(id)
	}
	go func() {
		wg.Wait()
		close(msgCh)
	}()
	return msgCh, errorsCh
}

func (mc *messageContent) getContent(
	service *gmail.Service, id string) (*gmail.Message, error) {
	return service.Users.Messages.Get(userID, id).Do()
}

func getAttachment(
	msgContentCh <-chan *gmail.Message,
	service *gmail.Service,
	as attachmentService,
) (<-chan *attachment, <-chan *messageError) {
	var wg sync.WaitGroup
	attachCh := make(chan *attachment)
	errs := new(messageError)
	errorsCh := make(chan *messageError, 1)

	for msgContent := range msgContentCh {
		wg.Add(1)
		go func(msgContent *gmail.Message) {
			defer wg.Done()
			attach := new(attachment)
			tm := time.Unix(0, msgContent.InternalDate*1e6)
			for _, part := range msgContent.Payload.Parts {
				if len(part.Filename) != 0 {
					newFileName := tm.Format("Jan-02-2006") + "-" + part.Filename
					msgPartBody, err := as.fetchAttachment(service, msgContent.Id, part.Body.AttachmentId)
					if err != nil {
						errs.msg = "Unable to retrieve Attachment"
						errs.err = err
						errorsCh <- errs
						close(errorsCh)
					}
					attach.data = msgPartBody.Data
					attach.fileName = newFileName
					attachCh <- attach
				}
			}
		}(msgContent)
	}
	go func() {
		wg.Wait()
		close(attachCh)
	}()

	return attachCh, errorsCh
}

func (a *attachment) fetchAttachment(
	service *gmail.Service, msgID string, attachID string) (*gmail.MessagePartBody, error) {
	return service.Users.Messages.Attachments.
		Get(userID, msgID, attachID).Do()
}

func saveAttachment(attachCh <-chan *attachment, doneCh chan bool) {
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

func exitOnError(errCh <-chan *messageError) {
	for e := range errCh {
		log.Fatalf("%s: %v", e.msg, e.err)
	}
}
