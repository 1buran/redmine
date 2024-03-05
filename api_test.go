package redmine

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"
)

const (
	// Let's assume that we have 110 entities for every type of Redmine data,
	// in this case (with default pagination limit = 25) we have to create 5 sets with appropriate
	// pagination parameters:
	// - 1st page (?page=1 or without pagination query params in URL):
	//    [0, 25) items, limit - 25, offset - 0, total count - 110
	// - 2nd page (?page=2):
	//    [25, 50) items, limit - 25, offset - 25, total count - 110
	// - 3rd page (?page=3):
	//    [50, 75) items, limit - 25, offset - 50, total count - 110
	// - 4th page (?page=4):
	//    [75, 100) items, limit - 25, offset - 75, total count - 110
	// - 5th page (?page=5):
	//    [100, 110] items, limit - 25, offset - 100, total count - 110
	PaginationLimit = 25
	TotalCount      = 110

	// Here are the samples of responses for different kind of redmine data: projects, issues and
	// time entries:
	ProjectsJSONResponseTpl = `
     {
       "projects": [
       {{- range $i := Iter .First .Last }}
          {
            "id": {{ $i }}, "name": "Project{{ $i }}",
            "description": "Project {{ $i }} Description", "is_public": false,
            "identifier": "Xlab-Project-{{ $i }}", "created_on": "Sat Sep 29 12:03:04 +0200 2007",
            "updated_on": "Sun Mar 15 12:35:11 +0100 2009"
          }{{ if lt $i $.Last }},{{ end }}
        {{- end }}
       ],
       "offset": {{ .Offset }},
       "limit": {{ .Limit }},
       "total_count": {{ .Total }}
     }`

	IssuesJSONResponseTpl = `
     {
       "issues": [
       {{- range $i := Iter .First .Last }}
          {
            "id": {{ $i }}, "subject": "Subject {{ $i }}",
            "description": "Issue {{ $i }} Description",
            "project": {"id": 1, "name": "Project1"}
          }{{ if lt $i $.Last }},{{ end }}
        {{- end }}
       ],
       "offset": {{ .Offset }},
       "limit": {{ .Limit }},
       "total_count": {{ .Total }}
     }`

	TimeEntriesJSONResponseTpl = `
     {
       "time_entries": [
       {{- range $i := Iter .First .Last }}
          {
            "id": {{ $i }}, "comments": "Time Entry {{ $i }} Comment",
            "project": {"id": 1, "name": "Project1"},
            "issue": {"id": {{ $i }}, "subject": "Subject {{ $i }}"},
            "user": {"id": 1, "name": "User1"},
            "hours": 7.35, "spent_on": "2006-01-02"
          }{{ if lt $i $.Last }},{{ end }}
        {{- end }}
       ],
       "offset": {{ .Offset }},
       "limit": {{ .Limit }},
       "total_count": {{ .Total }}
     }`
)

// Redmine API JSON response parameters
type ApiResponseParams struct {
	First  int
	Last   int
	Offset int
	Limit  int
	Total  int
}

// Create an slice of int [a, b]
var funcs = template.FuncMap{
	"Iter": func(a, b int) (res []int) {
		for i := a; i <= b; i++ {
			res = append(res, i)
		}
		return
	},
}

// Generate JSON with the data mimics API JSON response
func GenerateJSON(t string, context any) string {
	var b strings.Builder
	tmpl, err := template.New("test").Funcs(funcs).Parse(t)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(&b, context)
	if err != nil {
		panic(err)
	}
	return b.String()
}

// Test generation of Redmine REST API JSON responses
func TestGenerateJSON(t *testing.T) {
	res := GenerateJSON(ProjectsJSONResponseTpl, ApiResponseParams{1, 25, 0, 25, 25})
	if !strings.Contains(res, `"id": 24, "name": "Project24"`) {
		t.Errorf("unexpected generated JSON: %s", res)
	}
}

// Get the response paginatin settings from the given URL
func GetResponseParamsFromUrl(qs string) *ApiResponseParams {
	p := ApiResponseParams{
		First:  1,
		Last:   PaginationLimit,
		Offset: 0,
		Limit:  PaginationLimit,
		Total:  TotalCount,
	}

	// check if incoming request has pagination params
	if qs != "" {
		v, err := url.ParseQuery(qs)
		if err != nil {
			panic(err)
		}
		page := v.Get("page")
		if page != "" {
			pageNumber, err := strconv.Atoi(page)
			if err != nil {
				panic(err)
			}
			p.Offset = PaginationLimit * (pageNumber - 1)
		}
	}
	p.First = p.Offset + 1
	p.Last = p.Offset + PaginationLimit
	if p.Last > TotalCount {
		p.Last = TotalCount
	}
	return &p
}

// Test scroll over Redmine REST API paginated JSON resposes
func TestScroll(t *testing.T) {
	handleReq := func(w http.ResponseWriter, r *http.Request) {
		var payload string

		params := GetResponseParamsFromUrl(r.URL.RawQuery)

		switch r.URL.Path {
		case ProjectsApiEndpoint:
			payload = GenerateJSON(ProjectsJSONResponseTpl, params)
		case IssuesApiEndpoint:
			payload = GenerateJSON(IssuesJSONResponseTpl, params)
		case TimeEntriesEndpoint:
			payload = GenerateJSON(TimeEntriesJSONResponseTpl, params)
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}
	handler := http.HandlerFunc(handleReq)
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// Actually the filtration is not used in tests, but its needed for apiConfig.
	timeEntriesFilter := TimeEntriesFilter{
		time.Now(),
		time.Now().Add(time.Hour * 24 * 10),
		"1",
	}
	apiConfig := ApiConfig{
		testServer.URL,
		"ababab",
		true,
		timeEntriesFilter,
	}

	// test scrolling of projects
	t.Run("projects", func(t *testing.T) {
		i := 1
		dataChan, _ := Scroll[Project](&apiConfig)
		for p := range dataChan {
			expectedDesc := fmt.Sprintf("Project %d Description", i)
			if p.Desc != expectedDesc {
				t.Errorf("expected %s, got %s", expectedDesc, p.Desc)
			}
			if p.Id != i {
				t.Errorf("expected %d, got %d", i, p.Id)
			}
			i++
		}
		if i-1 != TotalCount {
			t.Errorf("expected %d items, got: %d", TotalCount, i-1)
		}
	})

	// test scrolling of issues
	t.Run("issues", func(t *testing.T) {
		i := 1
		dataChan, _ := Scroll[Issue](&apiConfig)
		for p := range dataChan {
			expectedDesc := fmt.Sprintf("Issue %d Description", i)
			if p.Desc != expectedDesc {
				t.Errorf("expected %s, got %s", expectedDesc, p.Desc)
			}
			if p.Id != i {
				t.Errorf("expected %d, got %d", i, p.Id)
			}
			i++
		}
		if i-1 != TotalCount {
			t.Errorf("expected %d items, got: %d", TotalCount, i-1)
		}
	})

	// test scrolling of time entries
	t.Run("time entries", func(t *testing.T) {
		i := 1
		dataChan, _ := Scroll[TimeEntry](&apiConfig)
		for p := range dataChan {
			expectedDesc := fmt.Sprintf("Time Entry %d Comment", i)
			if p.Comment != expectedDesc {
				t.Errorf("expected %s, got %s", expectedDesc, p.Comment)
			}
			if p.Id != i {
				t.Errorf("expected %d, got %d", i, p.Id)
			}
			i++
		}
		if i-1 != TotalCount {
			t.Errorf("expected %d items, got: %d", TotalCount, i-1)
		}
	})

	// test HTTP 404 Not Found error
	t.Run("404 http error", func(t *testing.T) {
		apiConfig.Url += "/not-found"
		dataChan, errChan := Scroll[Project](&apiConfig)

		select {
		case x := <-dataChan:
			t.Fatalf("expected not found error, got: %v", x)
		case err := <-errChan:
			if !errors.Is(err, JsonDecodeError) {
				t.Fatalf("expected JsonDecodeError, got: %s", err)
			}
			return
		case <-time.After(time.Second * 10):
			t.Fatal("Time out: http server does not respond")
		}
	})

	// test http error
	t.Run("http error", func(t *testing.T) {
		apiConfig.Url = "sd://sdsdsd"
		dataChan, errChan := Scroll[Project](&apiConfig)

		select {
		case x := <-dataChan:
			t.Fatalf("expected not found error, got: %v", x)
		case err := <-errChan:
			if !errors.Is(err, HttpError) {
				t.Fatalf("expected HttpError, got: %s", err)
			}
			return
		case <-time.After(time.Second * 10):
			t.Fatal("Time out: http server does not respond")
		}
	})

	// test malformed Redmine API endpoint url
	t.Run("malformed api endpoint url", func(t *testing.T) {
		apiConfig.Url = "\n"
		dataChan, errChan := Scroll[Project](&apiConfig)

		select {
		case x := <-dataChan:
			t.Fatalf("expected not found error, got: %v", x)
		case err := <-errChan:
			if !errors.Is(err, ApiEndpointUrlFatalError) {
				t.Fatalf("expected ApiEndpointUrlFatalError, got: %v", err)
			}
			return
		case <-time.After(time.Second * 10):
			t.Fatal("Time out: http server does not respond")
		}
	})

}

type fakeReadCloser struct{}

func (f *fakeReadCloser) Read(b []byte) (n int, err error) {
	return 0, errors.New("abort read")
}

func (f *fakeReadCloser) Close() error { return errors.New("abort close") }

func TestDecodeResp(t *testing.T) {
	f := fakeReadCloser{}
	if _, err := DecodeResp[Project](&f); !errors.Is(err, IoReadError) {
		t.Errorf("expected IoReadError, got: %s", err)
	}
}
