// Package clients · Cloudflare R2 (S3-compatible) presigned URL generator.
//
// We sign PUT URLs server-side and hand them to the mobile client, which
// uploads images directly to R2 without proxying through us. Egress is
// free on R2.
package clients

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type R2 struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBase      string
}

func NewR2(accountID, ak, sk, bucket, publicBase string) *R2 {
	return &R2{accountID, ak, sk, bucket, publicBase}
}

// PresignPUT returns a 5-minute upload URL for the given key, plus the
// publicly accessible URL after upload completes.
//
// Implementation: SigV4 signed PUT against r2.cloudflarestorage.com.
func (r *R2) PresignPUT(ctx context.Context, key, contentType string) (uploadURL, publicURL string, err error) {
	host := fmt.Sprintf("%s.r2.cloudflarestorage.com", r.AccountID)
	endpoint := fmt.Sprintf("https://%s/%s/%s", host, r.Bucket, key)

	now := time.Now().UTC()
	ts := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	region := "auto"
	service := "s3"
	credScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, region, service)

	q := url.Values{}
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", fmt.Sprintf("%s/%s", r.AccessKeyID, credScope))
	q.Set("X-Amz-Date", ts)
	q.Set("X-Amz-Expires", "300")
	q.Set("X-Amz-SignedHeaders", "host")
	if contentType != "" {
		q.Set("response-content-type", contentType)
	}

	canonicalURI := fmt.Sprintf("/%s/%s", r.Bucket, key)
	canonicalQuery := q.Encode()
	canonicalHeaders := fmt.Sprintf("host:%s\n", host)

	canonicalReq := strings.Join([]string{
		"PUT", canonicalURI, canonicalQuery,
		canonicalHeaders, "host", "UNSIGNED-PAYLOAD",
	}, "\n")
	hashedReq := sha256hex([]byte(canonicalReq))

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256", ts, credScope, hashedReq,
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+r.SecretAccessKey), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	q.Set("X-Amz-Signature", signature)
	uploadURL = endpoint + "?" + q.Encode()

	if r.PublicBase != "" {
		publicURL = r.PublicBase + "/" + key
	} else {
		publicURL = endpoint
	}
	return uploadURL, publicURL, nil
}

func sha256hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}
