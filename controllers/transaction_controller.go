package controllers

import (
	"math"
	"net/http"
	"strconv"

	"finance-tracker/dto"
	"finance-tracker/services"
	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

type TransactionController struct {
	txService services.TransactionService
}

func NewTransactionController(txService services.TransactionService) *TransactionController {
	return &TransactionController{txService: txService}
}

func (c *TransactionController) GetAll(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	filter := dto.TransactionFilter{
		Type:      ctx.Query("type"),
		StartDate: ctx.Query("start_date"),
		EndDate:   ctx.Query("end_date"),
	}

	if v := ctx.Query("category_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			filter.CategoryID = uint(id)
		}
	}
	if v := ctx.Query("month"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			filter.Month = m
		}
	}
	if v := ctx.Query("year"); v != "" {
		if y, err := strconv.Atoi(v); err == nil {
			filter.Year = y
		}
	}
	if v := ctx.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := ctx.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil {
			filter.Limit = l
		}
	}

	// Apply defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	transactions, total, err := c.txService.GetAllByUser(userID, filter)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	responses := make([]dto.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		responses[i] = dto.ToTransactionResponse(tx)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":        responses,
		"total":       total,
		"page":        filter.Page,
		"limit":       filter.Limit,
		"total_pages": totalPages,
	})
}

func (c *TransactionController) Create(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var req dto.TransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	tx, err := c.txService.Create(userID, req)
	if err != nil {
		switch err.Error() {
		case "invalid category":
			utils.ErrorResponse(ctx, http.StatusBadRequest, "invalid category")
		case "invalid date format, expected YYYY-MM-DD":
			utils.ErrorResponse(ctx, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		}
		return
	}

	utils.SuccessResponse(ctx, http.StatusCreated, "transaction created", dto.ToTransactionResponsePtr(tx))
}

func (c *TransactionController) Update(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	idParam := ctx.Param("id")
	txID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	var req dto.TransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	tx, err := c.txService.Update(userID, uint(txID), req)
	if err != nil {
		switch err.Error() {
		case "not found":
			utils.ErrorResponse(ctx, http.StatusNotFound, utils.ErrNotFound)
		case "invalid category":
			utils.ErrorResponse(ctx, http.StatusBadRequest, "invalid category")
		case "invalid date format, expected YYYY-MM-DD":
			utils.ErrorResponse(ctx, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		}
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "transaction updated", dto.ToTransactionResponsePtr(tx))
}

func (c *TransactionController) Delete(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	idParam := ctx.Param("id")
	txID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	if err := c.txService.Delete(userID, uint(txID)); err != nil {
		if err.Error() == "not found" {
			utils.ErrorResponse(ctx, http.StatusNotFound, utils.ErrNotFound)
			return
		}
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "transaction deleted", nil)
}

func (c *TransactionController) Export(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	filter := dto.ExportFilter{
		StartDate: ctx.Query("start_date"),
		EndDate:   ctx.Query("end_date"),
	}

	csvBytes, err := c.txService.ExportCSV(userID, filter)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	ctx.Header("Content-Type", "text/csv")
	ctx.Header("Content-Disposition", `attachment; filename="transactions.csv"`)
	ctx.Data(http.StatusOK, "text/csv", csvBytes)
}
