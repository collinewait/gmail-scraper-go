# Gmail-scraper-go

This app downloads attachments sent by a specific email. I had a couple of attachments sent to me over a long period of time. So I decided to create this app to download them at once.

## Installation

You need to turn on the Gmail API. You can turn on the Gmail API by doing step 1 in [this](https://developers.google.com/gmail/api/quickstart/go#step_1_turn_on_the) documentation. You can also sign up for and create an API instance from the Google Developers Console.

- Have [Go](https://golang.org/dl/) installed on your machine
- Clone the repo with `git clone https://github.com/collinewait/gmail-scraper-go.git` and navigate to the project with `cd gmail-scraper-go`
- Install the following dependencies:
```bash
go get -u google.golang.org/api/gmail/v1
go get -u golang.org/x/oauth2/google
go get -u golang.org/x/net
```

## Run the app.

To run the application, use the following command `go run exec.go`

For the first time, it will prompt you to authorize access. Authorization code is based off code provided by Google to interact with the API. The steps below are also provided ([here](https://developers.google.com/gmail/api/quickstart/go#step_4_run_the_sample)):

- Browse to the provided URL in your web browser.
If you are not already logged into your Google account, you will be prompted to log in. If you are logged into multiple Google accounts, you will be asked to select one account to use for the authorization.
- Click the Accept button.
- Copy the code you're given, paste it into the command-line prompt, and press Enter.

Your attachments will be downloaded to a folder called `attachments` at the root of the project.

A web version can be found here: [Frontend](https://github.com/collinewait/ika-gmail-scraper-frontend) and a [backend](https://github.com/collinewait/ika-gmail-scraper-backend)
