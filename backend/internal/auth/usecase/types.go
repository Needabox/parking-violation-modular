package usecase

import (
	"time"

	"github.com/google/uuid"
)

type UserOutput struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginOutput struct {
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresIn   int64      `json:"expires_in"`
	User        UserOutput `json:"user"`
}

type VehicleOutput struct {
	ID          uuid.UUID `json:"id"`
	PlateNumber string    `json:"plate_number"`
	CreatedAt   time.Time `json:"created_at"`
}

type PaginationMeta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type VehicleListOutput struct {
	Data []VehicleOutput `json:"data"`
	Meta PaginationMeta  `json:"meta"`
}
