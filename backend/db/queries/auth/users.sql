-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, role, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateVehicle :one
INSERT INTO vehicles (user_id, plate_number)
VALUES ($1, $2)
RETURNING id, user_id, plate_number, created_at;

-- name: ListVehiclesByUserID :many
SELECT id, user_id, plate_number, created_at
FROM vehicles
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountVehiclesByUserID :one
SELECT COUNT(*)::bigint
FROM vehicles
WHERE user_id = $1;

-- name: GetVehicleByID :one
SELECT id, user_id, plate_number, created_at
FROM vehicles
WHERE id = $1;

-- name: GetVehicleByPlateNumber :one
SELECT id, user_id, plate_number, created_at
FROM vehicles
WHERE plate_number = $1;

-- name: DeleteVehicle :execrows
DELETE FROM vehicles
WHERE id = $1 AND user_id = $2;
