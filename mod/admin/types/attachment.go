package types

import "time"

type Attachment struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	UID       int       `json:"uid" xorm:"uid"`
	Filename  string    `json:"filename" xorm:"filename"`
	Path      string    `json:"path" xorm:"path"`
	Driver    string    `json:"driver" xorm:"driver"`
	Size      int       `json:"size" xorm:"size"`
	Attr      string    `json:"attr" xorm:"attr"`
	Ext       string    `json:"ext" xorm:"ext"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *Attachment) TableName() string {
	return "attachment"
}
