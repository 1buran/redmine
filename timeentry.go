package redmine

import (
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
