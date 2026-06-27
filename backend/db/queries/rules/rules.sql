-- name: CreateRule :one
INSERT INTO fine_rules (code, name, description, fine_amount, version, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, name, description, fine_amount, version, is_active, created_at, updated_at;

-- name: DeactivateRuleVersions :exec
UPDATE fine_rules
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1 AND is_active = TRUE;

-- name: GetActiveRuleByCode :one
SELECT id, code, name, description, fine_amount, version, is_active, created_at, updated_at
FROM fine_rules
WHERE code = $1 AND is_active = TRUE;

-- name: GetRuleByID :one
SELECT id, code, name, description, fine_amount, version, is_active, created_at, updated_at
FROM fine_rules
WHERE id = $1;

-- name: GetRuleByCodeAndVersion :one
SELECT id, code, name, description, fine_amount, version, is_active, created_at, updated_at
FROM fine_rules
WHERE code = $1 AND version = $2;

-- name: ListActiveRules :many
SELECT id, code, name, description, fine_amount, version, is_active, created_at, updated_at
FROM fine_rules
WHERE is_active = TRUE
ORDER BY code ASC;

-- name: ListRuleVersions :many
SELECT id, code, name, description, fine_amount, version, is_active, created_at, updated_at
FROM fine_rules
WHERE code = $1
ORDER BY version DESC;
