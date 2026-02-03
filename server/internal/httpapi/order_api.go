package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// OrderAPI handles HTTP API requests for orders
type OrderAPI struct {
	orderStore     *database.OrderStore
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewOrderAPI creates a new order API handler
func NewOrderAPI(orderStore *database.OrderStore, authMiddleware *auth.Middleware, logger *logging.Logger) *OrderAPI {
	return &OrderAPI{
		orderStore:     orderStore,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers order routes on the given mux
func (api *OrderAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/orders", corsMiddleware(api.authMiddleware.RequireAuth(api.handleOrders)))
	mux.HandleFunc("/api/orders/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleOrderItem)))
}

// handleOrders handles GET (list) and POST (create) for orders
func (api *OrderAPI) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listOrders(w, r)
	case http.MethodPost:
		api.createOrder(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOrderItem handles GET, PUT, and DELETE for individual orders
func (api *OrderAPI) handleOrderItem(w http.ResponseWriter, r *http.Request) {
	// Extract order ID from path: /api/orders/{id}
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/api/orders/")
	if id == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.getOrder(w, r, id)
	case http.MethodPut:
		api.updateOrder(w, r, id)
	case http.MethodDelete:
		api.deleteOrder(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *OrderAPI) listOrders(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	query := r.URL.Query()

	limit := 20
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	includeArchived := query.Get("includeArchived") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response, err := api.orderStore.List(ctx, userID, includeArchived, limit, offset)
	if err != nil {
		api.logger.Error("Failed to list orders", logging.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to list orders",
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *OrderAPI) getOrder(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := api.orderStore.GetByID(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get order", logging.WithFields(map[string]interface{}{
			"order_id": id,
			"error":    err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get order",
		})
		return
	}

	if order == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Order not found",
		})
		return
	}

	// Verify ownership
	if order.UserID != userID {
		api.writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
		return
	}

	api.writeJSON(w, http.StatusOK, order)
}

func (api *OrderAPI) createOrder(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var params models.AddOrderParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if params.TrackingNumber == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Tracking number is required",
		})
		return
	}

	if !params.Carrier.IsValid() {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid carrier",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := api.orderStore.Add(ctx, userID, params)
	if err != nil {
		api.logger.Error("Failed to create order", logging.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create order",
		})
		return
	}

	api.logger.Info("Order created", logging.WithFields(map[string]interface{}{
		"user_id":  userID,
		"order_id": order.ID,
		"carrier":  order.Carrier,
	}))

	api.writeJSON(w, http.StatusCreated, order)
}

func (api *OrderAPI) updateOrder(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	var params models.UpdateOrderParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	params.ID = id

	// Validate carrier if provided
	if params.Carrier != nil && !params.Carrier.IsValid() {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid carrier",
		})
		return
	}

	// Validate status if provided
	if params.Status != nil && !params.Status.IsValid() {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid status",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := api.orderStore.Update(ctx, userID, params)
	if err != nil {
		api.logger.Error("Failed to update order", logging.WithFields(map[string]interface{}{
			"order_id": id,
			"user_id":  userID,
			"error":    err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to update order",
		})
		return
	}

	if order == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Order not found",
		})
		return
	}

	api.logger.Info("Order updated", logging.WithFields(map[string]interface{}{
		"user_id":  userID,
		"order_id": order.ID,
	}))

	api.writeJSON(w, http.StatusOK, order)
}

func (api *OrderAPI) deleteOrder(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := api.orderStore.Delete(ctx, id, userID); err != nil {
		api.logger.Error("Failed to delete order", logging.WithFields(map[string]interface{}{
			"order_id": id,
			"user_id":  userID,
			"error":    err.Error(),
		}))

		if strings.Contains(err.Error(), "not found") {
			api.writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "Order not found",
			})
			return
		}

		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete order",
		})
		return
	}

	api.logger.Info("Order deleted", logging.WithFields(map[string]interface{}{
		"user_id":  userID,
		"order_id": id,
	}))

	api.writeJSON(w, http.StatusNoContent, nil)
}

func (api *OrderAPI) writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	if v == nil {
		w.WriteHeader(statusCode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}
