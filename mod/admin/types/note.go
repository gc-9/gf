package types

import "time"

type Note struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Title     string    `json:"title" xorm:"'title'"`
	Content   string    `json:"content" xorm:"'content'"`
	Status    int       `json:"status" xorm:"'status'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *Note) TableName() string {
	return "note"
}
