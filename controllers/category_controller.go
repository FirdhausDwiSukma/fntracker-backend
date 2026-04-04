package controllers

import (
	"net/http"
	"strconv"

	"finance-tracker/dto"
	"finance-tracker/services"
	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

type CategoryController struct {
	categoryService services.CategoryService
}

func NewCategoryController(categoryService services.CategoryService) *CategoryController {
	return &CategoryController{categoryService: categoryService}
}

func (c *CategoryController) GetAll(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	categories, err := c.categoryService.GetAllByUser(userID)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "categories retrieved", categories)
}

func (c *CategoryController) Create(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var req dto.CategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	category, err := c.categoryService.Create(userID, req)
	if err != nil {
		if err.Error() == "category already exists" {
			utils.ErrorResponse(ctx, http.StatusConflict, utils.ErrConflict)
			return
		}
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusCreated, "category created", category)
}

func (c *CategoryController) Update(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	idParam := ctx.Param("id")
	categoryID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	var req dto.CategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	category, err := c.categoryService.Update(userID, uint(categoryID), req)
	if err != nil {
		switch err.Error() {
		case "not found":
			utils.ErrorResponse(ctx, http.StatusNotFound, utils.ErrNotFound)
		case "category already exists":
			utils.ErrorResponse(ctx, http.StatusConflict, utils.ErrConflict)
		default:
			utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		}
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "category updated", category)
}

func (c *CategoryController) Delete(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	idParam := ctx.Param("id")
	categoryID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	if err := c.categoryService.Delete(userID, uint(categoryID)); err != nil {
		if err.Error() == "not found" {
			utils.ErrorResponse(ctx, http.StatusNotFound, utils.ErrNotFound)
			return
		}
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "category deleted", nil)
}
