package certificate

import (
	"crypto/tls"
	"github.com/0x1306a94/go-apns/cer"
	"strings"
	"time"
)

type CertificateInfo struct {
	cer            tls.Certificate
	expirationTime time.Time
	bundleId       string
	isSandbox      bool
}

func NewCertificateInfo(path, password string) (*CertificateInfo, error) {
	cer, err := cert.FromP12File(path, password)
	if err != nil {
		return nil, err
	}

	//BundleId := ""
	//if cer.Leaf.Subject.Names != nil && len(cer.Leaf.Subject.Names) > 0 {
	//	BundleId = strings.Trim(cer.Leaf.Subject.Names[0].Value.(string), "")
	//} else if cer.Leaf.Subject.CommonName != "" && len(cer.Leaf.Subject.CommonName) > 0 {
	//	commonNames := strings.Split(cer.Leaf.Subject.CommonName, ":")
	//	BundleId = strings.Trim(commonNames[1], " ")
	//}
	commonNames := strings.Split(cer.Leaf.Subject.CommonName, ":")
	BundleId := strings.Trim(commonNames[1], " ")
	isSandbox := false
	if strings.Contains(commonNames[0], "Apple Development") {
		isSandbox = true
	}
	return &CertificateInfo{
		cer:            cer,
		expirationTime: cer.Leaf.NotAfter,
		bundleId:       BundleId,
		isSandbox:      isSandbox,
	}, nil
}

func (c *CertificateInfo) GetCertificate() tls.Certificate {
	return c.cer
}

func (c *CertificateInfo) GetBundleId() string {
	return c.bundleId
}

func (c *CertificateInfo) IsSandbox() bool {
	return c.isSandbox
}

func (c *CertificateInfo) IsExpiration() bool {
	if c.bundleId == "" {
		return false
	}
	return c.expirationTime.Sub(time.Now()) <= 0
}
