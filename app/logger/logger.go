package logger

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)
}

func Debug(msg string) {
	log.Debug(msg)
}

func DebugWithFields(fields map[string]interface{}, msg string) {
	log.WithFields(fields).Debug(msg)
}

func Error(msg string) {
	log.Error(msg)
}

func Info(msg string) {
	log.Info(msg)
}

func InfoFeedbackResponse(resp interface{}) {
	if resp == nil {
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		Error(err.Error())
		return
	}
	var fields map[string]interface{}
	err = json.Unmarshal(data, &fields)
	if err != nil {
		Error(err.Error())
		return
	}
	log.WithFields(fields).Info("An invalid token was received")
}
