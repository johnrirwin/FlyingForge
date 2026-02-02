package httpapi

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/battery"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// BatteryAPI handles HTTP API requests for battery management
type BatteryAPI struct {
	batterySvc     *battery.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewBatteryAPI creates a new battery API handler
func NewBatteryAPI(batterySvc *battery.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *BatteryAPI {
	return &BatteryAPI{
		batterySvc:     batterySvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers battery routes on the given mux
func (api *BatteryAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Battery routes (require authentication)
	mux.HandleFunc("/api/batteries", corsMiddleware(api.authMiddleware.RequireAuth(api.handleBatteries)))
	mux.HandleFunc("/api/batteries/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleBatteryItem)))
}

// handleBatteries handles list and create operations
func (api *BatteryAPI) handleBatteries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listBatteries(w, r)
	case http.MethodPost:
		api.createBattery(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listBatteries returns all batteries for the authenticated user
func (api *BatteryAPI) listBatteries(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	query := r.URL.Query()

	// Handle sort parameters
	sort := query.Get("sort")
	if sort == "" {
		sortBy := query.Get("sort_by")
		sortOrder := query.Get("sort_order")
		if sortBy != "" {
			if sortOrder != "" {
				sort = fmt.Sprintf("%s_%s", sortBy, strings.ToLower(sortOrder))
			} else {
				sort = sortBy
			}
		}
	}

	params := models.BatteryListParams{
		Chemistry: models.BatteryChemistry(query.Get("chemistry")),
		Query:     query.Get("query"),
		Sort:      sort,
		SortOrder: query.Get("sort_order"),
	}

	if cells := query.Get("cells"); cells != "" {
		if c, err := strconv.Atoi(cells); err == nil {
			params.Cells = c
		}
	}
	if minCap := query.Get("min_capacity"); minCap != "" {
		if c, err := strconv.Atoi(minCap); err == nil {
			params.MinCapacity = c
		}
	}
	if maxCap := query.Get("max_capacity"); maxCap != "" {
		if c, err := strconv.Atoi(maxCap); err == nil {
			params.MaxCapacity = c
		}
	}
	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			params.Limit = l
		}
	}
	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			params.Offset = o
		}
	}

	response, err := api.batterySvc.List(r.Context(), userID, params)
	if err != nil {
		api.logger.Error("Battery list failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// createBattery creates a new battery
func (api *BatteryAPI) createBattery(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var params models.CreateBatteryParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	battery, err := api.batterySvc.Create(r.Context(), userID, params)
	if err != nil {
		api.logger.Error("Create battery failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusCreated, battery)
}

// handleBatteryItem handles single battery operations
func (api *BatteryAPI) handleBatteryItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/batteries/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Battery ID required", http.StatusBadRequest)
		return
	}

	batteryID := parts[0]

	// Check for sub-resources
	if len(parts) > 1 {
		switch parts[1] {
		case "logs":
			if len(parts) > 2 {
				// /api/batteries/{id}/logs/{logId}
				api.handleLogItem(w, r, batteryID, parts[2])
			} else {
				// /api/batteries/{id}/logs
				api.handleLogs(w, r, batteryID)
			}
			return
		case "label":
			api.handleLabel(w, r, batteryID)
			return
		case "details":
			api.getBatteryDetails(w, r, batteryID)
			return
		default:
			http.Error(w, "Unknown resource", http.StatusNotFound)
			return
		}
	}

	// Handle base battery CRUD
	switch r.Method {
	case http.MethodGet:
		api.getBattery(w, r, batteryID)
	case http.MethodPut, http.MethodPatch:
		api.updateBattery(w, r, batteryID)
	case http.MethodDelete:
		api.deleteBattery(w, r, batteryID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getBattery retrieves a single battery
func (api *BatteryAPI) getBattery(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	battery, err := api.batterySvc.Get(r.Context(), id, userID)
	if err != nil {
		api.logger.Error("Get battery failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if battery == nil {
		http.Error(w, "Battery not found", http.StatusNotFound)
		return
	}

	api.writeJSON(w, http.StatusOK, battery)
}

// getBatteryDetails retrieves full battery details including logs
func (api *BatteryAPI) getBatteryDetails(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())

	details, err := api.batterySvc.GetDetails(r.Context(), id, userID)
	if err != nil {
		api.logger.Error("Get battery details failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if details == nil {
		http.Error(w, "Battery not found", http.StatusNotFound)
		return
	}

	api.writeJSON(w, http.StatusOK, details)
}

// updateBattery updates a battery
func (api *BatteryAPI) updateBattery(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	var params models.UpdateBatteryParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	params.ID = id

	battery, err := api.batterySvc.Update(r.Context(), userID, params)
	if err != nil {
		api.logger.Error("Update battery failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, battery)
}

// deleteBattery deletes a battery
func (api *BatteryAPI) deleteBattery(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	if err := api.batterySvc.Delete(r.Context(), id, userID); err != nil {
		api.logger.Error("Delete battery failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleLogs handles battery log operations
func (api *BatteryAPI) handleLogs(w http.ResponseWriter, r *http.Request, batteryID string) {
	switch r.Method {
	case http.MethodGet:
		api.listLogs(w, r, batteryID)
	case http.MethodPost:
		api.createLog(w, r, batteryID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listLogs lists logs for a battery
func (api *BatteryAPI) listLogs(w http.ResponseWriter, r *http.Request, batteryID string) {
	userID := auth.GetUserID(r.Context())

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	response, err := api.batterySvc.ListLogs(r.Context(), batteryID, userID, limit)
	if err != nil {
		api.logger.Error("List battery logs failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// createLog creates a new battery log entry
func (api *BatteryAPI) createLog(w http.ResponseWriter, r *http.Request, batteryID string) {
	userID := auth.GetUserID(r.Context())

	var params models.CreateBatteryLogParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	params.BatteryID = batteryID

	log, err := api.batterySvc.CreateLog(r.Context(), userID, params)
	if err != nil {
		api.logger.Error("Create battery log failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusCreated, log)
}

// handleLogItem handles single log operations
func (api *BatteryAPI) handleLogItem(w http.ResponseWriter, r *http.Request, batteryID string, logID string) {
	switch r.Method {
	case http.MethodDelete:
		api.deleteLog(w, r, logID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// deleteLog deletes a battery log entry
func (api *BatteryAPI) deleteLog(w http.ResponseWriter, r *http.Request, logID string) {
	userID := auth.GetUserID(r.Context())

	if err := api.batterySvc.DeleteLog(r.Context(), logID, userID); err != nil {
		api.logger.Error("Delete battery log failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleLabel generates a printable label for a battery
func (api *BatteryAPI) handleLabel(w http.ResponseWriter, r *http.Request, batteryID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())

	battery, err := api.batterySvc.Get(r.Context(), batteryID, userID)
	if err != nil {
		api.logger.Error("Get battery for label failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if battery == nil {
		http.Error(w, "Battery not found", http.StatusNotFound)
		return
	}

	// Get label size from query param
	size := r.URL.Query().Get("size")
	if size != "small" {
		size = "standard"
	}

	// Generate HTML label
	html := api.generateLabelHTML(battery, size)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// generateLabelHTML generates printer-friendly HTML for a battery label
func (api *BatteryAPI) generateLabelHTML(b *models.Battery, size string) string {
	// QR code content - use battery ID so scanners get a self-contained code
	qrContent := b.ID

	// Determine label dimensions based on size
	var width, height, fontSize, qrSize string
	if size == "small" {
		width = "1.5in"
		height = "1.25in" // more height for bottom margin
		fontSize = "10pt"
		qrSize = "44"
	} else {
		width = "2.5in"
		height = "1.85in" // more height for bottom margin
		fontSize = "12pt"
		qrSize = "60"
	}

	// Chemistry display
	chemistryDisplay := string(b.Chemistry)
	switch b.Chemistry {
	case models.ChemistryLIPO:
		chemistryDisplay = "LiPo"
	case models.ChemistryLIPOHV:
		chemistryDisplay = "LiPo HV"
	case models.ChemistryLIION:
		chemistryDisplay = "Li-Ion"
	}

	// Format capacity
	capacityStr := fmt.Sprintf("%dmAh", b.CapacityMah)

	// Escape all user-provided content for HTML safety
	batteryCodeEscaped := html.EscapeString(b.BatteryCode)
	chemistryEscaped := html.EscapeString(chemistryDisplay)
	qrContentEscaped := html.EscapeString(qrContent)

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Battery Label - %s</title>
    <style>
        @media print {
            @page {
                size: auto;
                margin: 0;
            }
            body {
                margin: 0;
            }
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 0;
            background: white;
        }
		.label {
			width: %s;
			height: %s;
			padding: 12px 12px 28px 12px;
			box-sizing: border-box;
			border: 1px dashed #ccc;
			display: flex;
			flex-direction: column;
			justify-content: center;
			align-items: center;
		}
        .battery-code {
            font-size: 18pt;
            font-weight: bold;
            text-align: center;
            margin-bottom: 4px;
        }
        .specs {
            font-size: %s;
            text-align: center;
            margin-bottom: 4px;
        }
        .qr-section {
            display: flex;
            justify-content: center;
            align-items: center;
        }
		.qr-code {
			width: %spx;
			height: %spx;
			margin: 4px;
		}
        .print-btn {
            position: fixed;
            top: 10px;
            right: 10px;
            padding: 10px 20px;
            background: #0066cc;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
        }
        .print-btn:hover {
            background: #0055aa;
        }
        @media print {
            .print-btn {
                display: none;
            }
            .label {
                border: none;
            }
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>
</head>
<body>
    <button class="print-btn" onclick="window.print()">🖨️ Print Label</button>
    
    <div class="label">
        <div class="battery-code">%s</div>
        <div class="specs">%s • %dS • %s</div>
        <div class="qr-section">
            <div id="qrcode" class="qr-code"></div>
        </div>
    </div>

    <script>
        var qr = qrGenerator(0, 'M');
        qr.addData('%s');
        qr.make();
		document.getElementById('qrcode').innerHTML = qr.createSvgTag({
			cellSize: 3,
			margin: 4
		});
        
        function qrGenerator(typeNumber, errorCorrectionLevel) {
            return qrcode(typeNumber, errorCorrectionLevel);
        }
    </script>
</body>
</html>`,
		batteryCodeEscaped,
		width, height, fontSize,
		qrSize, qrSize,
		batteryCodeEscaped,
		chemistryEscaped, b.Cells, capacityStr,
		qrContentEscaped,
	)
	if size == "small" {
		htmlContent := fmt.Sprintf(`<!DOCTYPE html>
		<html>
		<head>
		    <meta charset="utf-8">
		    <title>Battery Label - %s</title>
		    <style>
		        @media print {
		            @page {
		                size: auto;
		                margin: 0;
		            }
		            body {
		                margin: 0;
		            }
		        }
		        body {
		            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		            margin: 0;
		            padding: 0;
		            background: white;
		        }
		        .label {
		            width: %s;
		            height: %s;
		            padding: 8px 8px 8px 12px;
		            box-sizing: border-box;
		            border: 1px dashed #ccc;
		            display: flex;
		            flex-direction: row;
		            justify-content: space-between;
		            align-items: center;
		        }
		        .label-left {
		            display: flex;
		            flex-direction: column;
		            justify-content: center;
		            align-items: flex-start;
		            flex: 1;
		            min-width: 0;
		        }
		        .battery-code {
		            font-size: 13pt;
		            font-weight: bold;
		            text-align: left;
		            margin-bottom: 2px;
		            white-space: nowrap;
		        }
		        .specs {
		            font-size: %s;
		            text-align: left;
		            margin-bottom: 0;
		            white-space: nowrap;
		        }
		        .qr-section {
		            display: flex;
		            justify-content: center;
		            align-items: center;
		            margin-left: 8px;
		        }
		        .qr-code {
		            width: %spx;
		            height: %spx;
		            margin: 0;
		        }
		        .print-btn {
		            position: fixed;
		            top: 10px;
		            right: 10px;
		            padding: 10px 20px;
		            background: #0066cc;
		            color: white;
		            border: none;
		            border-radius: 4px;
		            cursor: pointer;
		            font-size: 14px;
		        }
		        .print-btn:hover {
		            background: #0055aa;
		        }
		        @media print {
		            .print-btn {
		                display: none;
		            }
		            .label {
		                border: none;
		            }
		        }
		    </style>
		    <script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>
		</head>
		<body>
		    <button class="print-btn" onclick="window.print()">🖨️ Print Label</button>
		    <div class="label">
		        <div class="label-left">
		            <div class="battery-code">%s</div>
		            <div class="specs">%s • %dS • %s</div>
		        </div>
		        <div class="qr-section">
		            <div id="qrcode" class="qr-code"></div>
		        </div>
		    </div>
		    <script>
		        var qr = qrGenerator(0, 'M');
		        qr.addData('%s');
		        qr.make();
		        document.getElementById('qrcode').innerHTML = qr.createSvgTag({
		            cellSize: 2,
		            margin: 0
		        });
		        function qrGenerator(typeNumber, errorCorrectionLevel) {
		            return qrcode(typeNumber, errorCorrectionLevel);
		        }
		    </script>
		</body>
		</html>`,
			batteryCodeEscaped,
			width, height, fontSize,
			qrSize, qrSize,
			batteryCodeEscaped,
			chemistryEscaped, b.Cells, capacityStr,
			qrContentEscaped,
		)
		return htmlContent
	}
	return htmlContent
}

// writeJSON writes a JSON response
func (api *BatteryAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
