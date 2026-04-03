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

	// TODO: Connect to database and wire all components.
	// db, err := config.ConnectDatabase(cfg.DBUrl,
	//     &models.User{}, &models.Category{}, &models.Transaction{}, &models.Budget{},
	// )
	// if err != nil {
	//     log.Fatalf("Failed to connect to database: %v", err)
	// }
	//
	// Repositories
	// userRepo := repositories.NewUserRepository(db)
	// categoryRepo := repositories.NewCategoryRepository(db)
	// transactionRepo := repositories.NewTransactionRepository(db)
	// budgetRepo := repositories.NewBudgetRepository(db)
	//
	// Services
	// authSvc := services.NewAuthService(userRepo, cfg.JWTSecret)
	// categorySvc := services.NewCategoryService(categoryRepo)
	// transactionSvc := services.NewTransactionService(transactionRepo, categoryRepo)
	// budgetSvc := services.NewBudgetService(budgetRepo, transactionRepo)
	// dashboardSvc := services.NewDashboardService(transactionRepo, budgetRepo)
	//
	// Controllers
	// authCtrl := controllers.NewAuthController(authSvc)
	// categoryCtrl := controllers.NewCategoryController(categorySvc)
	// transactionCtrl := controllers.NewTransactionController(transactionSvc)
	// budgetCtrl := controllers.NewBudgetController(budgetSvc)
	// dashboardCtrl := controllers.NewDashboardController(dashboardSvc)
	//
	// Router
	// router := routes.SetupRouter(cfg, authCtrl, categoryCtrl, transactionCtrl, budgetCtrl, dashboardCtrl)

	// Placeholder HTTP server — will be replaced with Gin router once all components are wired.
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"Finance Tracker API is running"}`))
		}),
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
