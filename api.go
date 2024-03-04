package redmine

import (
	"bytes"
	"encoding/json"
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

type TimeEntriesFilter struct {
	StartDate time.Time
	EndDate   time.Time
	UserId    string
}

type ApiConfig struct {
	Url        string
	Token      string
	LogEnabled bool
	TimeEntriesFilter
}

type Issue struct {
	Id      int    `json:"id"`
	Subject string `json:"subject"`
	Desc    string `json:"description"`
	Project `json:"project"`
}

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

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Date struct {
	time.Time
}

func (d *Date) UnmarshalJSON(b []byte) error {
	t, err := time.Parse("2006-01-02", string(bytes.Trim(b, "\"")))
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

func (d Date) String() string {
	return d.Time.Format("2006-01-02")
}

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

type Entities interface {
	Project | Issue | TimeEntry
}

// Redmine API items response container
type ApiResponse[E Entities] struct {
	Items []E
	Pagination
}

// Decode JSON Redmine API response to local entities
func DecodeResp[E Entities](body io.ReadCloser) (*ApiResponse[E], error) {
	defer body.Close()
	apiResp := ApiResponse[E]{}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	// KLUDGE because there is no way to make generic struct tag,
	// we have to replace original json node key to common "Items"

	var b []byte
	e := new(E)
	switch (interface{})(*e).(type) {
	case Project:
		b = bytes.Replace(data, []byte("projects"), []byte("Items"), 1)
	case Issue:
		b = bytes.Replace(data, []byte("issues"), []byte("Items"), 1)
	case TimeEntry:
		b = bytes.Replace(data, []byte("time_entries"), []byte("Items"), 1)
	}
	if err = json.Unmarshal(b, &apiResp); err != nil {
		return nil, err
	}

	// TODO find a way to make generic struct tag for simplify code:
	// if err := json.NewDecoder(body).Decode(&apiResp); err != nil {
	// 	return nil, err
	// }

	return &apiResp, nil

}

// Add pagination query string to URL
func BuildApiUrl(base, endpoint string, v *url.Values, p int) (string, error) {
	uri, err := url.JoinPath(base, endpoint)
	if err != nil {
		return "", err
	}

	if p > 1 {
		v.Add("page", strconv.Itoa(p))
	}

	if rq := v.Encode(); rq != "" {
		u, err := url.Parse(uri)
		if err != nil {
			return "", err
		}
		u.RawQuery = rq
		return u.String(), nil
	}

	return uri, err
}

// Redmine API Endpoint URL of redmine entity: projects, issues or time entries
func ApiEndpointURL[E Entities](ac *ApiConfig, page int) (string, error) {
	v := url.Values{}
	e := new(E)
	switch (interface{})(*e).(type) {
	case Project:
		return BuildApiUrl(ac.Url, ProjectsApiEndpoint, &v, page)
	case Issue:
		return BuildApiUrl(ac.Url, IssuesApiEndpoint, &v, page)
	case TimeEntry:
		// filter by user and dates: get the time entries of user for a month
		v.Set("user_id", ac.UserId)
		v.Set("from", ac.StartDate.Format("2006-01-02"))
		v.Set("to", ac.EndDate.Format("2006-01-02"))
		return BuildApiUrl(ac.Url, TimeEntriesEndpoint, &v, page)
	}
	return "", fmt.Errorf("unknown entity")
}

// Get Redmine entities
func Get[E Entities](ac *ApiConfig, page int) (*ApiResponse[E], error) {
	http_cli := http.Client{}

	api_endpoint_url, err := ApiEndpointURL[E](ac, page)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", api_endpoint_url, nil)
	req.Header.Add("User-Agent", "redmine go client v0.1")
	req.Header.Add("X-Redmine-API-Key", ac.Token)
	if ac.LogEnabled {
		log.Printf("> %s %s", req.Method, req.URL)
	}
	res, err := http_cli.Do(req)
	if err != nil {
		return nil, err
	}
	if ac.LogEnabled {
		log.Printf("< %d %s", res.StatusCode, res.Status)
	}

	return DecodeResp[E](res.Body)
}

// Scroll over Redmine API paginated responses, the pagination of redmine is based on offset&limit,
// but in URL you may use query string param ?page=, e.g. for 53 items and limit=25 it will be
// three requests:
// 0  25 53 - [0, 25] page=1
// 25 25 53 - [25, 50] page=2
// 50 25 53 - [50, 53] page=3
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
				errChan <- err
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
