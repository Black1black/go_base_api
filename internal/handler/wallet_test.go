package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockWalletUsecase struct {
	processOperationFunc func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) error
	getBalanceFunc       func(ctx context.Context, walletID uuid.UUID) (int64, error)
}

func (m *mockWalletUsecase) ProcessOperation(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) error {
	return m.processOperationFunc(ctx, walletID, operationType, amount)
}

func (m *mockWalletUsecase) GetWalletBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	return m.getBalanceFunc(ctx, walletID)
}

func setupTestRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return h.InitRoutes()
}

func TestHandler_ProcessOperation_DepositSuccess(t *testing.T) {
	mockUC := &mockWalletUsecase{
		processOperationFunc: func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) error {
			return nil
		},
	}

	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	walletID := uuid.New()
	reqBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "DEPOSIT",
		"amount":        100,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandler_ProcessOperation_InvalidJSON(t *testing.T) {
	mockUC := &mockWalletUsecase{}
	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	req := httptest.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_ProcessOperation_InvalidUUID(t *testing.T) {
	mockUC := &mockWalletUsecase{}
	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	reqBody := map[string]interface{}{
		"walletId":      "invalid-uuid",
		"operationType": "DEPOSIT",
		"amount":        100,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_ProcessOperation_InsufficientFunds(t *testing.T) {
	mockUC := &mockWalletUsecase{
		processOperationFunc: func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) error {
			return errors.New("insufficient funds")
		},
	}

	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	walletID := uuid.New()
	reqBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "WITHDRAW",
		"amount":        1000,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_GetBalance_Success(t *testing.T) {
	walletID := uuid.New()
	mockUC := &mockWalletUsecase{
		getBalanceFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
			return 500, nil
		},
	}

	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]int64
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["balance"] != 500 {
		t.Errorf("Expected balance 500, got %d", response["balance"])
	}
}

func TestHandler_GetBalance_InvalidUUID(t *testing.T) {
	mockUC := &mockWalletUsecase{}
	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/wallets/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_GetBalance_WalletNotFound(t *testing.T) {
	walletID := uuid.New()
	mockUC := &mockWalletUsecase{
		getBalanceFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
			return 0, errors.New("wallet not found")
		},
	}

	h := NewHandler(mockUC)
	router := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
