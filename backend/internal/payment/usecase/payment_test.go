package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authdomain "github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/payment/domain"
	"github.com/parking-violation-portal/backend/internal/payment/port"
	"github.com/parking-violation-portal/backend/internal/payment/usecase"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type mockPaymentRepo struct {
	createFn func(ctx context.Context, p domain.Payment) (domain.Payment, error)
}

func (m *mockPaymentRepo) CreatePayment(ctx context.Context, p domain.Payment) (domain.Payment, error) {
	return m.createFn(ctx, p)
}
func (m *mockPaymentRepo) GetPaymentByID(ctx context.Context, id uuid.UUID) (domain.Payment, error) {
	return domain.Payment{}, nil
}
func (m *mockPaymentRepo) ListPaymentsByViolationID(ctx context.Context, violationID uuid.UUID) ([]domain.Payment, error) {
	return nil, nil
}

type mockViolationService struct {
	getFn    func(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error)
	updateFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockViolationService) GetViolationForPayment(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error) {
	return m.getFn(ctx, id)
}
func (m *mockViolationService) UpdateViolationStatusToPaid(ctx context.Context, id uuid.UUID) error {
	return m.updateFn(ctx, id)
}

type mockVehicleRepo struct {
	getByPlateFn func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error)
}

func (m *mockVehicleRepo) CreateVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) (authdomain.Vehicle, error) {
	return authdomain.Vehicle{}, nil
}
func (m *mockVehicleRepo) ListVehiclesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]authdomain.Vehicle, error) {
	return nil, nil
}
func (m *mockVehicleRepo) CountVehiclesByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockVehicleRepo) GetVehicleByID(ctx context.Context, id uuid.UUID) (authdomain.Vehicle, error) {
	return authdomain.Vehicle{}, nil
}
func (m *mockVehicleRepo) GetVehicleByPlateNumber(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
	if m.getByPlateFn != nil {
		return m.getByPlateFn(ctx, plateNumber)
	}
	return authdomain.Vehicle{}, pgx.ErrNoRows
}
func (m *mockVehicleRepo) DeleteVehicle(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	return 0, nil
}

type mockTransactionRecorder struct {
	recordFn func(ctx context.Context, violationID uuid.UUID, amount float64, ruleID uuid.UUID) error
}

func (m *mockTransactionRecorder) RecordTransaction(ctx context.Context, violationID uuid.UUID, amount float64, ruleID uuid.UUID) error {
	if m.recordFn != nil {
		return m.recordFn(ctx, violationID, amount, ruleID)
	}
	return nil
}

func requireAppError(t *testing.T, err error, code apperrors.Code) {
	require.Error(t, err)
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok, "expected AppError, got %v", err)
	assert.Equal(t, code, appErr.Code)
}

func TestPaymentService_Pay_Success(t *testing.T) {
	violationID := uuid.New()
	ruleID := uuid.New()
	paymentID := uuid.New()
	memberID := uuid.New()
	now := time.Now().UTC()

	violationUpdated := false
	paymentCreated := false
	transactionRecorded := false

	service := usecase.NewPaymentService(
		&mockPaymentRepo{
			createFn: func(ctx context.Context, p domain.Payment) (domain.Payment, error) {
				assert.Equal(t, violationID, p.ViolationID)
				assert.Equal(t, float64(100.00), p.Amount)
				assert.Equal(t, domain.StatusSuccess, p.Status)
				assert.NotEmpty(t, p.ReferenceID)
				assert.Empty(t, p.ErrorMessage)
				paymentCreated = true
				p.ID = paymentID
				p.CreatedAt = now
				return p, nil
			},
		},
		&mockViolationService{
			getFn: func(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error) {
				assert.Equal(t, violationID, id)
				return port.ViolationDetail{
					ID:          violationID,
					PlateNumber: "B1234XYZ",
					FineAmount:  100.00,
					Status:      "PENDING",
					FineRuleID:  ruleID,
				}, nil
			},
			updateFn: func(ctx context.Context, id uuid.UUID) error {
				assert.Equal(t, violationID, id)
				violationUpdated = true
				return nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				assert.Equal(t, "B1234XYZ", plateNumber)
				return authdomain.Vehicle{
					UserID:      memberID,
					PlateNumber: "B1234XYZ",
				}, nil
			},
		},
		&mockTransactionRecorder{
			recordFn: func(ctx context.Context, vID uuid.UUID, amt float64, rID uuid.UUID) error {
				assert.Equal(t, violationID, vID)
				assert.Equal(t, float64(100.00), amt)
				assert.Equal(t, ruleID, rID)
				transactionRecorded = true
				return nil
			},
		},
	)

	out, err := service.Pay(context.Background(), memberID, authdomain.RoleMember, usecase.PayInput{
		ViolationID: violationID,
		CardNumber:  "1234 5678 1234 5678", // Cleaned to "1234567812345678" -> ends in "5678" (success)
		CVV:         "123",
		ExpiryMonth: 12,
		ExpiryYear:  2030,
	})

	require.NoError(t, err)
	assert.True(t, violationUpdated)
	assert.True(t, paymentCreated)
	assert.True(t, transactionRecorded)
	assert.Equal(t, paymentID, out.ID)
	assert.Equal(t, domain.StatusSuccess, out.Status)
}

func TestPaymentService_Pay_DeclinedCard(t *testing.T) {
	violationID := uuid.New()
	ruleID := uuid.New()
	paymentID := uuid.New()
	memberID := uuid.New()
	now := time.Now().UTC()

	violationUpdated := false
	paymentCreated := false

	service := usecase.NewPaymentService(
		&mockPaymentRepo{
			createFn: func(ctx context.Context, p domain.Payment) (domain.Payment, error) {
				assert.Equal(t, violationID, p.ViolationID)
				assert.Equal(t, domain.StatusFailed, p.Status)
				assert.NotEmpty(t, p.ErrorMessage)
				assert.Empty(t, p.ReferenceID)
				paymentCreated = true
				p.ID = paymentID
				p.CreatedAt = now
				return p, nil
			},
		},
		&mockViolationService{
			getFn: func(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error) {
				return port.ViolationDetail{
					ID:          violationID,
					PlateNumber: "B1234XYZ",
					FineAmount:  100.00,
					Status:      "PENDING",
					FineRuleID:  ruleID,
				}, nil
			},
			updateFn: func(ctx context.Context, id uuid.UUID) error {
				violationUpdated = true
				return nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				return authdomain.Vehicle{
					UserID:      memberID,
					PlateNumber: "B1234XYZ",
				}, nil
			},
		},
		&mockTransactionRecorder{},
	)

	_, err := service.Pay(context.Background(), memberID, authdomain.RoleMember, usecase.PayInput{
		ViolationID: violationID,
		CardNumber:  "4111 1111 1111 0000", // ends in "0000" (decline trigger)
		CVV:         "123",
		ExpiryMonth: 12,
		ExpiryYear:  2030,
	})

	requireAppError(t, err, apperrors.CodePaymentFailed)
	assert.False(t, violationUpdated)
	assert.True(t, paymentCreated)
}

func TestPaymentService_Pay_DeclinedCVV(t *testing.T) {
	violationID := uuid.New()
	memberID := uuid.New()
	paymentCreated := false

	service := usecase.NewPaymentService(
		&mockPaymentRepo{
			createFn: func(ctx context.Context, p domain.Payment) (domain.Payment, error) {
				assert.Equal(t, domain.StatusFailed, p.Status)
				paymentCreated = true
				return p, nil
			},
		},
		&mockViolationService{
			getFn: func(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error) {
				return port.ViolationDetail{
					ID:          violationID,
					PlateNumber: "B1234XYZ",
					FineAmount:  100.00,
					Status:      "PENDING",
				}, nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				return authdomain.Vehicle{
					UserID:      memberID,
					PlateNumber: "B1234XYZ",
				}, nil
			},
		},
		&mockTransactionRecorder{},
	)

	_, err := service.Pay(context.Background(), memberID, authdomain.RoleMember, usecase.PayInput{
		ViolationID: violationID,
		CardNumber:  "4111 1111 1111 1111",
		CVV:         "000", // decline trigger
		ExpiryMonth: 12,
		ExpiryYear:  2030,
	})

	requireAppError(t, err, apperrors.CodePaymentFailed)
	assert.True(t, paymentCreated)
}

func TestPaymentService_Pay_AlreadyPaid(t *testing.T) {
	violationID := uuid.New()
	memberID := uuid.New()

	service := usecase.NewPaymentService(
		&mockPaymentRepo{},
		&mockViolationService{
			getFn: func(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error) {
				return port.ViolationDetail{
					ID:          violationID,
					PlateNumber: "B1234XYZ",
					Status:      "PAID", // already paid
				}, nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				return authdomain.Vehicle{
					UserID:      memberID,
					PlateNumber: "B1234XYZ",
				}, nil
			},
		},
		&mockTransactionRecorder{},
	)

	_, err := service.Pay(context.Background(), memberID, authdomain.RoleMember, usecase.PayInput{
		ViolationID: violationID,
		CardNumber:  "4111 1111 1111 1111",
		CVV:         "123",
		ExpiryMonth: 12,
		ExpiryYear:  2030,
	})

	requireAppError(t, err, apperrors.CodeConflict)
}

func TestPaymentService_Pay_MemberForbidden(t *testing.T) {
	violationID := uuid.New()
	memberID := uuid.New()

	service := usecase.NewPaymentService(
		&mockPaymentRepo{},
		&mockViolationService{
			getFn: func(ctx context.Context, id uuid.UUID) (port.ViolationDetail, error) {
				return port.ViolationDetail{
					ID:          violationID,
					PlateNumber: "B1234XYZ",
					Status:      "PENDING",
				}, nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				return authdomain.Vehicle{
					UserID:      uuid.New(), // owned by another user
					PlateNumber: "B1234XYZ",
				}, nil
			},
		},
		&mockTransactionRecorder{},
	)

	_, err := service.Pay(context.Background(), memberID, authdomain.RoleMember, usecase.PayInput{
		ViolationID: violationID,
		CardNumber:  "4111 1111 1111 1111",
		CVV:         "123",
		ExpiryMonth: 12,
		ExpiryYear:  2030,
	})

	requireAppError(t, err, apperrors.CodeForbidden)
}

func TestPaymentService_Pay_ValidationError(t *testing.T) {
	service := usecase.NewPaymentService(&mockPaymentRepo{}, &mockViolationService{}, &mockVehicleRepo{}, &mockTransactionRecorder{})

	// Card too short
	_, err := service.Pay(context.Background(), uuid.New(), authdomain.RoleMember, usecase.PayInput{
		ViolationID: uuid.New(), CardNumber: "12345", CVV: "123", ExpiryMonth: 12, ExpiryYear: 2030,
	})
	requireAppError(t, err, apperrors.CodeValidation)

	// CVV too short
	_, err = service.Pay(context.Background(), uuid.New(), authdomain.RoleMember, usecase.PayInput{
		ViolationID: uuid.New(), CardNumber: "4111111111111111", CVV: "1", ExpiryMonth: 12, ExpiryYear: 2030,
	})
	requireAppError(t, err, apperrors.CodeValidation)

	// Expiry month invalid
	_, err = service.Pay(context.Background(), uuid.New(), authdomain.RoleMember, usecase.PayInput{
		ViolationID: uuid.New(), CardNumber: "4111111111111111", CVV: "123", ExpiryMonth: 15, ExpiryYear: 2030,
	})
	requireAppError(t, err, apperrors.CodeValidation)

	// Expiry year in past
	_, err = service.Pay(context.Background(), uuid.New(), authdomain.RoleMember, usecase.PayInput{
		ViolationID: uuid.New(), CardNumber: "4111111111111111", CVV: "123", ExpiryMonth: 12, ExpiryYear: 2020,
	})
	requireAppError(t, err, apperrors.CodeValidation)
}
