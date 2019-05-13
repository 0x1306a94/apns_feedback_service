package main

import (
	"apns_feedback_service/app/certificate"
	"apns_feedback_service/app/feedback"
	"apns_feedback_service/app/logger"
	"apns_feedback_service/app/model"
	"apns_feedback_service/app/store"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/astaxie/beego/orm"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	configPath string
	mock       bool
)
var (
	feedbackServices []*feedback.FeedbackService    = make([]*feedback.FeedbackService, 0)
	storeServices    []store.Storeable              = make([]store.Storeable, 0)
	storeChan        chan feedback.FeedbackResponse = make(chan feedback.FeedbackResponse, 1000)
	config           *Config
)


type Config struct {
	Certificate []CertificateConfig `json:"certificate"`
	Mysql       MySQLConfig         `json:"mysql,omitempty"`
	MockMysql   MySQLConfig         `json:"mock_mysql,omitempty"`
	InfluxDB    InfluxdbConfig      `json:"influx_db,omitempty"`
}

type CertificateConfig struct {
	Path     string `json:"path"`
	Password string `json:"password"`
}

type MySQLConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	DB       string `json:"db"`
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
}

type InfluxdbConfig struct {
	Addr     string `json:"addr"`
	DBName   string `json:"db_name"`
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	if err = json.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func main() {

	// 设置时区
	time.LoadLocation("Asia/Chongqing")
	flag.StringVar(&configPath, "config", "/Users/king/WorkSpec/Go/src/apns_feedback_service/app/config.json", "配置文件路径")
	flag.BoolVar(&mock, "mock", false, "生成测试数据")
	flag.Parse()

	if configPath == "" {
		flag.Usage()
		os.Exit(0)
	}

	conf, err := loadConfig(configPath)
	if err != nil {
		logger.Error(fmt.Sprintf("loadConfig error: %v", err))
		os.Exit(-1)
	}
	config = conf

	for _, val := range config.Certificate {
		path, err := filepath.Abs(val.Path)
		logger.Debug(path)
		if err != nil {
			logger.Error(fmt.Sprintf("加载证书错误: %v", err))
			os.Exit(-1)
			continue
		}
		certInfo, err := certificate.NewCertificateInfo(path, val.Password)
		if err != nil {
			logger.Error(fmt.Sprintf("加载证书错误: %v", err))
			os.Exit(-1)
			continue
		}
		if certInfo.IsExpiration() {
			logger.Error(fmt.Sprintf("%s 证书已过期", path))
			continue
		}
		services, _ := feedback.NewFeedbackService(certInfo)
		feedbackServices = append(feedbackServices, services)
	}

	if len(feedbackServices) == 0 {
		logger.Debug("没有 feedback services")
		os.Exit(0)
	}

	// register model
	//orm.Debug = true
	// 参数4(可选)  设置最大空闲连接
	// 参数5(可选)  设置最大数据库连接 (go >= 1.2)
	maxIdle := 30
	maxConn := 30
	if config.Mysql.Host != "" && config.Mysql.Port != "" && config.Mysql.DB != "" && config.Mysql.User != "" {
		dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			config.Mysql.User,
			config.Mysql.Password,
			config.Mysql.Host,
			config.Mysql.Port,
			config.Mysql.DB)

		orm.RegisterDataBase("default", "mysql", dataSource, maxIdle, maxConn)
		orm.RunSyncdb("default", false, true)
		storeServices = append(storeServices, store.NewMysqlStore("default"))

		orm.RegisterDriver("mysql", orm.DRMySQL)
		orm.RegisterModel(new(model.TokenRecord))
	}

	if config.MockMysql.Host != "" && config.MockMysql.Port != "" && config.MockMysql.DB != "" && config.MockMysql.User != "" {
		mockDataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			config.MockMysql.User,
			config.MockMysql.Password,
			config.MockMysql.Host,
			config.MockMysql.Port,
			config.MockMysql.DB)
		orm.RegisterDataBase("mock", "mysql", mockDataSource, maxIdle, maxConn)
		orm.RunSyncdb("mock", false, true)

		storeServices = append(storeServices, store.NewMysqlStore("mock"))

	} else if mock {
		logger.Error("请配置 mock mysql")
		os.Exit(-1)
	}

	if config.InfluxDB.Addr != "" && config.InfluxDB.DBName != "" && config.InfluxDB.UserName != "" && config.InfluxDB.Password != "" {
		influxdbService := store.NewInfluxdb(
			config.InfluxDB.Addr,
			config.InfluxDB.DBName,
			config.InfluxDB.UserName,
			config.InfluxDB.Password,
		)
		if err := influxdbService.Connection(); err != nil {
			logger.Error(fmt.Sprintf("连接InfluxDB 错误: %s", err.Error()))
			os.Exit(-1)
		}
		storeServices = append(storeServices, influxdbService)
	}

	logger.DebugWithFields(map[string]interface{}{
		"pid": os.Getpid(),
	}, "启动....")

	for _, service := range feedbackServices {
		service.Start(mock)
		go startFetch(service, storeChan)
	}
	go startStore(storeChan)


	c := make(chan os.Signal, 1)
	//监听指定信号 ctrl+c kill
	logger.Debug("监听指定信")
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		close(storeChan)
	for _, service := range feedbackServices {
		service.Stop()
	}

	for _, service := range storeServices {
		service.Disconnection()
	}

	logger.Debug("退出....")
}

func startFetch(service *feedback.FeedbackService, ch chan<- feedback.FeedbackResponse) {
	for resp := range service.Result() {
		select {
		case ch <- resp:
		default:

		}
	}
}

func startStore(ch <-chan feedback.FeedbackResponse) {
	for resp := range ch {
		for _, val := range storeServices {
			service := val
			go func(service store.Storeable) {
				if err := service.Store(resp); err != nil {
					logger.Error(fmt.Sprintf("写入 %s 出错: %s", service.Name(), err.Error()))
				}
			}(service)
		}
	}
}
