package redmine

import (
	"errors"
	"fmt"
	"time"
)

// Time Entries filtration by range of dates and user id.
type TimeEntriesFilter struct {
	StartDate time.Time
	EndDate   time.Time
	UserId    string
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

func (t TimeEntry) String() string {
	return fmt.Sprintf(
		"%-5d %5.2f %s %-15s %s", t.Issue.Id, t.Hours, t.SpentOn, t.User.Name, t.Comment)
}

type TimeEntries struct {
	Items []TimeEntry `json:"time_entries"`
	Pagination
}

// Payload of Redmine API POST /time_entries.
type CreateTimeEntryPayload struct {
	ProjectID  int     `json:"project_id,omitempty"`
	IssueID    int     `json:"issue_id,omitempty"`
	ActivityID int     `json:"activity_id,omitempty"`
	UserID     int     `json:"user_id,omitempty"`
	SpentOn    Date    `json:"spent_on,omitempty"`
	Comments   string  `json:"comments,omitempty"`
	Hours      float32 `json:"hours,omitempty"`
}

// Validate payload.
func (p CreateTimeEntryPayload) Validate() error {
	if p.SpentOn.IsZero() {
		return errors.Join(ValidationError, ZeroTimeDetectedError)
	}

	if p.ProjectID > 0 && p.IssueID > 0 {
		return errors.Join(ValidationError, ProjectAndIssuePassedError)
	}

	if p.ProjectID == 0 && p.IssueID == 0 {
		return errors.Join(ValidationError, ProjectAndIssueMissedError)
	}
	return nil
}
