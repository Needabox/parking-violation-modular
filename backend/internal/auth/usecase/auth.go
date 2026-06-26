package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/auth/port"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type BcryptHasher struct{}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{}
}

func (BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

type AuthService struct {
	users    port.UserRepository
	hasher   port.PasswordHasher
	tokens   port.TokenGenerator
}

func NewAuthService(users port.UserRepository, hasher port.PasswordHasher, tokens port.TokenGenerator) *AuthService {
	return &AuthService{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

type RegisterInput struct {
	Email    string
	Password string
	Role     string
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (UserOutput, error) {
	email := domain.NormalizeEmail(input.Email)
	role := strings.TrimSpace(strings.ToLower(input.Role))

	if !domain.IsValidEmail(email) {
		return UserOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "email", Message: "must be a valid email address",
		})
	}
	if !domain.IsValidPassword(input.Password) {
		return UserOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "password", Message: "must be at least 8 characters",
		})
	}
	if !domain.IsValidRole(role) {
		return UserOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "role", Message: "must be officer or member",
		})
	}

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return UserOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to hash password", 500, err)
	}

	user, err := s.users.CreateUser(ctx, email, hash, role)
	if err != nil {
		if isUniqueViolation(err) {
			return UserOutput{}, apperrors.New(apperrors.CodeConflict, "email already registered", 409)
		}
		return UserOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to create user", 500, err)
	}

	return toUserOutput(user), nil
}

type LoginInput struct {
	Email    string
	Password string
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (LoginOutput, error) {
	email := domain.NormalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return LoginOutput{}, apperrors.New(apperrors.CodeUnauthorized, "invalid email or password", 401)
	}

	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginOutput{}, apperrors.New(apperrors.CodeUnauthorized, "invalid email or password", 401)
		}
		return LoginOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to lookup user", 500, err)
	}

	if err := s.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		return LoginOutput{}, apperrors.New(apperrors.CodeUnauthorized, "invalid email or password", 401)
	}

	token, expiresIn, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return LoginOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to generate token", 500, err)
	}

	return LoginOutput{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User:        toUserOutput(user),
	}, nil
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (UserOutput, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserOutput{}, apperrors.New(apperrors.CodeNotFound, "user not found", 404)
		}
		return UserOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to get user", 500, err)
	}
	return toUserOutput(user), nil
}

func toUserOutput(user domain.User) UserOutput {
	return UserOutput{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
