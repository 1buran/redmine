package redmine

import (
	"net/url"
)

// POST /time_entries params
type PostTimeEntryParams struct {
	Payload CreateTimeEntryPayload `json:"time_entry"`
}

func (t PostTimeEntryParams) Validate() error { return t.Payload.Validate() }
func (t PostTimeEntryParams) Url(base string) (string, error) {
	return url.JoinPath(base, TimeEntriesEndpoint)
}

func NewPostTimeEntryParams() *PostTimeEntryParams {
	return &PostTimeEntryParams{}
}

// POST /issues params
type PostDataIssue struct {
	Payload CreateIssuePayload `json:"issue"`
}

func NewPostIssueParams() *PostDataIssue {
	return &PostDataIssue{}
}

func (i PostDataIssue) Validate() error { return i.Payload.Validate() }
func (i PostDataIssue) Url(base string) (string, error) {
	return url.JoinPath(base, IssuesApiEndpoint)
}

// PostData is a generic container for payloads of API endpoints.
type PostData interface {
	PostTimeEntryParams | PostDataIssue

	Validate() error
	Url(base string) (string, error)
}
