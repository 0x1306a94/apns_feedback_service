package feedback

import (
	"fmt"
	"time"
)

type FeedbackResponse struct {
	Timestamp   uint32
	TokenLength uint16
	DeviceToken string
	BundleId    string
	IsSandbox   bool
}

func (f FeedbackResponse) String() string {
	tm := time.Unix(int64(f.Timestamp), 0)
	return fmt.Sprintf("Timestamp: %v time: %v\nTokenLength: %v\nDeviceToken: %v\nBundleId: %v\nIsSandbox: %v",
		f.Timestamp,
		tm.Format("2006-01-02 15:04:05"),
		f.TokenLength,
		f.DeviceToken,
		f.BundleId,
		f.IsSandbox)
}
