package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type FineRule struct {
	ID          uuid.UUID
	Code        string
	Name        string
	Description string
	FineAmount  float64
	Version     int32
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeCode(code string) string {
	return strings.TrimSpace(strings.ToUpper(code))
}

func IsValidCode(code string) bool {
	code = NormalizeCode(code)
	return len(code) >= 3 && len(code) <= 50
}

func IsValidName(name string) bool {
	name = strings.TrimSpace(name)
	return len(name) >= 3 && len(name) <= 255
}

func IsValidAmount(amount float64) bool {
	return amount >= 0
}
