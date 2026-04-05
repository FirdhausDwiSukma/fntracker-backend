package controllers

import (
	"net/http"

	"finance-tracker/dto"
	"finance-tracker/services"
	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService services.AuthService
}

func NewAuthController(authService services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var req dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	user, err := c.authService.Register(req)
	if err != nil {
		if err.Error() == "email already registered" {
			utils.ErrorResponse(ctx, http.StatusConflict, utils.ErrConflict)
			return
		}
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	utils.SuccessResponse(ctx, http.StatusCreated, "registration successful", user)
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(ctx, http.StatusBadRequest, utils.ErrInvalidInput)
		return
	}

	result, err := c.authService.Login(req)
	if err != nil {
		if err.Error() == "invalid credentials" {
			utils.ErrorResponse(ctx, http.StatusUnauthorized, utils.ErrInvalidCredentials)
			return
		}
		utils.ErrorResponse(ctx, http.StatusInternalServerError, utils.ErrInternalServer)
		return
	}

	// Set JWT cookie (HttpOnly, not readable by JS)
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "jwt",
		Value:    result.Token,
		MaxAge:   86400,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Set CSRF cookie (readable by JS, not HttpOnly)
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    result.CsrfToken,
		MaxAge:   86400,
		Path:     "/",
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})

	utils.SuccessResponse(ctx, http.StatusOK, "login successful", result.User)
}

func (c *AuthController) Me(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	user, err := c.authService.GetUserByID(userID)
	if err != nil || user == nil {
		utils.ErrorResponse(ctx, http.StatusUnauthorized, utils.ErrUnauthorized)
		return
	}
	utils.SuccessResponse(ctx, http.StatusOK, "ok", user)
}

func (c *AuthController) Logout(ctx *gin.Context) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		MaxAge:   0,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		MaxAge:   0,
		Path:     "/",
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})

	utils.SuccessResponse(ctx, http.StatusOK, "logout successful", nil)
}
