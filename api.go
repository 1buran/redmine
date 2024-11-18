// This is a lightweight Redmine API client.
//
// It doesn't do a lot of things, you might probably only be interested in the
// scrolling feature [Scroll].
package redmine

import (
	"encoding/json"
	"errors"
	"io"
	"log"
)

// There are some custom error types, from low level to high level errors
// which are aggregates of first ones.
//
// Typically you should be expect only these high level errors in errChan:
//   - [JsonDecodeError]: errors related to unmarshaling redmine server response
//   - [IoReadError]: errors related to read input
//   - [HttpError]: errors related to network layer
//   - [ApiEndpointUrlFatalError]: fatal errors that means that most probably
//     the url of redmine api is malformed or bogus, please check it
//   - [ApiNewRequestFatalError]: actually will not be thrown (see the comments in code)
var (
	JsonDecodeError          = errors.New("JSON decode error")
	IoReadError              = errors.New("io.ReadAll error")
	UrlJoinPathError         = errors.New("url.JoinPath error")
	UrlParseError            = errors.New("url.Parse error")
	ApiEndpointUrlFatalError = errors.New("cannot build API endpoint url")
	ApiNewRequestFatalError  = errors.New("cannot create a new request with given url")
	HttpError                = errors.New("http error")
	UnknownDataTypeError     = errors.New("unknown or not supported data type requested")
)

// Data type constraint, a quick glance at which will let you know the supported data types
// for fetching from redmine server.
type Entities interface {
	Projects | Issues | TimeEntries

	NextPage() (n int)
}

// Decode JSON Redmine API response to package types.
func DecodeResp[E Entities](body io.ReadCloser) (*E, error) {
	defer body.Close()
	apiResp := new(E)

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, errors.Join(IoReadError, err)
	}

	if err := json.Unmarshal(data, apiResp); err != nil {
		return nil, errors.Join(JsonDecodeError, err)
	}

	return apiResp, nil
}

func ApiUrl[E Entities](ac *ApiClient, page int) (string, error) {
	e := new(E)
	switch any(*e).(type) {
	case Projects:
		return ac.ProjectsUrl(page)
	case Issues:
		return ac.IssuesUrl(page)
	case TimeEntries:
		return ac.TimeEntriesUrl(page)
	}
	return "", UnknownDataTypeError
}

// Scroll over Redmine API paginated responses. It going through all available data,
// so it may generate a lot of http requests (depending on a size of data and pagination limit).
//
// The pagination of redmine is based on offset&limit,
// but in URL you may use query string param ?page=, e.g. for 53 issues and limit=25 it will be
// three requests:
//   - 0  25 53 - [0, 25] /issues.json?page=1 or omitted page number: /issues.json
//   - 25 25 53 - [25, 50] /issues.json?page=2
//   - 50 25 53 - [50, 53] /issues.json?page=3
//
// This function do this automatically and send all the data to channel,
// if any error occurs, it will be send to the second, errors channel.
func Scroll[E Entities](ac *ApiClient) (<-chan E, <-chan error) {
	page := 1
	dataChan := make(chan E, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(dataChan)
		defer close(errChan)
		for page > -1 {
			api_endpoint_url, err := ApiUrl[E](ac, page)
			if err != nil {
				errChan <- errors.Join(ApiEndpointUrlFatalError, err)
				break
			}
			resp, err := ac.Get(api_endpoint_url)
			if err != nil {
				// first of all send error to err channel
				errChan <- err
				// analyze error and perform appropriate action
				switch {
				case errors.Is(err, ApiEndpointUrlFatalError),
					errors.Is(err, ApiNewRequestFatalError):
					log.Println("Scroll fatal error: ", err)
					return
				case errors.Is(err, HttpError):
					log.Println("Scroll error:", err)
					// todo control retries: count and delay...
				}
				continue
			}
			r, err := DecodeResp[E](resp)
			if err != nil {
				errChan <- err
				log.Println("Scroll error: ", err)
				continue
			}

			dataChan <- *r
			page = extractNextPage[E](*r)
		}
	}()

	return dataChan, errChan
}

func extractNextPage[E Entities](e E) int { return e.NextPage() }
