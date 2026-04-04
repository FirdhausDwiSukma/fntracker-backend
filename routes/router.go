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

	return r
}
