package redmine

// A Redmine project entity.
type Project struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Ident string `json:"identifier"`
	Desc  string `json:"description"`
	// TODO correct parsing date time
	// CreatedOn time.Time `json:"created_on"`
	// UpdatedOn time.Time `json:"updated_on"`
	IsPublic bool `json:"is_public"`
}

type Projects struct {
	Items []Project `json:"projects"`
	Pagination
}
