package handler

import (
	"chat-service/configs/middleware"
	"chat-service/configs/utils"
	"chat-service/internal/models"
	"chat-service/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServerHandler struct {
	serverService service.ServerService
}

func NewServerHandler(serverService service.ServerService) *ServerHandler {
	return &ServerHandler{serverService: serverService}
}

func (h *ServerHandler) CreateServer(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req models.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.serverService.CreateServer(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

func (h *ServerHandler) GetServer(c *gin.Context) {
	idParam := c.Param("id")
	id, err := utils.StringToUint(idParam)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	server, err := h.serverService.GetServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) GetUserServers(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	servers, err := h.serverService.GetUserServers(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, servers)
}

func (h *ServerHandler) UpdateServer(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, _ := utils.StringToUint(c.Param("id"))
	var req models.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.serverService.UpdateServer(c.Request.Context(), id, userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) DeleteServer(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	idParam := c.Param("id")
	id, _ := utils.StringToUint(idParam)

	if err := h.serverService.DeleteServer(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ServerHandler) JoinServer(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req models.JoinServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.serverService.JoinServer(c.Request.Context(), userID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *ServerHandler) LeaveServer(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	serverID, _ := utils.StringToUint(c.Param("id"))
	if err := h.serverService.LeaveServer(c.Request.Context(), serverID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *ServerHandler) GetServerMembers(c *gin.Context) {
	serverID, _ := utils.StringToUint(c.Param("id"))
	members, err := h.serverService.GetServerMembers(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, members)
}

func (h *ServerHandler) RegisterRoutes(r *gin.RouterGroup) {
	servers := r.Group("/servers")
	{

		servers.Use(middleware.Auth())
		servers.POST("", h.CreateServer)
		servers.GET("/:id", h.GetServer)
		servers.GET("/user", h.GetUserServers)
		servers.PUT("/:id", h.UpdateServer)
		servers.DELETE("/:id", h.DeleteServer)
		servers.POST("/:id/join", h.JoinServer)
		servers.POST("/:id/leave", h.LeaveServer)
		servers.GET("/:id/members", h.GetServerMembers)
	}
}
