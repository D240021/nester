package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/suncrestlabs/nester/apps/api/internal/domain/vault"
	"github.com/suncrestlabs/nester/apps/api/internal/service"
	logpkg "github.com/suncrestlabs/nester/apps/api/pkg/logger"
)

type VaultHandler struct {
	service *service.VaultService
}

type createVaultRequest struct {
	UserID          string `json:"user_id"`
	ContractAddress string `json:"contract_address"`
	Currency        string `json:"currency"`
	Status          string `json:"status,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewVaultHandler(service *service.VaultService) *VaultHandler {
	return &VaultHandler{service: service}
}

func (h *VaultHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/vaults", h.createVault)
	mux.HandleFunc("GET /api/v1/vaults/{id}", h.getVault)
	mux.HandleFunc("GET /api/v1/users/{userId}/vaults", h.listUserVaults)
}

func (h *VaultHandler) createVault(w http.ResponseWriter, r *http.Request) {
	var request createVaultRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(request.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user_id must be a valid UUID")
		return
	}

	model, err := h.service.CreateVault(r.Context(), service.CreateVaultInput{
		UserID:          userID,
		ContractAddress: request.ContractAddress,
		Currency:        request.Currency,
		Status:          request.Status,
	})
	if err != nil {
		h.writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, model)
}

func (h *VaultHandler) getVault(w http.ResponseWriter, r *http.Request) {
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "vault id must be a valid UUID")
		return
	}

	model, err := h.service.GetVault(r.Context(), vaultID)
	if err != nil {
		h.writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, model)
}

func (h *VaultHandler) listUserVaults(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "user id must be a valid UUID")
		return
	}

	models, err := h.service.GetUserVaults(r.Context(), userID)
	if err != nil {
		h.writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, models)
}

func (h *VaultHandler) writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, vault.ErrVaultNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, vault.ErrUserNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, vault.ErrInvalidVault), errors.Is(err, vault.ErrInvalidAmount), errors.Is(err, vault.ErrInvalidAllocation):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		logpkg.FromContext(r.Context()).Error("vault handler failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
