package vault

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type VaultStatus string

const (
	StatusActive VaultStatus = "active"
	StatusPaused VaultStatus = "paused"
	StatusClosed VaultStatus = "closed"
)

var (
	ErrVaultNotFound     = errors.New("vault not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidVault      = errors.New("invalid vault input")
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrInvalidAllocation = errors.New("invalid allocation input")
)

type Vault struct {
	ID              uuid.UUID       `json:"id"`
	UserID          uuid.UUID       `json:"user_id"`
	ContractAddress string          `json:"contract_address"`
	TotalDeposited  decimal.Decimal `json:"total_deposited"`
	CurrentBalance  decimal.Decimal `json:"current_balance"`
	Currency        string          `json:"currency"`
	Status          VaultStatus     `json:"status"`
	Allocations     []Allocation    `json:"allocations,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Allocation struct {
	ID          uuid.UUID       `json:"id"`
	VaultID     uuid.UUID       `json:"vault_id"`
	Protocol    string          `json:"protocol"`
	Amount      decimal.Decimal `json:"amount"`
	APY         decimal.Decimal `json:"apy"`
	AllocatedAt time.Time       `json:"allocated_at"`
}

type Repository interface {
	CreateVault(ctx context.Context, model Vault) (Vault, error)
	GetVault(ctx context.Context, id uuid.UUID) (Vault, error)
	GetUserVaults(ctx context.Context, userID uuid.UUID) ([]Vault, error)
	UpdateVaultBalances(ctx context.Context, id uuid.UUID, totalDeposited decimal.Decimal, currentBalance decimal.Decimal) error
	ReplaceAllocations(ctx context.Context, vaultID uuid.UUID, allocations []Allocation) error
}

func ParseStatus(value string) (VaultStatus, error) {
	switch VaultStatus(strings.ToLower(strings.TrimSpace(value))) {
	case StatusActive:
		return StatusActive, nil
	case StatusPaused:
		return StatusPaused, nil
	case StatusClosed:
		return StatusClosed, nil
	default:
		return "", ErrInvalidVault
	}
}
