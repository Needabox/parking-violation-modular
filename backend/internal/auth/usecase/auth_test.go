package usecase_test

import (
	"context"
	"errors"
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

type mockUserRepo struct {
	createFn      func(ctx context.Context, email, passwordHash, role string) (domain.User, error)
	getByEmailFn  func(ctx context.Context, email string) (domain.User, error)
	getByIDFn     func(ctx context.Context, id uuid.UUID) (domain.User, error)
}

func (m *mockUserRepo) CreateUser(ctx context.Context, email, passwordHash, role string) (domain.User, error) {
	return m.createFn(ctx, email, passwordHash, role)
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return m.getByIDFn(ctx, id)
}

type mockHasher struct {
	hashFn    func(password string) (string, error)
	compareFn func(hash, password string) error
}

func (m *mockHasher) Hash(password string) (string, error) {
	return m.hashFn(password)
}

func (m *mockHasher) Compare(hash, password string) error {
	return m.compareFn(hash, password)
}

type mockTokenGen struct {
	generateFn func(userID uuid.UUID, email, role string) (string, int64, error)
}

func (m *mockTokenGen) Generate(userID uuid.UUID, email, role string) (string, int64, error) {
	return m.generateFn(userID, email, role)
}

func TestAuthService_Register_Success(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	service := usecase.NewAuthService(
		&mockUserRepo{
			createFn: func(ctx context.Context, email, passwordHash, role string) (domain.User, error) {
				assert.Equal(t, "member@example.com", email)
				assert.Equal(t, "member", role)
				assert.NotEmpty(t, passwordHash)
				return domain.User{
					ID:        userID,
					Email:     email,
					Role:      role,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
		},
		&mockHasher{
			hashFn: func(password string) (string, error) {
				return "hashed-" + password, nil
			},
		},
		&mockTokenGen{},
	)

	out, err := service.Register(context.Background(), usecase.RegisterInput{
		Email:    "Member@Example.com",
		Password: "password123",
		Role:     "member",
	})
	require.NoError(t, err)
	assert.Equal(t, userID, out.ID)
	assert.Equal(t, "member@example.com", out.Email)
	assert.Equal(t, "member", out.Role)
}

func TestAuthService_Register_ValidationErrors(t *testing.T) {
	service := usecase.NewAuthService(&mockUserRepo{}, &mockHasher{}, &mockTokenGen{})

	_, err := service.Register(context.Background(), usecase.RegisterInput{
		Email: "bad", Password: "password123", Role: "member",
	})
	requireAppError(t, err, apperrors.CodeValidation)

	_, err = service.Register(context.Background(), usecase.RegisterInput{
		Email: "member@example.com", Password: "short", Role: "member",
	})
	requireAppError(t, err, apperrors.CodeValidation)

	_, err = service.Register(context.Background(), usecase.RegisterInput{
		Email: "member@example.com", Password: "password123", Role: "admin",
	})
	requireAppError(t, err, apperrors.CodeValidation)
}

func TestAuthService_Register_Conflict(t *testing.T) {
	service := usecase.NewAuthService(
		&mockUserRepo{
			createFn: func(ctx context.Context, email, passwordHash, role string) (domain.User, error) {
				return domain.User{}, &pgconn.PgError{Code: "23505"}
			},
		},
		&mockHasher{hashFn: func(password string) (string, error) { return "hash", nil }},
		&mockTokenGen{},
	)

	_, err := service.Register(context.Background(), usecase.RegisterInput{
		Email: "member@example.com", Password: "password123", Role: "member",
	})
	requireAppError(t, err, apperrors.CodeConflict)
}

func TestAuthService_Login_Success(t *testing.T) {
	userID := uuid.New()
	service := usecase.NewAuthService(
		&mockUserRepo{
			getByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
				return domain.User{
					ID:           userID,
					Email:        email,
					PasswordHash: "hash",
					Role:         domain.RoleMember,
				}, nil
			},
		},
		&mockHasher{
			compareFn: func(hash, password string) error {
				assert.Equal(t, "hash", hash)
				assert.Equal(t, "password123", password)
				return nil
			},
		},
		&mockTokenGen{
			generateFn: func(id uuid.UUID, email, role string) (string, int64, error) {
				return "token", 3600, nil
			},
		},
	)

	out, err := service.Login(context.Background(), usecase.LoginInput{
		Email:    "member@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "token", out.AccessToken)
	assert.Equal(t, int64(3600), out.ExpiresIn)
	assert.Equal(t, userID, out.User.ID)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	service := usecase.NewAuthService(
		&mockUserRepo{
			getByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
				return domain.User{}, pgx.ErrNoRows
			},
		},
		&mockHasher{},
		&mockTokenGen{},
	)

	_, err := service.Login(context.Background(), usecase.LoginInput{
		Email: "missing@example.com", Password: "password123",
	})
	requireAppError(t, err, apperrors.CodeUnauthorized)

	service = usecase.NewAuthService(
		&mockUserRepo{
			getByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
				return domain.User{PasswordHash: "hash"}, nil
			},
		},
		&mockHasher{compareFn: func(hash, password string) error { return errors.New("mismatch") }},
		&mockTokenGen{},
	)

	_, err = service.Login(context.Background(), usecase.LoginInput{
		Email: "member@example.com", Password: "wrong",
	})
	requireAppError(t, err, apperrors.CodeUnauthorized)
}

func TestAuthService_Me_NotFound(t *testing.T) {
	service := usecase.NewAuthService(
		&mockUserRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.User, error) {
				return domain.User{}, pgx.ErrNoRows
			},
		},
		&mockHasher{},
		&mockTokenGen{},
	)

	_, err := service.Me(context.Background(), uuid.New())
	requireAppError(t, err, apperrors.CodeNotFound)
}

func requireAppError(t *testing.T, err error, code apperrors.Code) {
	t.Helper()
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, code, appErr.Code)
}
