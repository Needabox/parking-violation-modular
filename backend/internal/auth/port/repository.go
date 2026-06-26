package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, email, passwordHash, role string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
}

type VehicleRepository interface {
	CreateVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) (domain.Vehicle, error)
	ListVehiclesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Vehicle, error)
	CountVehiclesByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetVehicleByID(ctx context.Context, id uuid.UUID) (domain.Vehicle, error)
	GetVehicleByPlateNumber(ctx context.Context, plateNumber string) (domain.Vehicle, error)
	DeleteVehicle(ctx context.Context, id, userID uuid.UUID) (int64, error)
}

type TokenGenerator interface {
	Generate(userID uuid.UUID, email, role string) (token string, expiresIn int64, err error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}
