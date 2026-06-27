-- name: CreateViolation :one
INSERT INTO violations (fine_rule_id, plate_number, officer_id, status, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, fine_rule_id, plate_number, officer_id, status, description, created_at, updated_at;

-- name: GetViolationByID :one
SELECT 
    v.id, v.fine_rule_id, v.plate_number, v.officer_id, v.status, v.description, v.created_at, v.updated_at,
    r.code AS rule_code, r.name AS rule_name, r.description AS rule_description, r.fine_amount, r.version AS rule_version
FROM violations v
JOIN fine_rules r ON v.fine_rule_id = r.id
WHERE v.id = $1;

-- name: ListAllViolations :many
SELECT 
    v.id, v.fine_rule_id, v.plate_number, v.officer_id, v.status, v.description, v.created_at, v.updated_at,
    r.code AS rule_code, r.name AS rule_name, r.description AS rule_description, r.fine_amount, r.version AS rule_version
FROM violations v
JOIN fine_rules r ON v.fine_rule_id = r.id
ORDER BY v.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllViolations :one
SELECT COUNT(*)::bigint FROM violations;

-- name: ListViolationsByPlates :many
SELECT 
    v.id, v.fine_rule_id, v.plate_number, v.officer_id, v.status, v.description, v.created_at, v.updated_at,
    r.code AS rule_code, r.name AS rule_name, r.description AS rule_description, r.fine_amount, r.version AS rule_version
FROM violations v
JOIN fine_rules r ON v.fine_rule_id = r.id
WHERE v.plate_number = ANY($1::VARCHAR[])
ORDER BY v.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountViolationsByPlates :one
SELECT COUNT(*)::bigint 
FROM violations
WHERE plate_number = ANY($1::VARCHAR[]);

-- name: UpdateViolationStatus :one
UPDATE violations
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, fine_rule_id, plate_number, officer_id, status, description, created_at, updated_at;
