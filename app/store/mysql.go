package store

import (
	"apns_feedback_service/app/feedback"
	"apns_feedback_service/app/model"
	"fmt"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type MysqlStore struct {
	o orm.Ormer
}

func NewMysqlStore(name string) *MysqlStore {
	o := orm.NewOrm()
	o.Using(name)
	return &MysqlStore{
		o: o,
	}
}

func (m *MysqlStore) Connection() error {

	return nil
}

func (m *MysqlStore) Disconnection() error {

	return nil
}

func (m *MysqlStore) Store(resp feedback.FeedbackResponse) error {
	tm := time.Unix(int64(resp.Timestamp), 0)
	timeKey := fmt.Sprintf("%v", resp.Timestamp)
	var sandbox uint = 1
	if resp.IsSandbox {
		sandbox = 0
	}
	token := model.TokenRecord{
		TimeKey:     timeKey,
		InvalidTime: tm,
		TokenLength: resp.TokenLength,
		Token:       resp.DeviceToken,
		BundleId:    resp.BundleId,
		Sandbox:     sandbox,
	}
	_, err := m.o.Insert(&token)
	return err
}

func (_ *MysqlStore) Name() string {
	return "MySQL"
}
