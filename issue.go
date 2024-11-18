package redmine

import (
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
