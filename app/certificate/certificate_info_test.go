package certificate

import (
	"fmt"
	"testing"
)

func TestNewCertificateInfo(t *testing.T) {

	cer, err := NewCertificateInfo("/Users/king/Go/src/apns_feedback_service/app/apns.p12", "123")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(cer)
}
