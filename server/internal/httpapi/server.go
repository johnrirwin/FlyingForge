package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/rotorlife/internal/aggregator"
	"github.com/johnrirwin/rotorlife/internal/aircraft"
	"github.com/johnrirwin/rotorlife/internal/auth"
	"github.com/johnrirwin/rotorlife/internal/equipment"
	"github.com/johnrirwin/rotorlife/internal/inventory"
	"github.com/johnrirwin/rotorlife/internal/logging"
	"github.com/johnrirwin/rotorlife/internal/models"
	"github.com/johnrirwin/rotorlife/internal/radio"
)

type Server struct {
	agg            *aggregator.Aggregator
	equipmentSvc   *equipment.Service
	inventorySvc   inventory.InventoryManager
	aircraftSvc    *aircraft.Service
	radioSvc       *radio.Service
	authSvc        *auth.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
	server         *http.Server
}

func New(agg *aggregator.Aggregator, equipmentSvc *equipment.Service, inventorySvc inventory.InventoryManager, aircraftSvc *aircraft.Service, radioSvc *radio.Service, authSvc *auth.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *Server {
	return &Server{
		agg:            agg,
		equipmentSvc:   equipmentSvc,
		inventorySvc:   inventorySvc,
		aircraftSvc:    aircraftSvc,
		radioSvc:       radioSvc,
		authSvc:        authSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// News feed routes
	mux.HandleFunc("/api/items", s.corsMiddleware(s.handleGetItems))
	mux.HandleFunc("/api/sources", s.corsMiddleware(s.handleGetSources))
	mux.HandleFunc("/api/refresh", s.corsMiddleware(s.handleRefresh))

	// Auth routes
	if s.authSvc != nil && s.authMiddleware != nil {
		authAPI := NewAuthAPI(s.authSvc, s.authMiddleware, s.logger)
		authAPI.RegisterRoutes(mux, s.corsMiddleware)
	}

	// Equipment and inventory routes
	equipmentAPI := NewEquipmentAPI(s.equipmentSvc, s.inventorySvc, s.authMiddleware, s.logger)
	equipmentAPI.RegisterRoutes(mux, s.corsMiddleware)

	// Aircraft routes
	if s.aircraftSvc != nil && s.authMiddleware != nil {
		aircraftAPI := NewAircraftAPI(s.aircraftSvc, s.authMiddleware, s.logger)
		aircraftAPI.RegisterRoutes(mux, s.corsMiddleware)
	}

	// Radio routes
	if s.radioSvc != nil && s.authMiddleware != nil {
		radioAPI := NewRadioAPI(s.radioSvc, s.authMiddleware, s.logger)
		radioAPI.RegisterRoutes(mux, s.corsMiddleware)
	}

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	s.logger.Info("HTTP API server starting", logging.WithField("addr", addr))
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleGetItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	limit := 50
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Parse sources (comma-separated)
	var sources []string
	if s := query.Get("sources"); s != "" {
		sources = strings.Split(s, ",")
	}

	params := models.FilterParams{
		Limit:      limit,
		Offset:     offset,
		Sources:    sources,
		SourceType: query.Get("sourceType"),
		Query:      query.Get("q"),
		Sort:       query.Get("sort"),
		FromDate:   query.Get("fromDate"),
		ToDate:     query.Get("toDate"),
		Tag:        query.Get("tag"),
	}

	response := s.agg.GetItems(params)

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sources := s.agg.GetSources()
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"sources": sources,
		"count":   len(sources),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	if err := s.agg.Refresh(ctx); err != nil {
		s.logger.Error("Failed to refresh feed", logging.WithField("error", err.Error()))
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Feed refreshed successfully",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
