package redmine

import (
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
			w.Write([]byte(`{}`))
		case TimeEntriesEndpoint:
			w.Write([]byte(`{}`))
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

	i := 1
	for p := range Scroll[Project](&apiConfig) {
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
}
