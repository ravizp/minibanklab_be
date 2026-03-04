package handler

import (
	"net/http"

	"account-service/internal/model"
	"account-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	svc service.AccountService
}

func NewAccountHandler(svc service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

// CreateAccount godoc
// @Summary      Create a new bank account
// @Description  Creates a new bank account for the authenticated user
// @Tags         accounts
// @Produce      json
// @Success      201  {object}  model.AccountResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized: user identity not found in token"})
		return
	}

	account, err := h.svc.CreateAccount(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account.ToResponse())
}

// GetAccount godoc
// @Summary      Get account by ID
// @Description  Retrieve a bank account by its UUID (must be owned by authenticated user)
// @Tags         accounts
// @Produce      json
// @Param        id   path      string  true  "Account ID (UUID)"
// @Success      200  {object}  model.AccountResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /accounts/{id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized: user identity not found in token"})
		return
	}

	account, err := h.svc.GetAccountByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "account not found"})
		return
	}

	if account.UserID.String() != userID.(string) {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "access denied: you do not own this account"})
		return
	}

	c.JSON(http.StatusOK, account.ToResponse())
}

// GetBalance godoc
// @Summary      Get account balance
// @Description  Retrieve the balance of a specific bank account owned by the authenticated user
// @Tags         accounts
// @Produce      json
// @Param        id   path      string  true  "Account ID (UUID)"
// @Success      200  {object}  model.BalanceResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /accounts/{id}/balance [get]
func (h *AccountHandler) GetBalance(c *gin.Context) {
	id := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized: user identity not found in token"})
		return
	}

	account, err := h.svc.GetAccountByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "account not found"})
		return
	}

	if account.UserID.String() != userID.(string) {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "access denied: you do not own this account"})
		return
	}

	c.JSON(http.StatusOK, account.ToBalanceResponse())
}

// GetMyAccounts godoc
// @Summary      Get my accounts
// @Description  Retrieve all bank accounts for the authenticated user
// @Tags         accounts
// @Produce      json
// @Success      200  {array}   model.AccountResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /accounts [get]
func (h *AccountHandler) GetMyAccounts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized: user identity not found in token"})
		return
	}

	accounts, err := h.svc.GetAccountsByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	var responses []model.AccountResponse
	for _, a := range accounts {
		responses = append(responses, a.ToResponse())
	}

	if responses == nil {
		responses = []model.AccountResponse{}
	}

	c.JSON(http.StatusOK, responses)
}
