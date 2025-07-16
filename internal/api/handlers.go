package api

import (
	"net/http"

	"github.com/alhamdibahri/my-echo-app/internal/config"
	"github.com/alhamdibahri/my-echo-app/internal/db"
	"github.com/alhamdibahri/my-echo-app/internal/rabbitmq"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handlers struct {
	Rabbit *rabbitmq.Manager
	DB     *db.Postgres
	Cfg    *config.Config
}

func RegisterRoutes(e *echo.Echo, rabbit *rabbitmq.Manager, db *db.Postgres, cfg *config.Config) {
	h := &Handlers{
		Rabbit: rabbit,
		DB:     db,
		Cfg:    cfg,
	}

	e.POST("/tenants", h.CreateTenant)
	e.DELETE("/tenants/:id", h.DeleteTenant)
	e.PUT("/tenants/:id/config/concurrency", h.UpdateConcurrency)
	e.GET("/messages", h.GetMessages)
}

// CreateTenant godoc
// @Summary Create new tenant
// @Description Creates a new tenant queue and consumer
// @Tags tenants
// @Produce json
// @Success 201 {object} map[string]string
// @Router /tenants [post]
func (h *Handlers) CreateTenant(c echo.Context) error {
	tenantID := uuid.New().String()

	err := h.Rabbit.CreateTenantQueue(tenantID, h.Cfg.Workers)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"tenant_id": tenantID})
}

// DeleteTenant godoc
// @Summary Delete tenant
// @Description Stops consumer, deletes queue
// @Tags tenants
// @Param id path string true "Tenant ID"
// @Success 204
// @Router /tenants/{id} [delete]
func (h *Handlers) DeleteTenant(c echo.Context) error {
	tenantID := c.Param("id")

	err := h.Rabbit.StopTenant(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateConcurrency godoc
// @Summary Update concurrency
// @Description Updates worker pool size for tenant
// @Tags tenants
// @Param id path string true "Tenant ID"
// @Accept json
// @Param request body map[string]int true "Concurrency config"
// @Success 204
// @Router /tenants/{id}/config/concurrency [put]
func (h *Handlers) UpdateConcurrency(c echo.Context) error {
	tenantID := c.Param("id")

	var req struct {
		Workers int `json:"workers"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	err := h.Rabbit.UpdateConcurrency(tenantID, req.Workers)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessages godoc
// @Summary Get messages with cursor pagination
// @Description Retrieves messages using cursor-based pagination
// @Tags messages
// @Param cursor query string false "Cursor"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /messages [get]
func (h *Handlers) GetMessages(c echo.Context) error {
	cursor := c.QueryParam("cursor")
	if cursor == "" {
		cursor = "00000000-0000-0000-0000-000000000000" // Default start
	}

	messages, nextCursor, err := h.DB.GetMessages(cursor, 10)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":        messages,
		"next_cursor": nextCursor,
	})
}
