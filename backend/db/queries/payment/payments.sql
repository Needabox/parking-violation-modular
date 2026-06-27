-- name: CreatePayment :one
INSERT INTO payments (violation_id, amount, status, payment_method, reference_id, error_message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, violation_id, amount, status, payment_method, reference_id, error_message, created_at, updated_at;

-- name: GetPaymentByID :one
SELECT id, violation_id, amount, status, payment_method, reference_id, error_message, created_at, updated_at
FROM payments
WHERE id = $1;

-- name: ListPaymentsByViolationID :many
SELECT id, violation_id, amount, status, payment_method, reference_id, error_message, created_at, updated_at
FROM payments
WHERE violation_id = $1
ORDER BY created_at DESC;
