package redmine

import (
	"bytes"
	"errors"
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

func (d Date) String() string {
	return d.Time.Format("2006-01-02")
}
