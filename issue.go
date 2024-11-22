package redmine

import (
	"errors"
	"fmt"
)

// A Redmine issue entity.
type Issue struct {
	Id      int    `json:"id"`
	Subject string `json:"subject"`
	Desc    string `json:"description"`
	Project `json:"project"`
}

func (i Issue) String() string {
	return fmt.Sprintf("%-5d %s %s", i.Id, i.Project.Name, i.Subject)
}

type Issues struct {
	Items []Issue `json:"issues"`
	Pagination
}

// Payload of Redmine API POST /issues
type CreateIssuePayload struct {
	ProjectID  int     `json:"project_id,omitempty"`
	TrackerID  int     `json:"tracker_id,omitempty"`
	StatusID   int     `json:"status_id,omitempty"`
	PriorityID int     `json:"priority_id,omitempty"`
	CategoryID int     `json:"category_id,omitempty"`
	ParrentID  int     `json:"parent_issue_id,omitempty"`
	FixedVerID int     `json:"fixed_version_id,omitempty"`
	AssignedID int     `json:"assigned_to_id,omitempty"`
	Watchers   []int   `json:"watcher_user_ids,omitempty"`
	Subject    string  `json:"string,omitempty"`
	Desc       string  `json:"description,omitempty"`
	Private    bool    `json:"is_private,omitempty"`
	Estimate   float32 `json:"estimated_hours,omitempty"`
}

// Validate payload.
func (p CreateIssuePayload) Validate() error {
	if p.ProjectID == 0 {
		return errors.Join(ValidationError, EmptyProjectError)
	}
	return nil
}
