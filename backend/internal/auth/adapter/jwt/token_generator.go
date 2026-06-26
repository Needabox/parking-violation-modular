package jwtadapter

import (
	"github.com/google/uuid"
	jwtutil "github.com/parking-violation-portal/backend/pkg/jwt"
)

type TokenGenerator struct {
	manager *jwtutil.Manager
}

func NewTokenGenerator(manager *jwtutil.Manager) *TokenGenerator {
	return &TokenGenerator{manager: manager}
}

func (g *TokenGenerator) Generate(userID uuid.UUID, email, role string) (string, int64, error) {
	return g.manager.Generate(userID, email, role)
}
