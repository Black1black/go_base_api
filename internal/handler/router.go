package handler

import (
	"github.com/gin-gonic/gin"
)

type Handler struct {
	walletUC WalletUsecase
}

func NewHandler(walletUC WalletUsecase) *Handler {
	return &Handler{walletUC: walletUC}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	api := router.Group("/api/v1")
	{
		api.POST("/wallet", h.processOperation)
		api.GET("/wallets/:WALLET_UUID", h.getBalance)
	}

	return router
}
