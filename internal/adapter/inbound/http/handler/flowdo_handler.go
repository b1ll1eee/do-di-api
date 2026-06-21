package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/inbound/http/middleware"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/pkg/response"
)

const (
	defaultLimit  = 20
	defaultOffset = 0
	maxLimit      = 100
)

// FlowdoHandler handles HTTP requests for flowdo endpoints.
type FlowdoHandler struct {
	flowdoSvc inbound.FlowdoService
}

// NewFlowdoHandler constructs a FlowdoHandler.
func NewFlowdoHandler(flowdoSvc inbound.FlowdoService) *FlowdoHandler {
	return &FlowdoHandler{flowdoSvc: flowdoSvc}
}

// createFlowdoRequest is the JSON body for creating a flowdo.
type createFlowdoRequest struct {
	Title       string `json:"title"       binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

// updateFlowdoRequest is the JSON body for updating a flowdo.
type updateFlowdoRequest struct {
	Title       string        `json:"title"       binding:"omitempty,min=1,max=255"`
	Description string        `json:"description" binding:"omitempty,max=1000"`
	Status      domain.Status `json:"status"      binding:"omitempty,oneof=pending in_progress done"`
}

// flowdoResponse is the JSON representation of a flowdo item.
type flowdoResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	DeletedAt   *string `json:"deleted_at,omitempty"`
}

func toFlowdoResponse(t *domain.Flowdo) flowdoResponse {
	r := flowdoResponse{
		ID:          t.ID.String(),
		UserID:      t.UserID.String(),
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		CreatedAt:   t.CreatedAt.String(),
		UpdatedAt:   t.UpdatedAt.String(),
	}
	if t.DeletedAt != nil {
		s := t.DeletedAt.String()
		r.DeletedAt = &s
	}
	return r
}

func extractUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return uuid.Nil, false
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		response.InternalServerError(c, "invalid user identity")
		return uuid.Nil, false
	}
	return userID, true
}

// Create godoc
//
//	@Summary		Create a flowdo
//	@Description	Create a new flowdo item for the authenticated user
//	@Tags			flowdos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		createFlowdoRequest	true	"Flowdo payload"
//	@Success		201		{object}	response.envelope{data=flowdoResponse}
//	@Failure		400		{object}	response.envelope
//	@Failure		401		{object}	response.envelope
//	@Failure		500		{object}	response.envelope
//	@Router			/api/v1/flowdos [post]
func (h *FlowdoHandler) Create(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req createFlowdoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	flowdo, err := h.flowdoSvc.Create(c.Request.Context(), inbound.CreateFlowdoInput{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		response.InternalServerError(c, "failed to create flowdo")
		return
	}

	response.Created(c, toFlowdoResponse(flowdo))
}

// GetByID godoc
//
//	@Summary		Get a flowdo by ID
//	@Description	Retrieve a single flowdo owned by the authenticated user
//	@Tags			flowdos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Flowdo UUID"
//	@Success		200	{object}	response.envelope{data=flowdoResponse}
//	@Failure		400	{object}	response.envelope
//	@Failure		401	{object}	response.envelope
//	@Failure		403	{object}	response.envelope
//	@Failure		404	{object}	response.envelope
//	@Router			/api/v1/flowdos/{id} [get]
func (h *FlowdoHandler) GetByID(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid flowdo id")
		return
	}

	flowdo, err := h.flowdoSvc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		mapFlowdoError(c, err)
		return
	}

	response.OK(c, toFlowdoResponse(flowdo))
}

// List godoc
//
//	@Summary		List flowdos
//	@Description	Return a paginated list of flowdos for the authenticated user
//	@Tags			flowdos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string	false	"Filter by status (pending|in_progress|done)"
//	@Param			limit	query		int		false	"Page size (max 100)"
//	@Param			offset	query		int		false	"Page offset"
//	@Success		200		{object}	response.envelope{data=[]flowdoResponse}
//	@Failure		401		{object}	response.envelope
//	@Failure		500		{object}	response.envelope
//	@Router			/api/v1/flowdos [get]
func (h *FlowdoHandler) List(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	limit := parseIntQuery(c.Query("limit"), defaultLimit)
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := parseIntQuery(c.Query("offset"), defaultOffset)
	status := c.Query("status")

	if status != "" && !domain.IsValidStatus(status) {
		response.BadRequest(c, "status must be one of: pending, in_progress, done")
		return
	}

	result, err := h.flowdoSvc.List(c.Request.Context(), inbound.ListFlowdosFilter{
		UserID: userID,
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		response.InternalServerError(c, "failed to list flowdos")
		return
	}

	items := make([]flowdoResponse, 0, len(result.Items))
	for _, t := range result.Items {
		items = append(items, toFlowdoResponse(t))
	}

	response.OKWithMeta(c, items, response.Meta{
		Limit:  result.Limit,
		Offset: result.Offset,
		Total:  result.Total,
	})
}

// Update godoc
//
//	@Summary		Update a flowdo
//	@Description	Update title, description, or status of a flowdo
//	@Tags			flowdos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Flowdo UUID"
//	@Param			body	body		updateFlowdoRequest	true	"Update payload"
//	@Success		200		{object}	response.envelope{data=flowdoResponse}
//	@Failure		400		{object}	response.envelope
//	@Failure		401		{object}	response.envelope
//	@Failure		403		{object}	response.envelope
//	@Failure		404		{object}	response.envelope
//	@Failure		422		{object}	response.envelope
//	@Router			/api/v1/flowdos/{id} [put]
func (h *FlowdoHandler) Update(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid flowdo id")
		return
	}

	var req updateFlowdoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	flowdo, err := h.flowdoSvc.Update(c.Request.Context(), id, userID, inbound.UpdateFlowdoInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		mapFlowdoError(c, err)
		return
	}

	response.OK(c, toFlowdoResponse(flowdo))
}

// Delete godoc
//
//	@Summary		Delete a flowdo
//	@Description	Soft-delete a flowdo (sets deleted_at timestamp)
//	@Tags			flowdos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Flowdo UUID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.envelope
//	@Failure		401	{object}	response.envelope
//	@Failure		403	{object}	response.envelope
//	@Failure		404	{object}	response.envelope
//	@Router			/api/v1/flowdos/{id} [delete]
func (h *FlowdoHandler) Delete(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid flowdo id")
		return
	}

	if err := h.flowdoSvc.Delete(c.Request.Context(), id, userID); err != nil {
		mapFlowdoError(c, err)
		return
	}

	response.NoContent(c)
}

// Reorder godoc
//
//	@Summary		Reorder flowdos
//	@Description	Set the display order of flowdos by providing an ordered list of IDs
//	@Tags			flowdos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		reorderRequest	true	"Ordered flowdo IDs"
//	@Success		200		{object}	response.envelope
//	@Failure		400		{object}	response.envelope
//	@Failure		401		{object}	response.envelope
//	@Failure		500		{object}	response.envelope
//	@Router			/api/v1/flowdos/reorder [patch]
func (h *FlowdoHandler) Reorder(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	orderedIDs := make([]uuid.UUID, 0, len(req.OrderedIDs))
	for _, s := range req.OrderedIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid flowdo id: "+s)
			return
		}
		orderedIDs = append(orderedIDs, id)
	}

	if err := h.flowdoSvc.Reorder(c.Request.Context(), userID, orderedIDs); err != nil {
		response.InternalServerError(c, "failed to reorder flowdos")
		return
	}

	response.OK(c, nil)
}

// reorderRequest is the JSON body for reordering flowdos.
type reorderRequest struct {
	OrderedIDs []string `json:"orderedIds" binding:"required"`
}

// mapFlowdoError converts domain errors into appropriate HTTP responses.
func mapFlowdoError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrFlowdoNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domain.ErrFlowdoDeleted):
		response.NotFound(c, "flowdo not found")
	case errors.Is(err, domain.ErrUnauthorized):
		response.Forbidden(c, err.Error())
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		response.UnprocessableEntity(c, err.Error())
	default:
		response.InternalServerError(c, "internal server error")
	}
}

func parseIntQuery(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
