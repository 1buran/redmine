package redmine

type Pagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Total  int `json:"total_count"`
}

func (p Pagination) NextPage() (n int) {
	if p.Total-p.Offset < p.Limit {
		return -1
	}
	if p.Limit > 0 {
		n = (p.Offset+p.Limit)/p.Limit + 1
	}
	return
}
