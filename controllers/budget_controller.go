package controllers

import (
	"net/http"
	"strconv"

	"finance-tracker/dto"
	"finance-tracker/services"
	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

type BudgetController struct {
	budgetService services.BudgetService
}

func NewBudgetController(budgetService services.BudgetService) *BudgetController {
	return &BudgetController{budgetService: budgetService}
}

func (c *BudgetController) GetAll(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	month, _ := strconv.Atoi(ctx.Query("month"))
	year, _ := strconv.Atoi(ctx.Query("year"))

	budgets, err := c.budgetService.GetAllByUser(userID, month, year)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "budgets retrieved", budgets)
}

func (c *BudgetController) Create(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var req dto.BudgetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	budget, err := c.budgetService.Create(userID, req)
	if err != nil {
		switch err.Error() {
		case "budget already exists":
			utils.ErrorResponse(ctx, http.StatusConflict, utils.ErrConflict)
		case "category not found":
			utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		default:
			utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		}
		return
	}

	utils.SuccessResponse(ctx, http.StatusCreated, "budget created", budget)
}

func (c *BudgetController) Update(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	idParam := ctx.Param("id")
	budgetID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	var req dto.BudgetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	budget, err := c.budgetService.Update(userID, uint(budgetID), req)
	if err != nil {
		switch err.Error() {
		case "not found":
			utils.ErrorResponse(ctx, http.StatusNotFound, utils.ErrNotFound)
		case "budget already exists":
			utils.ErrorResponse(ctx, http.StatusConflict, utils.ErrConflict)
		case "category not found":
			utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		default:
			utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		}
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "budget updated", budget)
}

func (c *BudgetController) Delete(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	idParam := ctx.Param("id")
	budgetID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	if err := c.budgetService.Delete(userID, uint(budgetID)); err != nil {
		if err.Error() == "not found" {
			utils.ErrorResponse(ctx, http.StatusNotFound, utils.ErrNotFound)
			return
		}
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "budget deleted", nil)
}
