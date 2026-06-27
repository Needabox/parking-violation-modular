package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/parking-violation-portal/backend/internal/rules/domain"
	"github.com/parking-violation-portal/backend/internal/rules/port"
	"github.com/parking-violation-portal/backend/internal/rules/usecase"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type mockRuleRepo struct {
	createFn              func(ctx context.Context, rule domain.FineRule) (domain.FineRule, error)
	deactivateFn          func(ctx context.Context, code string) error
	getActiveFn           func(ctx context.Context, code string) (domain.FineRule, error)
	getByIDFn             func(ctx context.Context, id uuid.UUID) (domain.FineRule, error)
	getByCodeAndVersionFn func(ctx context.Context, code string, version int32) (domain.FineRule, error)
	listActiveFn          func(ctx context.Context) ([]domain.FineRule, error)
	listVersionsFn        func(ctx context.Context, code string) ([]domain.FineRule, error)
	withTxFn              func(ctx context.Context, fn func(repo port.RuleRepository) error) error
}

func (m *mockRuleRepo) CreateRule(ctx context.Context, rule domain.FineRule) (domain.FineRule, error) {
	return m.createFn(ctx, rule)
}

func (m *mockRuleRepo) DeactivateActiveRule(ctx context.Context, code string) error {
	return m.deactivateFn(ctx, code)
}

func (m *mockRuleRepo) GetActiveRuleByCode(ctx context.Context, code string) (domain.FineRule, error) {
	if m.getActiveFn != nil {
		return m.getActiveFn(ctx, code)
	}
	return domain.FineRule{}, pgx.ErrNoRows
}

func (m *mockRuleRepo) GetRuleByID(ctx context.Context, id uuid.UUID) (domain.FineRule, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockRuleRepo) GetRuleByCodeAndVersion(ctx context.Context, code string, version int32) (domain.FineRule, error) {
	return m.getByCodeAndVersionFn(ctx, code, version)
}

func (m *mockRuleRepo) ListActiveRules(ctx context.Context) ([]domain.FineRule, error) {
	return m.listActiveFn(ctx)
}

func (m *mockRuleRepo) ListRuleVersions(ctx context.Context, code string) ([]domain.FineRule, error) {
	return m.listVersionsFn(ctx, code)
}

func (m *mockRuleRepo) WithTx(ctx context.Context, fn func(repo port.RuleRepository) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(m)
}

func requireAppError(t *testing.T, err error, code apperrors.Code) {
	require.Error(t, err)
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok, "expected AppError, got %v", err)
	assert.Equal(t, code, appErr.Code)
}

func TestRuleService_Create_Success(t *testing.T) {
	ruleID := uuid.New()
	now := time.Now().UTC()

	service := usecase.NewRuleService(&mockRuleRepo{
		getActiveFn: func(ctx context.Context, code string) (domain.FineRule, error) {
			return domain.FineRule{}, pgx.ErrNoRows
		},
		createFn: func(ctx context.Context, rule domain.FineRule) (domain.FineRule, error) {
			assert.Equal(t, "OVERTIME_PARKING", rule.Code)
			assert.Equal(t, "Overtime Parking", rule.Name)
			assert.Equal(t, float64(50), rule.FineAmount)
			assert.Equal(t, int32(1), rule.Version)
			assert.True(t, rule.IsActive)
			rule.ID = ruleID
			rule.CreatedAt = now
			rule.UpdatedAt = now
			return rule, nil
		},
	})

	out, err := service.Create(context.Background(), usecase.CreateRuleInput{
		Code:        " overtime_parking ",
		Name:        "Overtime Parking",
		Description: "Parking longer than allowed limit",
		FineAmount:  50.00,
	})
	require.NoError(t, err)
	assert.Equal(t, ruleID, out.ID)
	assert.Equal(t, "OVERTIME_PARKING", out.Code)
	assert.Equal(t, int32(1), out.Version)
}

func TestRuleService_Create_ValidationError(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{})

	// Invalid code
	_, err := service.Create(context.Background(), usecase.CreateRuleInput{
		Code: "OP", Name: "Overtime", FineAmount: 10,
	})
	requireAppError(t, err, apperrors.CodeValidation)

	// Invalid name
	_, err = service.Create(context.Background(), usecase.CreateRuleInput{
		Code: "OVERTIME", Name: "O", FineAmount: 10,
	})
	requireAppError(t, err, apperrors.CodeValidation)

	// Invalid fine amount
	_, err = service.Create(context.Background(), usecase.CreateRuleInput{
		Code: "OVERTIME", Name: "Overtime Parking", FineAmount: -5.50,
	})
	requireAppError(t, err, apperrors.CodeValidation)
}

func TestRuleService_Create_Conflict(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{
		getActiveFn: func(ctx context.Context, code string) (domain.FineRule, error) {
			return domain.FineRule{Code: "OVERTIME_PARKING"}, nil
		},
	})

	_, err := service.Create(context.Background(), usecase.CreateRuleInput{
		Code: "OVERTIME_PARKING", Name: "Overtime Parking", FineAmount: 50,
	})
	requireAppError(t, err, apperrors.CodeConflict)
}

func TestRuleService_Update_Success(t *testing.T) {
	ruleID1 := uuid.New()
	ruleID2 := uuid.New()
	now := time.Now().UTC()

	deactivateCalled := false
	createCalled := false

	service := usecase.NewRuleService(&mockRuleRepo{
		getActiveFn: func(ctx context.Context, code string) (domain.FineRule, error) {
			return domain.FineRule{
				ID:         ruleID1,
				Code:       "OVERTIME_PARKING",
				Name:       "Overtime",
				FineAmount: 50,
				Version:    1,
				IsActive:   true,
			}, nil
		},
		deactivateFn: func(ctx context.Context, code string) error {
			assert.Equal(t, "OVERTIME_PARKING", code)
			deactivateCalled = true
			return nil
		},
		createFn: func(ctx context.Context, rule domain.FineRule) (domain.FineRule, error) {
			assert.Equal(t, "OVERTIME_PARKING", rule.Code)
			assert.Equal(t, "Overtime Parking Updated", rule.Name)
			assert.Equal(t, float64(75), rule.FineAmount)
			assert.Equal(t, int32(2), rule.Version)
			assert.True(t, rule.IsActive)
			createCalled = true
			rule.ID = ruleID2
			rule.CreatedAt = now
			rule.UpdatedAt = now
			return rule, nil
		},
	})

	out, err := service.Update(context.Background(), "overtime_parking", usecase.UpdateRuleInput{
		Name:        "Overtime Parking Updated",
		Description: "Updated fee",
		FineAmount:  75.00,
	})
	require.NoError(t, err)
	assert.True(t, deactivateCalled)
	assert.True(t, createCalled)
	assert.Equal(t, ruleID2, out.ID)
	assert.Equal(t, int32(2), out.Version)
}

func TestRuleService_Update_NotFound(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{
		getActiveFn: func(ctx context.Context, code string) (domain.FineRule, error) {
			return domain.FineRule{}, pgx.ErrNoRows
		},
	})

	_, err := service.Update(context.Background(), "OVERTIME_PARKING", usecase.UpdateRuleInput{
		Name: "Updated Rule", FineAmount: 50,
	})
	requireAppError(t, err, apperrors.CodeNotFound)
}

func TestRuleService_GetActiveByCode_Success(t *testing.T) {
	ruleID := uuid.New()
	service := usecase.NewRuleService(&mockRuleRepo{
		getActiveFn: func(ctx context.Context, code string) (domain.FineRule, error) {
			return domain.FineRule{
				ID:         ruleID,
				Code:       "OVERTIME",
				Name:       "Overtime",
				FineAmount: 50,
				Version:    1,
				IsActive:   true,
			}, nil
		},
	})

	out, err := service.GetActiveByCode(context.Background(), "overtime")
	require.NoError(t, err)
	assert.Equal(t, ruleID, out.ID)
	assert.Equal(t, "OVERTIME", out.Code)
}

func TestRuleService_GetActiveByCode_NotFound(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{
		getActiveFn: func(ctx context.Context, code string) (domain.FineRule, error) {
			return domain.FineRule{}, pgx.ErrNoRows
		},
	})

	_, err := service.GetActiveByCode(context.Background(), "overtime")
	requireAppError(t, err, apperrors.CodeNotFound)
}

func TestRuleService_ListActive_Success(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{
		listActiveFn: func(ctx context.Context) ([]domain.FineRule, error) {
			return []domain.FineRule{
				{Code: "OVERTIME", Version: 1},
				{Code: "NO_PARKING", Version: 3},
			}, nil
		},
	})

	out, err := service.ListActive(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "OVERTIME", out[0].Code)
	assert.Equal(t, "NO_PARKING", out[1].Code)
}

func TestRuleService_ListVersions_Success(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{
		listVersionsFn: func(ctx context.Context, code string) ([]domain.FineRule, error) {
			return []domain.FineRule{
				{Code: "OVERTIME", Version: 2, IsActive: true},
				{Code: "OVERTIME", Version: 1, IsActive: false},
			}, nil
		},
	})

	out, err := service.ListVersions(context.Background(), "overtime")
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, int32(2), out[0].Version)
	assert.Equal(t, int32(1), out[1].Version)
}

func TestRuleService_ListVersions_NotFound(t *testing.T) {
	service := usecase.NewRuleService(&mockRuleRepo{
		listVersionsFn: func(ctx context.Context, code string) ([]domain.FineRule, error) {
			return []domain.FineRule{}, nil
		},
	})

	_, err := service.ListVersions(context.Background(), "overtime")
	requireAppError(t, err, apperrors.CodeNotFound)
}
