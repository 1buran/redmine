package redmine

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

// A date type is needed for proper parsing (unmarshaling) of redmine date format used in JSON.
type Date struct {
	time.Time
}

// Unmarshaling redmine dates.
func (d *Date) UnmarshalJSON(b []byte) error {
	t, err := time.Parse("2006-01-02", string(bytes.Trim(b, "\"")))
	if err != nil {
		return errors.Join(JsonDecodeError, err)
	}
	d.Time = t
	return nil
}

// Marshaling time.Time object to redmine format.
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", d.Time.Format("2006-01-02"))), nil
}

func (d Date) String() string {
	return d.Time.Format("2006-01-02")
}
