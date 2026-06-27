package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
)

type Payment struct {
	ID            uuid.UUID
	ViolationID   uuid.UUID
	Amount        float64
	Status        string
	PaymentMethod string
	ReferenceID   string
	ErrorMessage  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func CleanCardNumber(card string) string {
	return strings.ReplaceAll(strings.TrimSpace(card), " ", "")
}

func IsValidCardNumber(card string) bool {
	card = CleanCardNumber(card)
	return len(card) >= 12 && len(card) <= 19
}

func IsValidCVV(cvv string) bool {
	cvv = strings.TrimSpace(cvv)
	return len(cvv) >= 3 && len(cvv) <= 4
}
