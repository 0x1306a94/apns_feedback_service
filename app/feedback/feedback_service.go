package feedback

import (
	"apns_feedback_service/app/certificate"
	"apns_feedback_service/app/logger"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	hostSandbox                = "feedback.sandbox.push.apple.com:2196"
	hostProduction             = "feedback.push.apple.com:2196"
	feedbackTimeoutSeconds     = 30
	feedbackResponsePackLength = 38
	feedbackTokenLength        = 32
)

type FeedbackService struct {
	cerInfo    *certificate.CertificateInfo
	isSandbox  bool
	resultChan chan FeedbackResponse
	stopChan   chan struct{}
	stop       bool
	gateway    string
}

func NewFeedbackService(cerInfo *certificate.CertificateInfo) (*FeedbackService, error) {
	if cerInfo == nil {
		return nil, errors.New("cerInfo 不能为空")
	}
	gateway := hostProduction
	if cerInfo.IsSandbox() {
		gateway = hostSandbox
	}
	service := &FeedbackService{
		cerInfo:    cerInfo,
		isSandbox:  cerInfo.IsSandbox(),
		resultChan: make(chan FeedbackResponse, 1000),
		stopChan:   make(chan struct{}),
		stop:       false,
		gateway:    gateway,
	}
	return service, nil
}

func (f *FeedbackService) Start(mock bool) {
	go func() {
		if mock {
			ticker := time.NewTicker(time.Second * 10)
			for {
				select {
				case <-ticker.C:
					count := rand.Intn(500) + 50
					timestamp := uint32(time.Now().Unix())
					for i := 0; i < count; i++ {
						uid := uuid.New().String()
						resp := FeedbackResponse{
							Timestamp:   timestamp,
							TokenLength: 2,
							DeviceToken: uid,
							BundleId:    string(f.cerInfo.GetBundleId()),
							IsSandbox:   f.cerInfo.IsSandbox(),
						}
						f.resultChan <- resp
					}
				case <-f.stopChan:
					f.stop = true
					logger.DebugWithFields(map[string]interface{}{
						"bundle_id": f.cerInfo.GetBundleId(),
						"sandbox":   f.cerInfo.IsSandbox(),
					}, "feedbackService stop ....")
					break
				}
			}
		} else {
			for {
				select {
				case <-f.stopChan:
					f.stop = true
					logger.DebugWithFields(map[string]interface{}{
						"bundle_id": f.cerInfo.GetBundleId(),
						"sandbox":   f.cerInfo.IsSandbox(),
					}, "feedbackService stop ....")
					break
				default:
					if f.stop {
						return
					}
					logger.DebugWithFields(map[string]interface{}{
						"bundle_id": f.cerInfo.GetBundleId(),
						"sandbox":   f.cerInfo.IsSandbox(),
					}, fmt.Sprintf("------------------ %s ------------------", time.Now().Format("2006-01-02 15:04:05")))
					f.fetchInvalidToken()
					logger.DebugWithFields(map[string]interface{}{
						"bundle_id": f.cerInfo.GetBundleId(),
						"sandbox":   f.cerInfo.IsSandbox(),
					}, fmt.Sprintf("sleep 5 minute: %v", time.Now().Format("2006-01-02 15:04:05")))

					select {
					case <-time.After(time.Minute * 5):
					}
					//time.Sleep(time.Minute * 5)
					logger.DebugWithFields(map[string]interface{}{
						"bundle_id": f.cerInfo.GetBundleId(),
						"sandbox":   f.cerInfo.IsSandbox(),
					}, "end sleep 5 minute .....")
				}
			}
		}
	}()
}

func (f *FeedbackService) Stop() {
	go func() {
		f.stopChan <- struct{}{}
		close(f.resultChan)
	}()
}

func (f *FeedbackService) Result() <-chan FeedbackResponse {
	return f.resultChan
}

func (f *FeedbackService) fetchInvalidToken() {
	gatewayParts := strings.Split(f.gateway, ":")
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{f.cerInfo.GetCertificate()},
		ServerName:   gatewayParts[0],
	}
	conn, err := net.Dial("tcp", f.gateway)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(feedbackTimeoutSeconds * time.Second))
	tlsConn := tls.Client(conn, tlsCfg)
	err = tlsConn.Handshake()
	if err != nil {
		logger.Error(err.Error())
		return
	}
	buffer := make([]byte, feedbackResponsePackLength, feedbackResponsePackLength)
	deviceToken := make([]byte, feedbackTokenLength, feedbackTokenLength)
	count := 0
	for {
		readLength, err := tlsConn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				logger.Error(err.Error())
			}
			break
		}
		if readLength != feedbackResponsePackLength {
			break
		}
		count++
		resp := FeedbackResponse{}
		r := bytes.NewReader(buffer)
		binary.Read(r, binary.BigEndian, &resp.Timestamp)
		binary.Read(r, binary.BigEndian, &resp.TokenLength)
		binary.Read(r, binary.BigEndian, &deviceToken)
		if resp.TokenLength != feedbackTokenLength {
			logger.DebugWithFields(map[string]interface{}{
				"bundle_id": f.cerInfo.GetBundleId(),
				"sandbox":   f.cerInfo.IsSandbox(),
			}, "token length should be equal to 32, but isn't")
			break
		}
		resp.DeviceToken = hex.EncodeToString(deviceToken)
		resp.BundleId = string(f.cerInfo.GetBundleId())
		resp.IsSandbox = f.cerInfo.IsSandbox()
		logger.InfoFeedbackResponse(resp)
		select {
		case f.resultChan <- resp:
		default:
		}
	}
	logger.DebugWithFields(map[string]interface{}{
		"bundle_id": f.cerInfo.GetBundleId(),
		"sandbox":   f.cerInfo.IsSandbox(),
	}, fmt.Sprintf("本次 收到 %v 条无效 token\n", count))
}
