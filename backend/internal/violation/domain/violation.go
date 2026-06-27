package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending = "PENDING"
	StatusPaid    = "PAID"
)

type Violation struct {
	ID          uuid.UUID
	FineRuleID  uuid.UUID
	PlateNumber string
	OfficerID   uuid.UUID
	Status      string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ViolationDetail struct {
	Violation
	RuleCode        string
	RuleName        string
	RuleDescription string
	FineAmount      float64
	RuleVersion     int32
}

func NormalizePlate(plate string) string {
	return strings.TrimSpace(strings.ToUpper(plate))
}

func IsValidPlate(plate string) bool {
	plate = NormalizePlate(plate)
	return len(plate) >= 2 && len(plate) <= 20
}

func IsValidStatus(status string) bool {
	return status == StatusPending || status == StatusPaid
}
