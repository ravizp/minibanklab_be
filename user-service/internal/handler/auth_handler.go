package handler

import (
	"net/http"

	"user-service/internal/model"
	"user-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc       service.AuthService
	jwtSecret string
}

func NewAuthHandler(svc service.AuthService, jwtSecret string) *AuthHandler {
	return &AuthHandler{svc: svc, jwtSecret: jwtSecret}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with name, email, and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RegisterRequest  true  "Registration payload"
// @Success      201   {object}  model.UserResponse
// @Failure      400   {object}  model.ErrorResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	resp := user.ToResponse()
	c.JSON(http.StatusCreated, resp)
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate with email and password, returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.LoginRequest   true  "Login credentials"
// @Success      200   {object}  model.LoginResponse
// @Failure      401   {object}  model.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Profile godoc
// @Summary      Get user profile
// @Description  Returns the profile of the authenticated user
// @Tags         auth
// @Produce      json
// @Success      200  {object}  model.UserResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/profile [get]
func (h *AuthHandler) Profile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized: user identity not found in token"})
		return
	}

	user, err := h.svc.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "user not found"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// ProfileDetail godoc
// @Summary      Get detailed user profile
// @Description  Returns a comprehensive profile combining all user columns with computed metadata
// @Tags         auth
// @Produce      json
// @Success      200  {object}  model.ProfileDetailResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/profile/detail [get]
func (h *AuthHandler) ProfileDetail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized: user identity not found in token"})
		return
	}

	user, err := h.svc.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "user not found"})
		return
	}

	c.JSON(http.StatusOK, user.ToDetailResponse())
}
