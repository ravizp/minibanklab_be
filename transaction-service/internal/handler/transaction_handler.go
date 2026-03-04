package handler

import (
	"net/http"

	"transaction-service/internal/model"
	"transaction-service/internal/service"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	svc service.TransactionService
}

func NewTransactionHandler(svc service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

// Transfer godoc
// @Summary      Transfer money between accounts
// @Description  Transfer funds from one account to another. Uses DB transactions with row locking (SELECT FOR UPDATE) for ACID safety. Amount must be a numeric string.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        body  body      model.TransferRequest  true  "Transfer payload"
// @Success      200   {object}  model.TransactionResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      401   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /transactions/transfer [post]
func (h *TransactionHandler) Transfer(c *gin.Context) {
	var req model.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	tx, err := h.svc.Transfer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx.ToResponse())
}

// TopUp godoc
// @Summary      Top up account balance
// @Description  Add funds to a bank account. Amount must be a numeric string.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        body  body      model.TopUpRequest  true  "Top up payload"
// @Success      200   {object}  model.TransactionResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      401   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /transactions/topup [post]
func (h *TransactionHandler) TopUp(c *gin.Context) {
	var req model.TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	tx, err := h.svc.TopUp(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx.ToResponse())
}

// GetHistory godoc
// @Summary      Get transaction history
// @Description  Retrieve all transactions for a given account
// @Tags         transactions
// @Produce      json
// @Param        account_id  path      string  true  "Account ID (UUID)"
// @Success      200         {array}   model.TransactionResponse
// @Failure      400         {object}  model.ErrorResponse
// @Failure      401         {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /transactions/history/{account_id} [get]
func (h *TransactionHandler) GetHistory(c *gin.Context) {
	accountID := c.Param("account_id")

	transactions, err := h.svc.GetHistory(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	var responses []model.TransactionResponse
	for _, t := range transactions {
		responses = append(responses, t.ToResponse())
	}

	if responses == nil {
		responses = []model.TransactionResponse{}
	}

	c.JSON(http.StatusOK, responses)
}
