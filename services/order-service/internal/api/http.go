package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/repository"
)

// OrderStore is the persistence contract the handler needs. Defining it
// here (the consumer), rather than in the repository package, keeps the
// handler unit-testable with an in-memory fake instead of a real Postgres
// connection — *repository.OrderRepository satisfies this implicitly.
type OrderStore interface {
	Save(ctx context.Context, o *domain.Order) error
	FindByID(ctx context.Context, id string) (*domain.Order, error)
}

// EventPublisher is the outbound-event contract the handler needs.
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, o *domain.Order) error
}

// Handler wires HTTP transport to the domain/repository/publisher layers.
// It stays deliberately thin — no business logic lives here.
type Handler struct {
	repo      OrderStore
	publisher EventPublisher
	logger    *slog.Logger
}

func NewHandler(repo OrderStore, publisher EventPublisher, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, publisher: publisher, logger: logger}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.Handle("GET /metrics", promhttp.Handler())
	return mux
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id"`
	ItemSKU    string `json:"item_sku"`
	Quantity   int    `json:"quantity"`
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := domain.NewOrder(req.CustomerID, req.ItemSKU, req.Quantity)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	ctx := r.Context()

	if err := h.repo.Save(ctx, order); err != nil {
		h.logger.Error("save order failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to save order")
		return
	}

	// Publish after commit: consumers should only ever see orders that
	// actually exist. If the publish fails, the order is still durable
	// in Postgres and can be re-published by a reconciliation job —
	// that job is a documented follow-up, not implemented here.
	if err := h.publisher.PublishOrderCreated(ctx, order); err != nil {
		h.logger.Error("publish order created event failed", "error", err, "order_id", order.ID)
	}

	writeJSON(w, http.StatusCreated, order)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	order, err := h.repo.FindByID(r.Context(), id)
	if errors.Is(err, repository.ErrOrderNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		h.logger.Error("find order failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch order")
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
