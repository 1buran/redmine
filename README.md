# Redmine REST API client

[![codecov](https://codecov.io/github/1buran/redmine/graph/badge.svg?token=NNJYP8B5V8)](https://codecov.io/github/1buran/redmine)
[![Go Reference](https://pkg.go.dev/badge/github.com/1buran/redmine.svg)](https://pkg.go.dev/github.com/1buran/redmine)
[![goreportcard](https://goreportcard.com/badge/github.com/1buran/redmine)](https://goreportcard.com/report/github.com/1buran/redmine)

This is a lightweight Redmine API client.

## Introduction

> [!CAUTION]
> I created it for my own personal use (while I was learning the Go language),
> you probably shouldn't use it! At least until it ready for production: ver >= 1.0.

I wanted create something like TUI of Redmine, but when i dive into learning of TUI libs like bubbletea, i spent all my free time slots and this idea was on pause for a while...until once i needed
small lib to get all my time entries from redmine to build over this one some another cool TUI app.

So basically this lib, a Go module, used for scroll over all Redmine time entries.

Another functional will be added (maybe) later...

## Installation

```sh
go get github.com/1buran/redmine
```

## Usage

Supported Redmine types:

- `User`
- `Project`
- `TimeEntry`
- `Issue`

```go
// Configure the time entries filter: set start and end date of time frame and user id
// for whom you want to get time entries
start := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
end := start.AddDate(0, 0, 30)

timeEntriesFilter := redmine.TimeEntriesFilter{
    StartDate: start,
    EndDate: end,
    UserId: "1"
}

// Create an API client: set Redmine REST API url, token, enable or disable logging
apiClient := redmine.CreateApiClient(
    "https://example.com", "asdadwwefwefwf", true, timeEntriesFilter)

// Open the channels to data and errors from the redmine client:
// dataChan, errChan := redmine.Scroll[redmine.Projects](apiClient)
// dataChan, errChan := redmine.Scroll[redmine.Issues](apiClient)
dataChan, errChan := redmine.Scroll[redmine.TimeEntries](apiClient)
for {
    select {
    case data, ok := <-dataChan:
        if ok { // data channel is open
            // perform action on the gotten items e.g. print the data
            for _, t := range data.Items {
                fmt.Printf("On %s user spent %.2f hours for %s\n", t.SpentOn, t.Hours, t.Comment)
            }
            continue
        }
        return // data channel is closed, all data is transmitted, return to the main loop
    case err, ok := <-errChan:
        if ok { // err channel is open
            // perform action depending on the gotten error and your business logic
            switch {
                case errors.Is(err, redmine.ApiEndpointUrlFatalError):
                    fmt.Fatalf("redmine api url is malformed: %s", err)
                case errors.Is(err, redmine.HttpError):
                    fmt.Println("http error: %s, retry...", err)
                default:
                    fmt.Println("err: %s", err)
            }
        }
    }
}
```

There are some custom error types, from low level to high level errors which are aggregates of first ones. Typically you should be expect only these high level errors in errChan:

- `JsonDecodeError`: errors related to unmarshaling redmine server response
- `IoReadError`: errors related to read input (`io.ReadAll(body)`)
- `HttpError`: errors related to network layer (`http_client.Do(req)`)
- `ApiEndpointUrlFatalError`: fatal errors that means that most probably
  the url of redmine api is malformed or bogus, please check it
- `ApiNewRequestFatalError`: actually will not be thrown (see the comments in code)
