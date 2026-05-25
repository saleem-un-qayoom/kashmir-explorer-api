-- name: CreateBooking :one
INSERT INTO bookings (ref, user_id, provider_id, start_date, end_date, guests,
                     base_inr, gst_inr, fee_inr, total_inr, status,
                     razorpay_order_id, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11, $12)
RETURNING id;

-- name: ListBookings :many
SELECT b.id, b.ref, b.start_date, b.end_date, b.guests, b.total_inr, b.status,
       p.name AS provider_name, p.type AS provider_type, p.phone AS provider_phone
FROM bookings b
JOIN providers p ON p.id = b.provider_id
WHERE b.user_id = $1
ORDER BY b.start_date DESC;

-- name: GetBooking :one
SELECT b.id, b.ref, b.start_date, b.end_date, b.guests,
       b.base_inr, b.gst_inr, b.fee_inr, b.total_inr, b.status, b.notes,
       p.name AS provider_name, p.type AS provider_type, p.phone AS provider_phone
FROM bookings b
JOIN providers p ON p.id = b.provider_id
WHERE b.id = $1 AND b.user_id = $2;

-- name: CancelBooking :exec
UPDATE bookings
SET status = 'cancelled', updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: ConfirmBookingFromWebhook :exec
UPDATE bookings
SET status = 'confirmed', razorpay_payment_id = $1, updated_at = now()
WHERE razorpay_order_id = $2;

-- name: FailBookingFromWebhook :exec
UPDATE bookings
SET status = 'cancelled', updated_at = now()
WHERE razorpay_order_id = $1 AND status = 'pending';
