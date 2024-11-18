package redmine

// A Redmine user entity.
type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
