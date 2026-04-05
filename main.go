package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finance-tracker/config"
	"finance-tracker/controllers"
	"finance-tracker/models"
	"finance-tracker/repositories"
	"finance-tracker/routes"
	"finance-tracker/services"
)

func main() {
	// Load configuration from environment.
	cfg := config.Load()

	if cfg.DBUrl == "" {
		log.Println("Warning: DB_URL is not set. Database features will not work.")
	}

	if cfg.JWTSecret == "" {
		log.Println("Warning: JWT_SECRET is not set. Authentication will not work.")
	}

	// Connect DB
	db, err := config.ConnectDatabase(cfg.DBUrl,
		&models.User{}, &models.Category{}, &models.Transaction{}, &models.Budget{},
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Repositories
	userRepo := repositories.NewUserRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	budgetRepo := repositories.NewBudgetRepository(db)

	// Services
	authSvc := services.NewAuthService(userRepo, cfg.JWTSecret)
	categorySvc := services.NewCategoryService(categoryRepo)
	transactionSvc := services.NewTransactionService(transactionRepo, categoryRepo)
	budgetSvc := services.NewBudgetService(budgetRepo, transactionRepo, categoryRepo)
	dashboardSvc := services.NewDashboardService(transactionRepo, budgetRepo)

	// Controllers
	authCtrl := controllers.NewAuthController(authSvc)
	categoryCtrl := controllers.NewCategoryController(categorySvc)
	transactionCtrl := controllers.NewTransactionController(transactionSvc)
	budgetCtrl := controllers.NewBudgetController(budgetSvc)
	dashboardCtrl := controllers.NewDashboardController(dashboardSvc)

	// Router
	router := routes.SetupRouter(cfg, authCtrl, categoryCtrl, transactionCtrl, budgetCtrl, dashboardCtrl)

	// Use router as the HTTP handler
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine for graceful shutdown support.
	go func() {
		log.Printf("Server starting on port %s (env: %s)", cfg.Port, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal (SIGINT / SIGTERM) for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
