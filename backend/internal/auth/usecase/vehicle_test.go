package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/auth/usecase"
)

type mockVehicleRepo struct {
	createFn     func(ctx context.Context, userID uuid.UUID, plateNumber string) (domain.Vehicle, error)
	listFn       func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Vehicle, error)
	countFn      func(ctx context.Context, userID uuid.UUID) (int64, error)
	deleteFn     func(ctx context.Context, id, userID uuid.UUID) (int64, error)
	getByPlateFn func(ctx context.Context, plateNumber string) (domain.Vehicle, error)
}

func (m *mockVehicleRepo) CreateVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) (domain.Vehicle, error) {
	return m.createFn(ctx, userID, plateNumber)
}

func (m *mockVehicleRepo) ListVehiclesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Vehicle, error) {
	return m.listFn(ctx, userID, limit, offset)
}

func (m *mockVehicleRepo) CountVehiclesByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return m.countFn(ctx, userID)
}

func (m *mockVehicleRepo) GetVehicleByID(ctx context.Context, id uuid.UUID) (domain.Vehicle, error) {
	return domain.Vehicle{}, nil
}

func (m *mockVehicleRepo) GetVehicleByPlateNumber(ctx context.Context, plateNumber string) (domain.Vehicle, error) {
	if m.getByPlateFn != nil {
		return m.getByPlateFn(ctx, plateNumber)
	}
	return domain.Vehicle{}, pgx.ErrNoRows
}

func (m *mockVehicleRepo) DeleteVehicle(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	return m.deleteFn(ctx, id, userID)
}

func TestVehicleService_Create_Success(t *testing.T) {
	userID := uuid.New()
	vehicleID := uuid.New()
	now := time.Now().UTC()

	service := usecase.NewVehicleService(&mockVehicleRepo{
		createFn: func(ctx context.Context, uid uuid.UUID, plateNumber string) (domain.Vehicle, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, "ABC1234", plateNumber)
			return domain.Vehicle{
				ID:          vehicleID,
				UserID:      uid,
				PlateNumber: plateNumber,
				CreatedAt:   now,
			}, nil
		},
	})

	out, err := service.Create(context.Background(), usecase.CreateVehicleInput{
		UserID:      userID,
		PlateNumber: " abc1234 ",
	})
	require.NoError(t, err)
	assert.Equal(t, vehicleID, out.ID)
	assert.Equal(t, "ABC1234", out.PlateNumber)
}

func TestVehicleService_Create_ValidationError(t *testing.T) {
	service := usecase.NewVehicleService(&mockVehicleRepo{})

	_, err := service.Create(context.Background(), usecase.CreateVehicleInput{
		UserID: uuid.New(), PlateNumber: "A",
	})
	requireAppError(t, err, apperrors.CodeValidation)
}

func TestVehicleService_Create_Conflict(t *testing.T) {
	service := usecase.NewVehicleService(&mockVehicleRepo{
		createFn: func(ctx context.Context, userID uuid.UUID, plateNumber string) (domain.Vehicle, error) {
			return domain.Vehicle{}, &pgconn.PgError{Code: "23505"}
		},
	})

	_, err := service.Create(context.Background(), usecase.CreateVehicleInput{
		UserID: uuid.New(), PlateNumber: "ABC1234",
	})
	requireAppError(t, err, apperrors.CodeConflict)
}

func TestVehicleService_List_PaginationDefaults(t *testing.T) {
	userID := uuid.New()
	service := usecase.NewVehicleService(&mockVehicleRepo{
		listFn: func(ctx context.Context, uid uuid.UUID, limit, offset int32) ([]domain.Vehicle, error) {
			assert.Equal(t, int32(20), limit)
			assert.Equal(t, int32(0), offset)
			return []domain.Vehicle{}, nil
		},
		countFn: func(ctx context.Context, uid uuid.UUID) (int64, error) {
			return 0, nil
		},
	})

	out, err := service.List(context.Background(), usecase.ListVehiclesInput{UserID: userID})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Meta.Page)
	assert.Equal(t, 20, out.Meta.PageSize)
}

func TestVehicleService_Delete_NotFound(t *testing.T) {
	service := usecase.NewVehicleService(&mockVehicleRepo{
		deleteFn: func(ctx context.Context, id, userID uuid.UUID) (int64, error) {
			return 0, nil
		},
	})

	err := service.Delete(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, apperrors.CodeNotFound)
}

func TestDomain_Normalization(t *testing.T) {
	assert.Equal(t, "member@example.com", domain.NormalizeEmail(" Member@Example.com "))
	assert.Equal(t, "ABC1234", domain.NormalizePlate(" abc1234 "))
	assert.True(t, domain.IsValidRole(domain.RoleOfficer))
	assert.False(t, domain.IsValidRole("admin"))
}
