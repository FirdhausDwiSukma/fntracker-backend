package controllers

import (
	"net/http"
	"strconv"
	"time"

	"finance-tracker/services"
	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	dashboardService services.DashboardService
}

func NewDashboardController(dashboardService services.DashboardService) *DashboardController {
	return &DashboardController{dashboardService: dashboardService}
}

func (c *DashboardController) GetSummary(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	now := time.Now()
	month, _ := strconv.Atoi(ctx.Query("month"))
	year, _ := strconv.Atoi(ctx.Query("year"))

	if month == 0 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}

	summary, err := c.dashboardService.GetSummary(userID, month, year)
	if err != nil {
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusOK, "dashboard retrieved", summary)
}
