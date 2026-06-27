package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/parking-violation-portal/backend/internal/payment/domain"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, p domain.Payment) (domain.Payment, error)
	GetPaymentByID(ctx context.Context, id uuid.UUID) (domain.Payment, error)
	ListPaymentsByViolationID(ctx context.Context, violationID uuid.UUID) ([]domain.Payment, error)
}

type ViolationDetail struct {
	ID          uuid.UUID
	PlateNumber string
	FineAmount  float64
	Status      string
	FineRuleID  uuid.UUID
}

type ViolationService interface {
	GetViolationForPayment(ctx context.Context, id uuid.UUID) (ViolationDetail, error)
	UpdateViolationStatusToPaid(ctx context.Context, id uuid.UUID) error
}

type TransactionRecorder interface {
	RecordTransaction(ctx context.Context, violationID uuid.UUID, amount float64, ruleID uuid.UUID) error
}
