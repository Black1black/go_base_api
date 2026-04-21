package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) processOperation(c *gin.Context) {
	var req struct {
		WalletID      string `json:"walletId"`
		OperationType string `json:"operationType"`
		Amount        int64  `json:"amount"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	walletID, err := uuid.Parse(req.WalletID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid wallet id"})
		return
	}

	err = h.walletUC.ProcessOperation(c.Request.Context(), walletID, req.OperationType, req.Amount)
	if err != nil {
		if err.Error() == "wallet not found" {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "insufficient funds" {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}

// getBalance - получение баланса
func (h *Handler) getBalance(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("WALLET_UUID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid wallet id"})
		return
	}

	balance, err := h.walletUC.GetWalletBalance(c.Request.Context(), walletID)
	if err != nil {
		if err.Error() == "wallet not found" {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"balance": balance})
}
