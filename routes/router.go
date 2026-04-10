package routes

import (
	"os"

	"finance-tracker/config"
	"finance-tracker/controllers"
	"finance-tracker/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	cfg *config.Config,
	authCtrl *controllers.AuthController,
	categoryCtrl *controllers.CategoryController,
	transactionCtrl *controllers.TransactionController,
	budgetCtrl *controllers.BudgetController,
	dashboardCtrl *controllers.DashboardController,
) *gin.Engine {
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	middlewares := []gin.HandlerFunc{
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware(cfg.AllowedOrigins),
		gin.Recovery(),
	}
	// Only enable request logger in non-production to avoid leaking sensitive data
	if os.Getenv("ENV") != "production" {
		middlewares = append([]gin.HandlerFunc{gin.Logger()}, middlewares...)
	}
	r.Use(middlewares...)

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/login", middleware.LoginRateLimiter(), authCtrl.Login)
		auth.POST("/logout", authCtrl.Logout)
		auth.GET("/me", middleware.JWTAuthMiddleware(cfg.JWTSecret), authCtrl.Me)
	}

	protected := r.Group("/api")
	protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware())
	{
		protected.GET("/categories", categoryCtrl.GetAll)
		protected.POST("/categories", categoryCtrl.Create)
		protected.PUT("/categories/:id", categoryCtrl.Update)
		protected.DELETE("/categories/:id", categoryCtrl.Delete)

		protected.GET("/transactions", transactionCtrl.GetAll)
		protected.GET("/transactions/export", transactionCtrl.Export)
		protected.POST("/transactions", transactionCtrl.Create)
		protected.PUT("/transactions/:id", transactionCtrl.Update)
		protected.DELETE("/transactions/:id", transactionCtrl.Delete)

		protected.GET("/budgets", budgetCtrl.GetAll)
		protected.POST("/budgets", budgetCtrl.Create)
		protected.PUT("/budgets/:id", budgetCtrl.Update)
		protected.DELETE("/budgets/:id", budgetCtrl.Delete)

		protected.GET("/dashboard", dashboardCtrl.GetSummary)
	}

	return r
}
