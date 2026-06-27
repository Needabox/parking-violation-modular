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
	rulesdomain "github.com/parking-violation-portal/backend/internal/rules/domain"
	rulesport "github.com/parking-violation-portal/backend/internal/rules/port"
	"github.com/parking-violation-portal/backend/internal/violation/domain"
	"github.com/parking-violation-portal/backend/internal/violation/usecase"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type mockViolationRepo struct {
	createFn        func(ctx context.Context, v domain.Violation) (domain.Violation, error)
	getByIDFn       func(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error)
	listAllFn       func(ctx context.Context, limit, offset int32) ([]domain.ViolationDetail, error)
	countAllFn      func(ctx context.Context) (int64, error)
	listByPlatesFn  func(ctx context.Context, plates []string, limit, offset int32) ([]domain.ViolationDetail, error)
	countByPlatesFn func(ctx context.Context, plates []string) (int64, error)
	updateStatusFn  func(ctx context.Context, id uuid.UUID, status string) (domain.Violation, error)
}

func (m *mockViolationRepo) CreateViolation(ctx context.Context, v domain.Violation) (domain.Violation, error) {
	return m.createFn(ctx, v)
}
func (m *mockViolationRepo) GetViolationByID(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockViolationRepo) ListAllViolations(ctx context.Context, limit, offset int32) ([]domain.ViolationDetail, error) {
	return m.listAllFn(ctx, limit, offset)
}
func (m *mockViolationRepo) CountAllViolations(ctx context.Context) (int64, error) {
	return m.countAllFn(ctx)
}
func (m *mockViolationRepo) ListViolationsByPlates(ctx context.Context, plates []string, limit, offset int32) ([]domain.ViolationDetail, error) {
	return m.listByPlatesFn(ctx, plates, limit, offset)
}
func (m *mockViolationRepo) CountViolationsByPlates(ctx context.Context, plates []string) (int64, error) {
	return m.countByPlatesFn(ctx, plates)
}
func (m *mockViolationRepo) UpdateViolationStatus(ctx context.Context, id uuid.UUID, status string) (domain.Violation, error) {
	return m.updateStatusFn(ctx, id, status)
}

type mockVehicleRepo struct {
	createFn      func(ctx context.Context, userID uuid.UUID, plateNumber string) (authdomain.Vehicle, error)
	listByFn      func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]authdomain.Vehicle, error)
	countFn       func(ctx context.Context, userID uuid.UUID) (int64, error)
	getByIDFn     func(ctx context.Context, id uuid.UUID) (authdomain.Vehicle, error)
	getByPlateFn  func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error)
	deleteFn      func(ctx context.Context, id, userID uuid.UUID) (int64, error)
}

func (m *mockVehicleRepo) CreateVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) (authdomain.Vehicle, error) {
	return m.createFn(ctx, userID, plateNumber)
}
func (m *mockVehicleRepo) ListVehiclesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]authdomain.Vehicle, error) {
	return m.listByFn(ctx, userID, limit, offset)
}
func (m *mockVehicleRepo) CountVehiclesByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return m.countFn(ctx, userID)
}
func (m *mockVehicleRepo) GetVehicleByID(ctx context.Context, id uuid.UUID) (authdomain.Vehicle, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockVehicleRepo) GetVehicleByPlateNumber(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
	if m.getByPlateFn != nil {
		return m.getByPlateFn(ctx, plateNumber)
	}
	return authdomain.Vehicle{}, pgx.ErrNoRows
}
func (m *mockVehicleRepo) DeleteVehicle(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	return m.deleteFn(ctx, id, userID)
}

type mockRuleRepo struct {
	createFn              func(ctx context.Context, rule rulesdomain.FineRule) (rulesdomain.FineRule, error)
	deactivateFn          func(ctx context.Context, code string) error
	getActiveFn           func(ctx context.Context, code string) (rulesdomain.FineRule, error)
	getByIDFn             func(ctx context.Context, id uuid.UUID) (rulesdomain.FineRule, error)
	getByCodeAndVersionFn func(ctx context.Context, code string, version int32) (rulesdomain.FineRule, error)
	listActiveFn          func(ctx context.Context) ([]rulesdomain.FineRule, error)
	listVersionsFn        func(ctx context.Context, code string) ([]rulesdomain.FineRule, error)
	withTxFn              func(ctx context.Context, fn func(repo rulesport.RuleRepository) error) error
}

func (m *mockRuleRepo) CreateRule(ctx context.Context, rule rulesdomain.FineRule) (rulesdomain.FineRule, error) {
	return m.createFn(ctx, rule)
}
func (m *mockRuleRepo) DeactivateActiveRule(ctx context.Context, code string) error {
	return m.deactivateFn(ctx, code)
}
func (m *mockRuleRepo) GetActiveRuleByCode(ctx context.Context, code string) (rulesdomain.FineRule, error) {
	if m.getActiveFn != nil {
		return m.getActiveFn(ctx, code)
	}
	return rulesdomain.FineRule{}, pgx.ErrNoRows
}
func (m *mockRuleRepo) GetRuleByID(ctx context.Context, id uuid.UUID) (rulesdomain.FineRule, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockRuleRepo) GetRuleByCodeAndVersion(ctx context.Context, code string, version int32) (rulesdomain.FineRule, error) {
	return m.getByCodeAndVersionFn(ctx, code, version)
}
func (m *mockRuleRepo) ListActiveRules(ctx context.Context) ([]rulesdomain.FineRule, error) {
	return m.listActiveFn(ctx)
}
func (m *mockRuleRepo) ListRuleVersions(ctx context.Context, code string) ([]rulesdomain.FineRule, error) {
	return m.listVersionsFn(ctx, code)
}
func (m *mockRuleRepo) WithTx(ctx context.Context, fn func(repo rulesport.RuleRepository) error) error {
	return fn(m)
}

func requireAppError(t *testing.T, err error, code apperrors.Code) {
	require.Error(t, err)
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok, "expected AppError, got %v", err)
	assert.Equal(t, code, appErr.Code)
}

func TestViolationService_Report_Success(t *testing.T) {
	officerID := uuid.New()
	ruleID := uuid.New()
	violationID := uuid.New()
	now := time.Now().UTC()

	service := usecase.NewViolationService(
		&mockViolationRepo{
			createFn: func(ctx context.Context, v domain.Violation) (domain.Violation, error) {
				assert.Equal(t, ruleID, v.FineRuleID)
				assert.Equal(t, "B1234XYZ", v.PlateNumber)
				assert.Equal(t, officerID, v.OfficerID)
				assert.Equal(t, domain.StatusPending, v.Status)
				v.ID = violationID
				v.CreatedAt = now
				v.UpdatedAt = now
				return v, nil
			},
			getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
				assert.Equal(t, violationID, id)
				return domain.ViolationDetail{
					Violation: domain.Violation{
						ID:          violationID,
						FineRuleID:  ruleID,
						PlateNumber: "B1234XYZ",
						OfficerID:   officerID,
						Status:      domain.StatusPending,
						Description: "Parked on crosswalk",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
					RuleCode:        "NO_PARKING",
					RuleName:        "No Parking Zone",
					RuleDescription: "Parking forbidden",
					FineAmount:      100.00,
					RuleVersion:     1,
				}, nil
			},
		},
		&mockVehicleRepo{},
		&mockRuleRepo{
			getActiveFn: func(ctx context.Context, code string) (rulesdomain.FineRule, error) {
				assert.Equal(t, "NO_PARKING", code)
				return rulesdomain.FineRule{
					ID:         ruleID,
					Code:       "NO_PARKING",
					FineAmount: 100.00,
					Version:    1,
				}, nil
			},
		},
	)

	out, err := service.Report(context.Background(), officerID, usecase.ReportInput{
		RuleCode:    "NO_PARKING",
		PlateNumber: " b1234xyz ",
		Description: "Parked on crosswalk",
	})
	require.NoError(t, err)
	assert.Equal(t, violationID, out.ID)
	assert.Equal(t, "B1234XYZ", out.PlateNumber)
	assert.Equal(t, "NO_PARKING", out.RuleCode)
	assert.Equal(t, float64(100), out.FineAmount)
}

func TestViolationService_Report_RuleNotFound(t *testing.T) {
	service := usecase.NewViolationService(
		&mockViolationRepo{},
		&mockVehicleRepo{},
		&mockRuleRepo{
			getActiveFn: func(ctx context.Context, code string) (rulesdomain.FineRule, error) {
				return rulesdomain.FineRule{}, pgx.ErrNoRows
			},
		},
	)

	_, err := service.Report(context.Background(), uuid.New(), usecase.ReportInput{
		RuleCode:    "NOT_EXIST",
		PlateNumber: "B1234XYZ",
	})
	requireAppError(t, err, apperrors.CodeValidation)
}

func TestViolationService_Report_ValidationError(t *testing.T) {
	service := usecase.NewViolationService(
		&mockViolationRepo{},
		&mockVehicleRepo{},
		&mockRuleRepo{},
	)

	// Plate too short
	_, err := service.Report(context.Background(), uuid.New(), usecase.ReportInput{
		RuleCode:    "NO_PARKING",
		PlateNumber: "A",
	})
	requireAppError(t, err, apperrors.CodeValidation)

	// Rule code missing
	_, err = service.Report(context.Background(), uuid.New(), usecase.ReportInput{
		RuleCode:    "",
		PlateNumber: "ABC1234",
	})
	requireAppError(t, err, apperrors.CodeValidation)
}

func TestViolationService_GetViolation_OfficerSuccess(t *testing.T) {
	violationID := uuid.New()
	service := usecase.NewViolationService(
		&mockViolationRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
				return domain.ViolationDetail{
					Violation: domain.Violation{
						ID:          violationID,
						PlateNumber: "B1234XYZ",
					},
					RuleCode: "NO_PARKING",
				}, nil
			},
		},
		&mockVehicleRepo{},
		&mockRuleRepo{},
	)

	out, err := service.GetViolation(context.Background(), uuid.New(), authdomain.RoleOfficer, violationID)
	require.NoError(t, err)
	assert.Equal(t, violationID, out.ID)
}

func TestViolationService_GetViolation_MemberSuccess(t *testing.T) {
	memberID := uuid.New()
	violationID := uuid.New()
	plate := "B1234XYZ"

	service := usecase.NewViolationService(
		&mockViolationRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
				return domain.ViolationDetail{
					Violation: domain.Violation{
						ID:          violationID,
						PlateNumber: plate,
					},
					RuleCode: "NO_PARKING",
				}, nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				assert.Equal(t, plate, plateNumber)
				return authdomain.Vehicle{
					UserID:      memberID,
					PlateNumber: plate,
				}, nil
			},
		},
		&mockRuleRepo{},
	)

	out, err := service.GetViolation(context.Background(), memberID, authdomain.RoleMember, violationID)
	require.NoError(t, err)
	assert.Equal(t, violationID, out.ID)
}

func TestViolationService_GetViolation_MemberForbidden(t *testing.T) {
	memberID := uuid.New()
	violationID := uuid.New()
	plate := "B1234XYZ"

	// Scenario 1: Vehicle registered to someone else
	service1 := usecase.NewViolationService(
		&mockViolationRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
				return domain.ViolationDetail{
					Violation: domain.Violation{
						ID:          violationID,
						PlateNumber: plate,
					},
				}, nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				return authdomain.Vehicle{
					UserID:      uuid.New(), // different member ID
					PlateNumber: plate,
				}, nil
			},
		},
		&mockRuleRepo{},
	)

	_, err := service1.GetViolation(context.Background(), memberID, authdomain.RoleMember, violationID)
	requireAppError(t, err, apperrors.CodeForbidden)

	// Scenario 2: Vehicle not registered in system at all
	service2 := usecase.NewViolationService(
		&mockViolationRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
				return domain.ViolationDetail{
					Violation: domain.Violation{
						ID:          violationID,
						PlateNumber: plate,
					},
				}, nil
			},
		},
		&mockVehicleRepo{
			getByPlateFn: func(ctx context.Context, plateNumber string) (authdomain.Vehicle, error) {
				return authdomain.Vehicle{}, pgx.ErrNoRows
			},
		},
		&mockRuleRepo{},
	)

	_, err = service2.GetViolation(context.Background(), memberID, authdomain.RoleMember, violationID)
	requireAppError(t, err, apperrors.CodeForbidden)
}

func TestViolationService_List_OfficerSuccess(t *testing.T) {
	service := usecase.NewViolationService(
		&mockViolationRepo{
			listAllFn: func(ctx context.Context, limit, offset int32) ([]domain.ViolationDetail, error) {
				assert.Equal(t, int32(20), limit)
				assert.Equal(t, int32(0), offset)
				return []domain.ViolationDetail{
					{Violation: domain.Violation{PlateNumber: "A"}},
					{Violation: domain.Violation{PlateNumber: "B"}},
				}, nil
			},
			countAllFn: func(ctx context.Context) (int64, error) {
				return 2, nil
			},
		},
		&mockVehicleRepo{},
		&mockRuleRepo{},
	)

	out, err := service.List(context.Background(), uuid.New(), authdomain.RoleOfficer, 1, 20)
	require.NoError(t, err)
	assert.Len(t, out.Data, 2)
	assert.Equal(t, int64(2), out.Meta.TotalCount)
	assert.Equal(t, 1, out.Meta.PageCount)
}

func TestViolationService_List_MemberSuccess(t *testing.T) {
	memberID := uuid.New()
	plates := []string{"MEMBER_PLATE1", "MEMBER_PLATE2"}

	service := usecase.NewViolationService(
		&mockViolationRepo{
			listByPlatesFn: func(ctx context.Context, plt []string, limit, offset int32) ([]domain.ViolationDetail, error) {
				assert.Equal(t, plates, plt)
				return []domain.ViolationDetail{
					{Violation: domain.Violation{PlateNumber: "MEMBER_PLATE1"}},
				}, nil
			},
			countByPlatesFn: func(ctx context.Context, plt []string) (int64, error) {
				assert.Equal(t, plates, plt)
				return 1, nil
			},
		},
		&mockVehicleRepo{
			listByFn: func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]authdomain.Vehicle, error) {
				assert.Equal(t, memberID, userID)
				return []authdomain.Vehicle{
					{PlateNumber: "MEMBER_PLATE1"},
					{PlateNumber: "MEMBER_PLATE2"},
				}, nil
			},
		},
		&mockRuleRepo{},
	)

	out, err := service.List(context.Background(), memberID, authdomain.RoleMember, 1, 10)
	require.NoError(t, err)
	assert.Len(t, out.Data, 1)
	assert.Equal(t, "MEMBER_PLATE1", out.Data[0].PlateNumber)
	assert.Equal(t, int64(1), out.Meta.TotalCount)
}

func TestViolationService_List_MemberNoVehicles(t *testing.T) {
	memberID := uuid.New()
	service := usecase.NewViolationService(
		&mockViolationRepo{},
		&mockVehicleRepo{
			listByFn: func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]authdomain.Vehicle, error) {
				return []authdomain.Vehicle{}, nil // no vehicles
			},
		},
		&mockRuleRepo{},
	)

	out, err := service.List(context.Background(), memberID, authdomain.RoleMember, 1, 20)
	require.NoError(t, err)
	assert.Empty(t, out.Data)
	assert.Equal(t, int64(0), out.Meta.TotalCount)
}
