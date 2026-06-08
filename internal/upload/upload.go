// Package upload — presigned upload URLs for the mobile app + admin.
package upload

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/internal/config"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	store *clients.S3Presigner
}

func NewService(cfg config.SupabaseConfig) *Service {
	return &Service{store: clients.NewSupabaseStorage(cfg.ProjectRef, cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Bucket, cfg.PublicBase)}
}

type presignReq struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

// Upload doc-models (OpenAPI/codegen).
type PresignInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}
type PresignResult struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Key       string `json:"key"`
}

// Presign godoc
// @Summary  Get a presigned upload URL
// @Tags     upload
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body upload.PresignInput true "File"
// @Success  200 {object} response.Envelope{data=upload.PresignResult}
// @Failure  400 {object} response.Envelope
// @Router   /v1/upload/presign [post]
func (s *Service) Presign(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var body presignReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Filename == "" {
		response.BadRequest(w, "filename required")
		return
	}
	if body.ContentType == "" {
		body.ContentType = "image/jpeg"
	}
	if !allowedType(body.ContentType) {
		response.BadRequest(w, "content type not allowed")
		return
	}

	// Key: uploads/{user-or-anon}/{ts}-{rand}-{cleanname}
	key := buildKey(userID, body.Filename)

	upload, public, err := s.store.PresignPUT(r.Context(), key, body.ContentType)
	if err != nil {
		response.Internal(w, err)
		return
	}

	response.OK(w, map[string]any{
		"upload_url": upload,
		"public_url": public,
		"key":        key,
		"expires_in": 300,
	})
}

func allowedType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "image/avif", "image/heic":
		return true
	}
	return false
}

func buildKey(userID, filename string) string {
	rand4 := make([]byte, 4)
	_, _ = rand.Read(rand4)
	ts := time.Now().UTC().Format("20060102")
	clean := path.Base(filename)
	clean = strings.ReplaceAll(clean, " ", "-")
	clean = strings.ToLower(clean)
	owner := "anon"
	if userID != "" {
		owner = userID
	}
	return "uploads/" + owner + "/" + ts + "-" + hex.EncodeToString(rand4) + "-" + clean
}
