package handler

import (
	"ex_proj_go/pkg/auth"
	"time"

	"github.com/gin-gonic/gin"
)

var authTools = auth.GetNewAuthTools("salt", "signingKey", 12*time.Hour, 12*time.Hour)

type Handler struct {
	usrUC  Users
	authUC Authorization
}

func NewHandler(usrUC Users, authUC Authorization) *Handler {
	return &Handler{usrUC: usrUC, authUC: authUC}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	auth := router.Group("/auth")
	{
		auth.POST("/registration", h.registrationUser)
		auth.PATCH("/replace_password", h.getUserToken, h.getUserId, h.getUserModel, h.replacePassword)
		auth.POST("/login", h.login)
		auth.POST("/refresh", h.getUserRefreshToken, h.getUserId, h.getUserModel, h.refresh)
		auth.DELETE("/logout", h.getUserRefreshToken, h.logout)

	}

	users := router.Group("/users")
	{

		users.GET("/user", h.getUserToken, h.getUserId, h.myInfo)
		users.GET("/user:id")
		users.PATCH("/users")
		users.PATCH("/location")

	}

	return router
}

func (h *Handler) getUserModel(c *gin.Context) {
	rawUserId, ok := c.Get(userCtx)
	if !ok {
		return
	}
	userId, ok := rawUserId.(int64)
	if !ok {
		return
	}
	userModel, err := h.usrUC.GetByID(userId)
	if err != nil {
		return
	}
	c.Set(userModelCtx, userModel)

}
