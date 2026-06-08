// Package clients · S3-compatible presigned URL generator.
//
// We sign PUT URLs server-side and hand them to the mobile client, which
// uploads images directly to object storage without proxying through us.
// The same SigV4 signer drives both Cloudflare R2 and Supabase Storage —
// they differ only in host, region, and URL path prefix.
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

// S3Presigner signs PUT requests against any S3-compatible endpoint.
//
//	Host   — virtual host, e.g. "{ref}.storage.supabase.co".
//	Region — the project region for Supabase.
//	Prefix — path segment before /{bucket}/{key};
//	         "/storage/v1/s3" for Supabase's S3 protocol endpoint.
type S3Presigner struct {
	Host            string
	Region          string
	Prefix          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBase      string
}

// NewSupabaseStorage builds a presigner for Supabase Storage's S3-compatible
// endpoint. When publicBase is empty it defaults to the bucket's public
// object URL (valid only for buckets marked public in Supabase).
func NewSupabaseStorage(projectRef, region, ak, sk, bucket, publicBase string) *S3Presigner {
	host := fmt.Sprintf("%s.storage.supabase.co", projectRef)
	if publicBase == "" {
		publicBase = fmt.Sprintf("https://%s/storage/v1/object/public/%s", host, bucket)
	}
	return &S3Presigner{
		Host:            host,
		Region:          region,
		Prefix:          "/storage/v1/s3",
		AccessKeyID:     ak,
		SecretAccessKey: sk,
		Bucket:          bucket,
		PublicBase:      publicBase,
	}
}

// PresignPUT returns a 5-minute upload URL for the given key, plus the
// publicly accessible URL after upload completes.
//
// Implementation: SigV4 signed PUT (path-style) against the configured host.
func (r *S3Presigner) PresignPUT(ctx context.Context, key, contentType string) (uploadURL, publicURL string, err error) {
	host := r.Host
	canonicalURI := fmt.Sprintf("%s/%s/%s", r.Prefix, r.Bucket, key)
	endpoint := "https://" + host + canonicalURI

	now := time.Now().UTC()
	ts := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	region := r.Region
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
