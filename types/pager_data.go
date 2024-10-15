package types

type Pager struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type PagerData[V any] struct {
	List  []V    `json:"list"`
	Pager *Pager `json:"pager,omitempty"`
}

func EmptyPagerData[V any]() *PagerData[V] {
	return &PagerData[V]{
		List:  make([]V, 0),
		Pager: &Pager{},
	}
}
