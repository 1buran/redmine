// This is a lightweight Redmine API client.
//
// It doesn't do a lot of things, you might probably only be interested in the
// scrolling feature [Scroll].
package redmine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	ProjectsApiEndpoint = "/projects.json"
	IssuesApiEndpoint   = "/issues.json"
	TimeEntriesEndpoint = "/time_entries.json"
)

// Time Entries filtration by range of dates and user id.
type TimeEntriesFilter struct {
	StartDate time.Time
	EndDate   time.Time
	UserId    string
}

// Config of Redmine REST API client: url, token, logging and time entries filtration.
type ApiConfig struct {
	Url        string
	Token      string
	LogEnabled bool
	TimeEntriesFilter
}

// A Redmine issue entity.
type Issue struct {
	Id      int    `json:"id"`
	Subject string `json:"subject"`
	Desc    string `json:"description"`
	Project `json:"project"`
}

// A Redmine project entity.
type Project struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Ident string `json:"identifier"`
	Desc  string `json:"description"`
	// TODO correct parsing date time
	// CreatedOn time.Time `json:"created_on"`
	// UpdatedOn time.Time `json:"updated_on"`
	IsPublic bool `json:"is_public"`
}

// A Redmine user entity.
type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// A date type is needed for proper parsing (unmarshaling) of redmine date format used in JSON.
type Date struct {
	time.Time
}

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
)

// Unmarshaling redmine dates.
func (d *Date) UnmarshalJSON(b []byte) error {
	t, err := time.Parse("2006-01-02", string(bytes.Trim(b, "\"")))
	if err != nil {
		return errors.Join(JsonDecodeError, err)
	}
	d.Time = t
	return nil
}

func (d Date) String() string {
	return d.Time.Format("2006-01-02")
}

// A Redmine time entries.
type TimeEntry struct {
	Id      int `json:"id"`
	Project `json:"project"`
	Issue   `json:"issue"`
	User    `json:"user"`
	Hours   float32 `json:"hours"`
	Comment string  `json:"comments"`
	SpentOn Date    `json:"spent_on"`
}

type Pagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Total  int `json:"total_count"`
}

func (t TimeEntry) String() string {
	return fmt.Sprintf(
		"%-5d %5.2f %s %-15s %s", t.Issue.Id, t.Hours, t.SpentOn, t.User.Name, t.Comment)
}

func (i Issue) String() string {
	return fmt.Sprintf("%-5d %s %s", i.Id, i.Project.Name, i.Subject)
}

// Data type constraint, a quick glance at which will let you know the supported data types
// for fetching from redmine server.
type Entities interface {
	Project | Issue | TimeEntry
}

// Redmine API items response container.
type ApiResponse[E Entities] struct {
	Items []E
	Pagination
}

// Decode JSON Redmine API response to package types.
func DecodeResp[E Entities](body io.ReadCloser) (*ApiResponse[E], error) {
	defer body.Close()
	apiResp := ApiResponse[E]{}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, errors.Join(IoReadError, err)
	}

	// KLUDGE because there is no way to make generic struct tag,
	// we have to replace original json node key to common "Items"

	var b []byte
	e := new(E)
	switch any(*e).(type) {
	case Project:
		b = bytes.Replace(data, []byte("projects"), []byte("Items"), 1)
	case Issue:
		b = bytes.Replace(data, []byte("issues"), []byte("Items"), 1)
	case TimeEntry:
		b = bytes.Replace(data, []byte("time_entries"), []byte("Items"), 1)
	}
	if err = json.Unmarshal(b, &apiResp); err != nil {
		return nil, errors.Join(JsonDecodeError, err)
	}

	// TODO find a way to make generic struct tag for simplify code:
	// if err := json.NewDecoder(body).Decode(&apiResp); err != nil {
	// 	return nil, err
	// }

	return &apiResp, nil

}

// Add pagination query string to URL.
func BuildApiUrl(base, endpoint string, v *url.Values, p int) (string, error) {
	uri, err := url.JoinPath(base, endpoint)
	if err != nil {
		return "", errors.Join(UrlJoinPathError, err)
	}

	if p > 1 {
		v.Add("page", strconv.Itoa(p))
	}

	if rq := v.Encode(); rq != "" {
		u, err := url.Parse(uri)
		if err != nil {
			return "", errors.Join(UrlParseError, err)
		}
		u.RawQuery = rq
		return u.String(), nil
	}

	return uri, nil
}

// Construct the final URL for http requests depending on redmine entities
// (projects, issues or time entries) and pagination, filtration.
func ApiEndpointURL[E Entities](ac *ApiConfig, page int) (u string, err error) {
	v := url.Values{}
	e := new(E)
	switch any(*e).(type) {
	case Project:
		u, err = BuildApiUrl(ac.Url, ProjectsApiEndpoint, &v, page)
	case Issue:
		u, err = BuildApiUrl(ac.Url, IssuesApiEndpoint, &v, page)
	case TimeEntry:
		// filter by user and dates: get the time entries of user for a month
		v.Set("user_id", ac.UserId)
		v.Set("from", ac.StartDate.Format("2006-01-02"))
		v.Set("to", ac.EndDate.Format("2006-01-02"))
		u, err = BuildApiUrl(ac.Url, TimeEntriesEndpoint, &v, page)
	}
	return
}

// Get Redmine entities respecting the setted filtration (time entries) and page of pagination.
func Get[E Entities](ac *ApiConfig, page int) (*ApiResponse[E], error) {
	http_cli := http.Client{}

	api_endpoint_url, err := ApiEndpointURL[E](ac, page)
	if err != nil {
		return nil, errors.Join(ApiEndpointUrlFatalError, err)
	}

	req, err := http.NewRequest("GET", api_endpoint_url, nil)
	if err != nil {
		// actually this block is never be run cos the url already passed the validation
		// in ApiEndpointURL function,
		// method is correct and hardcoded, there are no other cases when the
		// NewRequest will failed (check the source code)
		return nil, errors.Join(ApiNewRequestFatalError, err)
	}
	req.Header.Add("User-Agent", "redmine go client v0.1")
	req.Header.Add("X-Redmine-API-Key", ac.Token)
	if ac.LogEnabled {
		log.Printf("> %s %s", req.Method, req.URL)
	}
	res, err := http_cli.Do(req)
	if err != nil {
		return nil, errors.Join(HttpError, err)
	}
	if ac.LogEnabled {
		log.Printf("< %s", res.Status)
	}

	return DecodeResp[E](res.Body)
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
func Scroll[E Entities](ac *ApiConfig) (<-chan E, <-chan error) {
	var p int
	dataChan := make(chan E)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)
		oneMore := true
		for oneMore {
			r, err := Get[E](ac, p)
			if err != nil {
				// first of all send error to err channel
				errChan <- err
				// analyze error and perform appropriate action
				switch {
				case errors.Is(err, JsonDecodeError):
					log.Println(err)
				case errors.Is(err, IoReadError):
					log.Println(err)
				case errors.Is(err, ApiEndpointUrlFatalError):
					log.Println("fatal error: ", err)
					break
				case errors.Is(err, ApiNewRequestFatalError):
					log.Println("fatal error: ", err)
					break
				case errors.Is(err, HttpError):
					log.Println(err)
					// TODO control retries: count and delay...
				}
				continue
			}
			if r.Limit > 0 {
				p = (r.Offset+r.Limit)/r.Limit + 1
			}
			oneMore = r.Total-r.Offset > r.Limit
			for _, v := range r.Items {
				dataChan <- v
			}
		}
	}()

	return dataChan, errChan
}
