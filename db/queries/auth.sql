-- name: UpsertOTP :exec
INSERT INTO otp_codes (phone, code_hash, expires_at, attempts)
VALUES ($1, $2, $3, 0)
ON CONFLICT (phone) DO UPDATE
  SET code_hash = $2, expires_at = $3, attempts = 0;

-- name: GetOTP :one
SELECT code_hash, expires_at, attempts FROM otp_codes WHERE phone = $1;

-- name: IncrementOTPAttempts :exec
UPDATE otp_codes SET attempts = attempts + 1 WHERE phone = $1;

-- name: DeleteOTP :exec
DELETE FROM otp_codes WHERE phone = $1;
