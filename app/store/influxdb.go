package store

import (
	"apns_feedback_service/app/feedback"
	"apns_feedback_service/app/logger"
	"errors"
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"time"
)

type Influxdb struct {
	client   client.Client
	addr     string
	dbName   string
	userName string
	password string
}

func NewInfluxdb(addr, dbName, userName, password string) *Influxdb {
	return &Influxdb{
		addr:     addr,
		dbName:   dbName,
		userName: userName,
		password: password,
	}
}

func (c *Influxdb) Connection() error {
	conn, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     c.addr,
		Username: c.userName,
		Password: c.password,
	})
	if err != nil {
		return err
	}
	c.client = conn
	return nil
}

func (c *Influxdb) Disconnection() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Influxdb) Store(resp feedback.FeedbackResponse) error {
	if c.client == nil {
		return errors.New("请先连接 influxdb.....")
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  c.dbName,
		Precision: "s",
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Influxdb NewBatchPoints error: %s", err.Error()))
		return err
	}
	sandbox := "0"
	if resp.IsSandbox {
		sandbox = "1"
	}
	tags := map[string]string{
		"bundle_id": resp.BundleId,
		"sandbox":   sandbox,
	}
	tm := time.Unix(int64(resp.Timestamp), 0)

	fields := map[string]interface{}{
		"invalid_time": tm.Format("2006-01-02 15:04:05"),
		"token":        resp.DeviceToken,
	}

	pt, err := client.NewPoint(
		"invalid_tokens",
		tags,
		fields,
		tm,
	)
	if err != nil {
		logger.Error(fmt.Sprintf("Influxdb NewPoint error: %s", err.Error()))
		return err
	}
	bp.AddPoint(pt)
	if err = c.client.Write(bp); err != nil {
		logger.Error(fmt.Sprintf("Influxdb Write error: %s", err.Error()))
		return err
	}
	return nil
}

func (_ *Influxdb) Name() string {
	return "Influxdb"
}
