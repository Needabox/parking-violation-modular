package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/parking-violation-portal/backend/internal/auth/adapter/postgres/authdb"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
)

type VehicleRepository struct {
	queries *authdb.Queries
}

func NewVehicleRepository(pool *pgxpool.Pool) *VehicleRepository {
	return &VehicleRepository{queries: authdb.New(pool)}
}

func (r *VehicleRepository) CreateVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) (domain.Vehicle, error) {
	row, err := r.queries.CreateVehicle(ctx, authdb.CreateVehicleParams{
		UserID:      userID,
		PlateNumber: plateNumber,
	})
	if err != nil {
		return domain.Vehicle{}, err
	}
	return mapVehicle(row), nil
}

func (r *VehicleRepository) ListVehiclesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Vehicle, error) {
	rows, err := r.queries.ListVehiclesByUserID(ctx, authdb.ListVehiclesByUserIDParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	vehicles := make([]domain.Vehicle, 0, len(rows))
	for _, row := range rows {
		vehicles = append(vehicles, mapVehicle(row))
	}
	return vehicles, nil
}

func (r *VehicleRepository) CountVehiclesByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.queries.CountVehiclesByUserID(ctx, userID)
}

func (r *VehicleRepository) GetVehicleByID(ctx context.Context, id uuid.UUID) (domain.Vehicle, error) {
	row, err := r.queries.GetVehicleByID(ctx, id)
	if err != nil {
		return domain.Vehicle{}, err
	}
	return mapVehicle(row), nil
}

func (r *VehicleRepository) GetVehicleByPlateNumber(ctx context.Context, plateNumber string) (domain.Vehicle, error) {
	row, err := r.queries.GetVehicleByPlateNumber(ctx, plateNumber)
	if err != nil {
		return domain.Vehicle{}, err
	}
	return mapVehicle(row), nil
}

func (r *VehicleRepository) DeleteVehicle(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	return r.queries.DeleteVehicle(ctx, authdb.DeleteVehicleParams{
		ID:     id,
		UserID: userID,
	})
}

func mapVehicle(row authdb.Vehicle) domain.Vehicle {
	return domain.Vehicle{
		ID:          row.ID,
		UserID:      row.UserID,
		PlateNumber: row.PlateNumber,
		CreatedAt:   row.CreatedAt,
	}
}
