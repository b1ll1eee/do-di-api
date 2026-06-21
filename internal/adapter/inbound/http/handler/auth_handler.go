package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/inbound/http/middleware"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/pkg/response"
)

// AuthHandler handles HTTP requests for authentication endpoints.
type AuthHandler struct {
	authSvc inbound.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authSvc inbound.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// registerRequest is the JSON body for the register endpoint.
type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// loginRequest is the JSON body for the login endpoint.
type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// authResponse is the JSON body returned on success.
type authResponse struct {
	Token string      `json:"token"`
	User  userPayload `json:"user"`
}

type userPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Me godoc
//
//	@Summary		Get current user
//	@Description	Return the authenticated user's profile
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.envelope{data=userPayload}
//	@Failure		401	{object}	response.envelope
//	@Failure		404	{object}	response.envelope
//	@Router			/api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	val, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		response.InternalServerError(c, "invalid user identity")
		return
	}

	user, err := h.authSvc.GetMe(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, "failed to get user")
		return
	}

	response.OK(c, userPayload{ID: user.ID.String(), Email: user.Email})
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Create a new account and receive a JWT
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		registerRequest	true	"Registration payload"
//	@Success		201		{object}	response.envelope{data=authResponse}
//	@Failure		400		{object}	response.envelope
//	@Failure		409		{object}	response.envelope
//	@Failure		500		{object}	response.envelope
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.authSvc.Register(c.Request.Context(), inbound.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalServerError(c, "registration failed")
		return
	}

	response.Created(c, authResponse{
		Token: result.Token,
		User:  userPayload{ID: result.User.ID.String(), Email: result.User.Email},
	})
}

// Login godoc
//
//	@Summary		Authenticate a user
//	@Description	Exchange credentials for a JWT
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login payload"
//	@Success		200		{object}	response.envelope{data=authResponse}
//	@Failure		400		{object}	response.envelope
//	@Failure		401		{object}	response.envelope
//	@Failure		500		{object}	response.envelope
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), inbound.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.Unauthorized(c, err.Error())
			return
		}
		response.InternalServerError(c, "login failed")
		return
	}

	response.OK(c, authResponse{
		Token: result.Token,
		User:  userPayload{ID: result.User.ID.String(), Email: result.User.Email},
	})
}
