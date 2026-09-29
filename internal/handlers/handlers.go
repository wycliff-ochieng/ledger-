package handlers

import (
	"Wycliff-Ochieng/internal/models"
	"Wycliff-Ochieng/internal/repository"
	"Wycliff-Ochieng/internal/service"
	"encoding/json"
	"errors"
	"net/http"
)

type LedgerHandler struct {
	svc service.LedgerService
}

func NewLedgerHandler(svc service.LedgerService) *LedgerHandler {
	return &LedgerHandler{svc: svc}
}

type CreateAccountReq struct {
	Name string             `json:"name"`
	Type models.AccountType `json:"type"`
}

func (h *LedgerHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	acc, err := h.svc.CreateAccount(r.Context(), req.Name, req.Type)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(acc)
}

func (h *LedgerHandler) PostTransaction(w http.ResponseWriter, r *http.Request) {
	var req service.PostTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.svc.PostTransaction(r.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateIdempotencyKey) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, models.ErrUnbalancedLedger) || errors.Is(err, models.ErrInsuffucientEntries) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tx)
}

func (h *LedgerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/accounts", h.CreateAccount)
	mux.HandleFunc("POST /v1/transactions", h.PostTransaction)
}
