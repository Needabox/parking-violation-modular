package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	RoleOfficer = "officer"
	RoleMember  = "member"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Vehicle struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	PlateNumber string
	CreatedAt   time.Time
}

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func NormalizePlate(plate string) string {
	return strings.TrimSpace(strings.ToUpper(plate))
}

func IsValidRole(role string) bool {
	return role == RoleOfficer || role == RoleMember
}

func IsValidPassword(password string) bool {
	return len(password) >= 8
}

func IsValidEmail(email string) bool {
	email = NormalizeEmail(email)
	return strings.Contains(email, "@") && len(email) >= 5
}

func IsValidPlate(plate string) bool {
	plate = NormalizePlate(plate)
	return len(plate) >= 2 && len(plate) <= 20
}
