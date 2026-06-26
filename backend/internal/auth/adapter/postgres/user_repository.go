package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/parking-violation-portal/backend/internal/auth/adapter/postgres/authdb"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
)

type UserRepository struct {
	queries *authdb.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: authdb.New(pool)}
}

func (r *UserRepository) CreateUser(ctx context.Context, email, passwordHash, role string) (domain.User, error) {
	row, err := r.queries.CreateUser(ctx, authdb.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	})
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func mapUser(row authdb.User) domain.User {
	return domain.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
