package redmine

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestPostTimeEntryParams(t *testing.T) {
	t.Parallel()

	t.Run("Validate/ok", func(t *testing.T) {
		params := NewPostTimeEntryParams()
		params.Payload.ActivityID = 9
		params.Payload.Comments = "debug"
		params.Payload.Hours = 1.5
		params.Payload.UserID = 112
		params.Payload.IssueID = 156
		params.Payload.SpentOn = Date{time.Now()}

		if err := params.Validate(); err != nil {
			t.Error(err)
		}
	})

	t.Run("Validate/project and issue are missed", func(t *testing.T) {
		params := NewPostTimeEntryParams()
		params.Payload.ActivityID = 9
		params.Payload.Comments = "debug"
		params.Payload.Hours = 1.5
		params.Payload.UserID = 112
		params.Payload.SpentOn = Date{time.Now()}

		err := params.Validate()
		if err == nil {
			t.Errorf("expected %q, got: %q", ValidationError, err)
		} else {
			if !errors.Is(err, ProjectAndIssueMissedError) {
				t.Errorf("expected %q, got: %q", ProjectAndIssueMissedError, err)
			}
		}
	})

	t.Run("Validate/project and issue are passed", func(t *testing.T) {
		params := NewPostTimeEntryParams()
		params.Payload.ActivityID = 9
		params.Payload.Comments = "debug"
		params.Payload.Hours = 1.5
		params.Payload.UserID = 112
		params.Payload.ProjectID = 101
		params.Payload.IssueID = 32
		params.Payload.SpentOn = Date{time.Now()}

		err := params.Validate()
		if err == nil {
			t.Errorf("expected %q, got: %q", ValidationError, err)
		} else {
			if !errors.Is(err, ProjectAndIssuePassedError) {
				t.Errorf("expected %q, got: %q", ProjectAndIssuePassedError, err)
			}
		}
	})

	t.Run("Validate/zero time", func(t *testing.T) {
		params := NewPostTimeEntryParams()
		params.Payload.ActivityID = 9
		params.Payload.Comments = "debug"
		params.Payload.Hours = 1.5
		params.Payload.UserID = 112
		params.Payload.ProjectID = 101
		params.Payload.SpentOn = Date{time.Time{}}

		err := params.Validate()
		if err == nil {
			t.Errorf("expected %q, got: %q", ValidationError, err)
		} else {
			if !errors.Is(err, ZeroTimeDetectedError) {
				t.Errorf("expected %q, got: %q", ZeroTimeDetectedError, err)
			}
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		params := NewPostTimeEntryParams()
		params.Payload.ActivityID = 9
		params.Payload.Comments = "debug"
		params.Payload.Hours = 1.5
		params.Payload.UserID = 112
		params.Payload.IssueID = 156
		params.Payload.SpentOn = Date{time.Date(2024, time.November, 1, 0, 0, 0, 0, time.UTC)}

		b, err := json.Marshal(params)
		if err != nil {
			t.Error(err)
		}

		t.Log(string(b))

		if bytes.Contains(b, []byte("project_id")) {
			t.Error("should not contains project_id")
		}

		if !bytes.Contains(b, []byte(`"activity_id":9`)) {
			t.Error(`Not found "activity_id":9, params:`, string(b))
		}

		if !bytes.Contains(b, []byte(`"comments":"debug"`)) {
			t.Error(`Not found "comments":"debug", params:`, string(b))
		}

		if !bytes.Contains(b, []byte(`"hours":1.5`)) {
			t.Error(`Not found "hours":1.5, params:`, string(b))
		}

		if !bytes.Contains(b, []byte(`"user_id":112`)) {
			t.Error(`Not found "user_id":112, params:`, string(b))
		}

		if !bytes.Contains(b, []byte(`"issue_id":156`)) {
			t.Error(`Not found "issue_id":156, params:`, string(b))
		}

		if !bytes.Contains(b, []byte(`"spent_on":"2024-11-01"`)) {
			t.Error(`Not found "spent_on":"2024-11-01", params:`, string(b))
		}
	})

	// todo
	t.Run("issue", func(t *testing.T) {
		params := NewPostIssueParams()
		params.Payload.ProjectID = 101
		params.Payload.TrackerID = 9
		params.Payload.StatusID = 1
		params.Payload.Subject = "issue-1"
		params.Payload.Desc = "test issue"

		b, err := json.Marshal(params)
		if err != nil {
			t.Error(err)
		}
		t.Log(string(b))
	})
}
