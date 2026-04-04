package routes

import (
	"finance-tracker/config"
	"finance-tracker/controllers"
	"finance-tracker/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	cfg *config.Config,
	authCtrl *controllers.AuthController,
	categoryCtrl *controllers.CategoryController,
) *gin.Engine {
	r := gin.New()

	r.Use(
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware(cfg.AllowedOrigins),
		gin.Logger(),
		gin.Recovery(),
	)

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/login", middleware.LoginRateLimiter(), authCtrl.Login)
		auth.POST("/logout", authCtrl.Logout)
	}

	protected := r.Group("/api")
	protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware())
	{
		protected.GET("/categories", categoryCtrl.GetAll)
		protected.POST("/categories", categoryCtrl.Create)
		protected.PUT("/categories/:id", categoryCtrl.Update)
		protected.DELETE("/categories/:id", categoryCtrl.Delete)
	}

	return r
}
