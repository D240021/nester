package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suncrestlabs/nester/apps/api/internal/domain/vault"
)

type VaultService struct {
	repository vault.Repository
}

type CreateVaultInput struct {
	UserID          uuid.UUID
	ContractAddress string
	Currency        string
	Status          string
}

type RecordDepositInput struct {
	VaultID uuid.UUID
	Amount  decimal.Decimal
}

type UpdateAllocationsInput struct {
	VaultID     uuid.UUID
	Allocations []vault.Allocation
}

func NewVaultService(repository vault.Repository) *VaultService {
	return &VaultService{repository: repository}
}

func (s *VaultService) CreateVault(ctx context.Context, input CreateVaultInput) (vault.Vault, error) {
	if input.UserID == uuid.Nil || strings.TrimSpace(input.ContractAddress) == "" || strings.TrimSpace(input.Currency) == "" {
		return vault.Vault{}, vault.ErrInvalidVault
	}

	status := vault.StatusActive
	if strings.TrimSpace(input.Status) != "" {
		parsedStatus, err := vault.ParseStatus(input.Status)
		if err != nil {
			return vault.Vault{}, err
		}
		status = parsedStatus
	}

	model := vault.Vault{
		ID:              uuid.New(),
		UserID:          input.UserID,
		ContractAddress: strings.TrimSpace(input.ContractAddress),
		TotalDeposited:  decimal.Zero,
		CurrentBalance:  decimal.Zero,
		Currency:        strings.ToUpper(strings.TrimSpace(input.Currency)),
		Status:          status,
	}

	return s.repository.CreateVault(ctx, model)
}

func (s *VaultService) GetVault(ctx context.Context, id uuid.UUID) (vault.Vault, error) {
	if id == uuid.Nil {
		return vault.Vault{}, vault.ErrInvalidVault
	}
	return s.repository.GetVault(ctx, id)
}

func (s *VaultService) GetUserVaults(ctx context.Context, userID uuid.UUID) ([]vault.Vault, error) {
	if userID == uuid.Nil {
		return nil, vault.ErrInvalidVault
	}
	return s.repository.GetUserVaults(ctx, userID)
}

func (s *VaultService) RecordDeposit(ctx context.Context, input RecordDepositInput) (vault.Vault, error) {
	if input.VaultID == uuid.Nil {
		return vault.Vault{}, vault.ErrInvalidVault
	}
	if input.Amount.Cmp(decimal.Zero) <= 0 {
		return vault.Vault{}, vault.ErrInvalidAmount
	}

	currentVault, err := s.repository.GetVault(ctx, input.VaultID)
	if err != nil {
		return vault.Vault{}, err
	}

	totalDeposited := currentVault.TotalDeposited.Add(input.Amount)
	currentBalance := currentVault.CurrentBalance.Add(input.Amount)

	if err := s.repository.UpdateVaultBalances(ctx, input.VaultID, totalDeposited, currentBalance); err != nil {
		return vault.Vault{}, err
	}

	return s.repository.GetVault(ctx, input.VaultID)
}

func (s *VaultService) UpdateAllocations(ctx context.Context, input UpdateAllocationsInput) (vault.Vault, error) {
	if input.VaultID == uuid.Nil {
		return vault.Vault{}, vault.ErrInvalidVault
	}

	normalized := make([]vault.Allocation, 0, len(input.Allocations))
	now := time.Now().UTC()

	for _, allocation := range input.Allocations {
		if strings.TrimSpace(allocation.Protocol) == "" || allocation.Amount.Cmp(decimal.Zero) < 0 || allocation.APY.Cmp(decimal.Zero) < 0 {
			return vault.Vault{}, vault.ErrInvalidAllocation
		}

		if allocation.ID == uuid.Nil {
			allocation.ID = uuid.New()
		}
		if allocation.AllocatedAt.IsZero() {
			allocation.AllocatedAt = now
		}

		allocation.Protocol = strings.ToLower(strings.TrimSpace(allocation.Protocol))
		allocation.VaultID = input.VaultID
		normalized = append(normalized, allocation)
	}

	if err := s.repository.ReplaceAllocations(ctx, input.VaultID, normalized); err != nil {
		return vault.Vault{}, err
	}

	return s.repository.GetVault(ctx, input.VaultID)
}
