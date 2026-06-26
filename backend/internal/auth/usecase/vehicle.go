package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/auth/port"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

type VehicleService struct {
	vehicles port.VehicleRepository
}

func NewVehicleService(vehicles port.VehicleRepository) *VehicleService {
	return &VehicleService{vehicles: vehicles}
}

type CreateVehicleInput struct {
	UserID      uuid.UUID
	PlateNumber string
}

func (s *VehicleService) Create(ctx context.Context, input CreateVehicleInput) (VehicleOutput, error) {
	plate := domain.NormalizePlate(input.PlateNumber)
	if !domain.IsValidPlate(plate) {
		return VehicleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "plate_number", Message: "must be between 2 and 20 characters",
		})
	}

	vehicle, err := s.vehicles.CreateVehicle(ctx, input.UserID, plate)
	if err != nil {
		if isUniqueViolation(err) {
			return VehicleOutput{}, apperrors.New(apperrors.CodeConflict, "plate number already registered", 409)
		}
		return VehicleOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to create vehicle", 500, err)
	}

	return toVehicleOutput(vehicle), nil
}

type ListVehiclesInput struct {
	UserID   uuid.UUID
	Page     int
	PageSize int
}

func (s *VehicleService) List(ctx context.Context, input ListVehiclesInput) (VehicleListOutput, error) {
	page, pageSize := normalizePagination(input.Page, input.PageSize)
	offset := int32((page - 1) * pageSize)

	vehicles, err := s.vehicles.ListVehiclesByUserID(ctx, input.UserID, int32(pageSize), offset)
	if err != nil {
		return VehicleListOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to list vehicles", 500, err)
	}

	total, err := s.vehicles.CountVehiclesByUserID(ctx, input.UserID)
	if err != nil {
		return VehicleListOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to count vehicles", 500, err)
	}

	data := make([]VehicleOutput, 0, len(vehicles))
	for _, vehicle := range vehicles {
		data = append(data, toVehicleOutput(vehicle))
	}

	return VehicleListOutput{
		Data: data,
		Meta: PaginationMeta{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *VehicleService) Delete(ctx context.Context, userID, vehicleID uuid.UUID) error {
	rows, err := s.vehicles.DeleteVehicle(ctx, vehicleID, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, "failed to delete vehicle", 500, err)
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "vehicle not found", 404)
	}
	return nil
}

func (s *VehicleService) GetByPlate(ctx context.Context, plateNumber string) (domain.Vehicle, error) {
	plate := domain.NormalizePlate(plateNumber)
	vehicle, err := s.vehicles.GetVehicleByPlateNumber(ctx, plate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Vehicle{}, apperrors.New(apperrors.CodeNotFound, "vehicle not found", 404)
		}
		return domain.Vehicle{}, apperrors.Wrap(apperrors.CodeInternal, "failed to get vehicle", 500, err)
	}
	return vehicle, nil
}

func toVehicleOutput(vehicle domain.Vehicle) VehicleOutput {
	return VehicleOutput{
		ID:          vehicle.ID,
		PlateNumber: vehicle.PlateNumber,
		CreatedAt:   vehicle.CreatedAt,
	}
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}
