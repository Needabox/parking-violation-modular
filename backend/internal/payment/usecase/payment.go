package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	authdomain "github.com/parking-violation-portal/backend/internal/auth/domain"
	authport "github.com/parking-violation-portal/backend/internal/auth/port"
	"github.com/parking-violation-portal/backend/internal/payment/domain"
	"github.com/parking-violation-portal/backend/internal/payment/port"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type PaymentService struct {
	repo       port.PaymentRepository
	violations port.ViolationService
	vehicles   authport.VehicleRepository
	txRecorder port.TransactionRecorder
}

func NewPaymentService(
	repo port.PaymentRepository,
	violations port.ViolationService,
	vehicles authport.VehicleRepository,
	txRecorder port.TransactionRecorder,
) *PaymentService {
	return &PaymentService{
		repo:       repo,
		violations: violations,
		vehicles:   vehicles,
		txRecorder: txRecorder,
	}
}

type PayInput struct {
	ViolationID uuid.UUID
	CardNumber  string
	CVV         string
	ExpiryMonth int
	ExpiryYear  int
}

type PaymentOutput struct {
	ID            uuid.UUID `json:"id"`
	ViolationID   uuid.UUID `json:"violation_id"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"payment_method"`
	ReferenceID   string    `json:"reference_id,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func toPaymentOutput(p domain.Payment) PaymentOutput {
	return PaymentOutput{
		ID:            p.ID,
		ViolationID:   p.ViolationID,
		Amount:        p.Amount,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		ReferenceID:   p.ReferenceID,
		ErrorMessage:  p.ErrorMessage,
		CreatedAt:     p.CreatedAt,
	}
}

func (s *PaymentService) Pay(ctx context.Context, requesterID uuid.UUID, requesterRole string, input PayInput) (PaymentOutput, error) {
	// 1. Validate Input
	cleanCard := domain.CleanCardNumber(input.CardNumber)
	if !domain.IsValidCardNumber(cleanCard) {
		return PaymentOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "card_number", Message: "must be a valid card number (12-19 digits)",
		})
	}
	if !domain.IsValidCVV(input.CVV) {
		return PaymentOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "cvv", Message: "must be a valid 3-4 digit CVV",
		})
	}
	if input.ExpiryMonth < 1 || input.ExpiryMonth > 12 {
		return PaymentOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "expiry_month", Message: "must be between 1 and 12",
		})
	}
	if input.ExpiryYear < 2026 { // Current year is 2026 based on metadata
		return PaymentOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "expiry_year", Message: "card has expired",
		})
	}

	// 2. Fetch Violation
	violation, err := s.violations.GetViolationForPayment(ctx, input.ViolationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentOutput{}, apperrors.New(apperrors.CodeNotFound, "violation not found", 404)
		}
		return PaymentOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to get violation for payment", 500, err)
	}

	// 3. Authorization Check (for Members)
	if requesterRole == authdomain.RoleMember {
		vehicle, err := s.vehicles.GetVehicleByPlateNumber(ctx, violation.PlateNumber)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return PaymentOutput{}, apperrors.New(apperrors.CodeForbidden, "access forbidden: vehicle not registered to you", 403)
			}
			return PaymentOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to verify vehicle ownership", 500, err)
		}
		if vehicle.UserID != requesterID {
			return PaymentOutput{}, apperrors.New(apperrors.CodeForbidden, "access forbidden: vehicle not registered to you", 403)
		}
	}

	// 4. Check status
	if violation.Status == "PAID" {
		return PaymentOutput{}, apperrors.New(apperrors.CodeConflict, "violation has already been paid", 409)
	}

	// 5. Mock Gateway Processing
	// DECLINED TRIGGER: ends in "0000" or CVV is "000"
	var payment domain.Payment
	if cleanCard[len(cleanCard)-4:] == "0000" || input.CVV == "000" {
		// Save failed payment record
		payment = domain.Payment{
			ViolationID:   violation.ID,
			Amount:        violation.FineAmount,
			Status:        domain.StatusFailed,
			PaymentMethod: "CARD",
			ErrorMessage:  "card declined by gateway provider",
		}
		saved, err := s.repo.CreatePayment(ctx, payment)
		if err != nil {
			return PaymentOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to record failed payment attempt", 500, err)
		}
		return toPaymentOutput(saved), apperrors.New(apperrors.CodePaymentFailed, "payment failed: card declined by processor", 402)
	}

	// SUCCESS SCENARIO
	refID := fmt.Sprintf("ref_tx_%d_%s", time.Now().Unix(), uuid.NewString()[:8])
	payment = domain.Payment{
		ViolationID:   violation.ID,
		Amount:        violation.FineAmount,
		Status:        domain.StatusSuccess,
		PaymentMethod: "CARD",
		ReferenceID:   refID,
	}

	// Update violation status
	if err := s.violations.UpdateViolationStatusToPaid(ctx, violation.ID); err != nil {
		return PaymentOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to update violation status to PAID", 500, err)
	}

	// Save successful payment record
	saved, err := s.repo.CreatePayment(ctx, payment)
	if err != nil {
		return PaymentOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to record successful payment", 500, err)
	}

	// Trigger transaction recorder (decoupled)
	if err := s.txRecorder.RecordTransaction(ctx, violation.ID, violation.FineAmount, violation.FineRuleID); err != nil {
		// Log error but do not roll back payment since money was processed and status is updated.
		// In production, we'd queue an outbox message or retry, but for tech assignment we continue.
		// Note: We can log to stdout
		fmt.Printf("failed to log transaction: %v\n", err)
	}

	return toPaymentOutput(saved), nil
}
