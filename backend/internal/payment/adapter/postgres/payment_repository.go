package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/parking-violation-portal/backend/internal/payment/adapter/postgres/paymentdb"
	"github.com/parking-violation-portal/backend/internal/payment/domain"
)

type PaymentRepository struct {
	queries *paymentdb.Queries
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{
		queries: paymentdb.New(pool),
	}
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, p domain.Payment) (domain.Payment, error) {
	row, err := r.queries.CreatePayment(ctx, paymentdb.CreatePaymentParams{
		ViolationID:   p.ViolationID,
		Amount:        p.Amount,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		ReferenceID:   p.ReferenceID,
		ErrorMessage:  p.ErrorMessage,
	})
	if err != nil {
		return domain.Payment{}, err
	}
	return mapPayment(row), nil
}

func (r *PaymentRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (domain.Payment, error) {
	row, err := r.queries.GetPaymentByID(ctx, id)
	if err != nil {
		return domain.Payment{}, err
	}
	return mapPayment(row), nil
}

func (r *PaymentRepository) ListPaymentsByViolationID(ctx context.Context, violationID uuid.UUID) ([]domain.Payment, error) {
	rows, err := r.queries.ListPaymentsByViolationID(ctx, violationID)
	if err != nil {
		return nil, err
	}
	return mapPayments(rows), nil
}

func mapPayment(p paymentdb.Payment) domain.Payment {
	return domain.Payment{
		ID:            p.ID,
		ViolationID:   p.ViolationID,
		Amount:        p.Amount,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		ReferenceID:   p.ReferenceID,
		ErrorMessage:  p.ErrorMessage,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func mapPayments(rows []paymentdb.Payment) []domain.Payment {
	items := make([]domain.Payment, len(rows))
	for i, p := range rows {
		items[i] = mapPayment(p)
	}
	return items
}
