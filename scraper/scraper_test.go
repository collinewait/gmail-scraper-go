package scraper

import (
	"reflect"
	"sort"
	"testing"

	"google.golang.org/api/gmail/v1"
)

type mockMessageSevice interface {
	fetchMessages(
		service *gmail.Service,
		query string) (*gmail.ListMessagesResponse, error)
	fetchNextPage(service *gmail.Service,
		query string,
		NextPageToken string) (*gmail.ListMessagesResponse, error)
}

type mockMessage struct {
}

func (m *mockMessage) fetchMessages(
	service *gmail.Service,
	query string) (*gmail.ListMessagesResponse, error) {
	gm := []*gmail.Message{
		{Id: "16c2"},
		{Id: "41ff9"},
		{Id: "41hfi"},
		{Id: "fgb"},
		{Id: "ifgh9"},
	}

	r := gmail.ListMessagesResponse{
		Messages: gm,
	}
	return &r, nil
}
func (m *mockMessage) fetchNextPage(
	service *gmail.Service,
	query string,
	NextPageToken string) (*gmail.ListMessagesResponse, error) {
	r := gmail.ListMessagesResponse{
		Messages: []*gmail.Message{},
	}
	return &r, nil
}

func Test_getIDsCanReturnIDsWithoutNextPageToken(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "test case without NextPageToken",
			want: []string{"16c2", "41ff9", "41hfi", "fgb", "ifgh9"},
		},
	}
	service := new(gmail.Service)
	testmail := "test@mail.com"
	var ms mockMessageSevice
	ms = &mockMessage{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cgot := getIDs(service, &testmail, ms)
			var got []string

			for i := range cgot {
				got = append(got, i)
			}

			sort.Sort(sort.StringSlice(got))
			sort.Sort(sort.StringSlice(tt.want))

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockMessageWithNextPage struct {
}

func (m *mockMessageWithNextPage) fetchMessages(
	service *gmail.Service,
	query string) (*gmail.ListMessagesResponse, error) {
	gm := []*gmail.Message{
		{Id: "16c2"},
		{Id: "41ff9"},
		{Id: "41hfi"},
		{Id: "fgb"},
		{Id: "ifgh9"},
	}

	r := gmail.ListMessagesResponse{
		Messages:      gm,
		NextPageToken: "someToken",
	}
	return &r, nil
}

func (m *mockMessageWithNextPage) fetchNextPage(
	service *gmail.Service,
	query string,
	NextPageToken string) (*gmail.ListMessagesResponse, error) {
	gm := []*gmail.Message{
		{Id: "fgbmm"},
	}

	r := gmail.ListMessagesResponse{
		Messages: gm,
	}
	return &r, nil
}

func Test_getIDsCanReturnIDsWithNextPageToken(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "test case with NextPageToken",
			want: []string{"16c2", "41ff9", "41hfi", "fgb", "ifgh9", "fgbmm"},
		},
	}
	service := new(gmail.Service)
	testmail := "test@mail.com"
	var ms mockMessageSevice
	ms = &mockMessageWithNextPage{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cgot := getIDs(service, &testmail, ms)
			var got []string

			for i := range cgot {
				got = append(got, i)
			}

			sort.Sort(sort.StringSlice(got))
			sort.Sort(sort.StringSlice(tt.want))

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}
