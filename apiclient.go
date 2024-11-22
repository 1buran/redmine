package redmine

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const (
	ProjectsApiEndpoint = "/projects.json"
	IssuesApiEndpoint   = "/issues.json"
	TimeEntriesEndpoint = "/time_entries.json"
)

// Config of Redmine REST API client: url, token, logging and time entries filtration.
type ApiClient struct {
	Url        string
	Token      string
	LogEnabled bool
	TimeEntriesFilter
}

func (ac ApiClient) ProjectsUrl(page int) (string, error) {
	v := url.Values{}
	return BuildApiUrl(ac.Url, ProjectsApiEndpoint, &v, page)
}

func (ac ApiClient) IssuesUrl(page int) (string, error) {
	v := url.Values{}
	return BuildApiUrl(ac.Url, IssuesApiEndpoint, &v, page)
}

func (ac ApiClient) TimeEntriesUrl(page int) (string, error) {
	v := url.Values{}
	v.Set("user_id", ac.UserId)
	v.Set("from", ac.StartDate.Format("2006-01-02"))
	v.Set("to", ac.EndDate.Format("2006-01-02"))
	return BuildApiUrl(ac.Url, TimeEntriesEndpoint, &v, page)
}

// Create entity
func (ac ApiClient) Create(url string, data io.Reader) error {
	rcode, rbody, err := ac.Post(url, data)
	if err != nil {
		return err
	}

	if rcode != 201 {
		msg, err := io.ReadAll(rbody)
		return errors.Join(
			HttpError, fmt.Errorf("response code: %d, body: %s", rcode, msg), err)
	}
	return nil
}

// Post Redmine entity
func (ac ApiClient) Post(url string, data io.Reader) (int, io.ReadCloser, error) {
	http_cli := http.Client{}

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		// actually this block is never be run cos the url already passed the validation
		// in ApiEndpointURL function,
		// method is correct and hardcoded, there are no other cases when the
		// NewRequest will failed (check the source code)
		return -1, nil, errors.Join(ApiNewRequestFatalError, err)
	}

	req.Header.Add("User-Agent", "redmine go client v0.1")
	req.Header.Add("X-Redmine-API-Key", ac.Token)
	req.Header.Add("Content-Type", "application/json")

	if ac.LogEnabled {
		log.Printf("> %s %s", req.Method, req.URL)
	}
	res, err := http_cli.Do(req)
	if err != nil {
		return -1, nil, errors.Join(HttpError, err)
	}
	if ac.LogEnabled {
		log.Printf("< %s", res.Status)
	}

	return res.StatusCode, res.Body, nil
}

// Get Redmine entities respecting the setted filtration (time entries) and page of pagination.
func (ac ApiClient) Get(url string) (io.ReadCloser, error) {
	http_cli := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
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

	return res.Body, nil
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

func CreateApiClient(url, token string, logging bool, teFilter TimeEntriesFilter) *ApiClient {
	return &ApiClient{Url: url, Token: token, LogEnabled: logging, TimeEntriesFilter: teFilter}
}
