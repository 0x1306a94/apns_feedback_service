package model

import "time"

// Model struct
type TokenRecord struct {
	Id          uint32
	TimeKey     string    `orm:"column(time_key)"`
	InvalidTime time.Time `orm:"column(invalid_time)"`
	TokenLength uint16    `orm:"column(token_length)"`
	Token       string    `orm:"column(token)"`
	BundleId    string    `orm:"column(bundle_id)"`
	Sandbox     uint      `orm:"column(sand_box)"`
}

func (u *TokenRecord) TableName() string {
	return "token_record_tab"
}
