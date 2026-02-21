package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

type mockInventoryManager struct {
	added       []models.AddInventoryParams
	addItem     *models.InventoryItem
	addErr      error
	getItem     *models.InventoryItem
	getItemErr  error
	updateItem  *models.InventoryItem
	updateErr   error
	updateCalls []models.UpdateInventoryParams
}

func (m *mockInventoryManager) AddItem(ctx context.Context, userID string, params models.AddInventoryParams) (*models.InventoryItem, error) {
	m.added = append(m.added, params)
	if m.addErr != nil {
		return nil, m.addErr
	}
	if m.addItem != nil {
		return m.addItem, nil
	}
	return &models.InventoryItem{
		ID:       "inv-1",
		UserID:   userID,
		Name:     params.Name,
		Category: params.Category,
		Quantity: params.Quantity,
	}, nil
}

func (m *mockInventoryManager) AddFromEquipment(ctx context.Context, userID string, equipment models.EquipmentItem, quantity int, notes string) (*models.InventoryItem, error) {
	return nil, nil
}

func (m *mockInventoryManager) GetItem(ctx context.Context, id string, userID string) (*models.InventoryItem, error) {
	if m.getItemErr != nil {
		return nil, m.getItemErr
	}
	return m.getItem, nil
}

func (m *mockInventoryManager) GetInventory(ctx context.Context, userID string, params models.InventoryFilterParams) (*models.InventoryResponse, error) {
	return nil, nil
}

func (m *mockInventoryManager) UpdateItem(ctx context.Context, userID string, params models.UpdateInventoryParams) (*models.InventoryItem, error) {
	m.updateCalls = append(m.updateCalls, params)
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.updateItem != nil {
		return m.updateItem, nil
	}
	return &models.InventoryItem{
		ID:       params.ID,
		UserID:   userID,
		Category: models.CategoryAccessories,
		Quantity: 1,
	}, nil
}

func (m *mockInventoryManager) RemoveItem(ctx context.Context, id string, userID string) error {
	return nil
}

func (m *mockInventoryManager) GetSummary(ctx context.Context, userID string) (*models.InventorySummary, error) {
	return nil, nil
}

type mockBatteryCreator struct {
	createCalls []models.CreateBatteryParams
	createErr   error
	failOnCall  int
}

func (m *mockBatteryCreator) Create(ctx context.Context, userID string, params models.CreateBatteryParams) (*models.Battery, error) {
	m.createCalls = append(m.createCalls, params)
	if m.failOnCall > 0 && len(m.createCalls) >= m.failOnCall {
		return nil, &models.ValidationError{Field: "battery", Message: "simulated battery create failure"}
	}
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &models.Battery{ID: "bat-1"}, nil
}

func TestAddInventoryItem_AutoCreatesTrackedBatteries(t *testing.T) {
	inventory := &mockInventoryManager{
		addItem: &models.InventoryItem{
			ID:       "inv-battery-1",
			Category: models.CategoryBatteries,
			Quantity: 3,
		},
	}
	battery := &mockBatteryCreator{}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	requestBody := `{
		"name":"Tattu R-Line 1300mAh 6S 120C XT60",
		"category":"batteries",
		"manufacturer":"Tattu",
		"quantity":3
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/inventory", strings.NewReader(requestBody))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.addInventoryItem(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	if len(inventory.added) != 1 {
		t.Fatalf("inventory AddItem calls = %d, want 1", len(inventory.added))
	}
	if inventory.added[0].Category != models.CategoryBatteries {
		t.Fatalf("inventory category = %q, want %q", inventory.added[0].Category, models.CategoryBatteries)
	}

	if len(battery.createCalls) != 3 {
		t.Fatalf("battery Create calls = %d, want 3", len(battery.createCalls))
	}

	for i, call := range battery.createCalls {
		if call.Chemistry != models.ChemistryLIPO {
			t.Fatalf("call[%d].Chemistry = %q, want %q", i, call.Chemistry, models.ChemistryLIPO)
		}
		if call.Cells != 6 {
			t.Fatalf("call[%d].Cells = %d, want 6", i, call.Cells)
		}
		if call.CapacityMah != 1300 {
			t.Fatalf("call[%d].CapacityMah = %d, want 1300", i, call.CapacityMah)
		}
		if call.CRating == nil || *call.CRating != 120 {
			t.Fatalf("call[%d].CRating = %v, want 120", i, call.CRating)
		}
		if call.Connector != "XT60" {
			t.Fatalf("call[%d].Connector = %q, want XT60", i, call.Connector)
		}
	}
}

func TestAddInventoryItem_NonBatteryCategorySkipsBatteryCreation(t *testing.T) {
	inventory := &mockInventoryManager{}
	battery := &mockBatteryCreator{}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/inventory", strings.NewReader(`{
		"name":"2207 Motor",
		"category":"motors",
		"quantity":2
	}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.addInventoryItem(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if len(battery.createCalls) != 0 {
		t.Fatalf("battery Create calls = %d, want 0", len(battery.createCalls))
	}
}

func TestInferBatteryCreateParams(t *testing.T) {
	params := models.AddInventoryParams{
		Name:         "Mystery Pack",
		Category:     models.CategoryBatteries,
		Manufacturer: "Unknown",
		Specs: json.RawMessage(`{
			"chemistry":"LIION",
			"cells":"5S",
			"capacity_mah":"3000",
			"c_rating":"15C",
			"connector":"XT30",
			"weight_grams":"420g"
		}`),
	}

	got := inferBatteryCreateParams(params)

	if got.Chemistry != models.ChemistryLIION {
		t.Fatalf("Chemistry = %q, want %q", got.Chemistry, models.ChemistryLIION)
	}
	if got.Cells != 5 {
		t.Fatalf("Cells = %d, want 5", got.Cells)
	}
	if got.CapacityMah != 3000 {
		t.Fatalf("CapacityMah = %d, want 3000", got.CapacityMah)
	}
	if got.CRating == nil || *got.CRating != 15 {
		t.Fatalf("CRating = %v, want 15", got.CRating)
	}
	if got.Connector != "XT30" {
		t.Fatalf("Connector = %q, want XT30", got.Connector)
	}
	if got.WeightGrams == nil || *got.WeightGrams != 420 {
		t.Fatalf("WeightGrams = %v, want 420", got.WeightGrams)
	}
}

func TestInferBatteryCreateParams_Defaults(t *testing.T) {
	got := inferBatteryCreateParams(models.AddInventoryParams{
		Name:     "Unknown Battery Pack",
		Category: models.CategoryBatteries,
	})

	if got.Chemistry != models.ChemistryLIPO {
		t.Fatalf("Chemistry = %q, want %q", got.Chemistry, models.ChemistryLIPO)
	}
	if got.Cells != defaultBatteryCells {
		t.Fatalf("Cells = %d, want %d", got.Cells, defaultBatteryCells)
	}
	if got.CapacityMah != defaultBatteryCapacityMah {
		t.Fatalf("CapacityMah = %d, want %d", got.CapacityMah, defaultBatteryCapacityMah)
	}
}

func TestAddInventoryItem_PartialBatteryCreationIncludesWarning(t *testing.T) {
	inventory := &mockInventoryManager{
		addItem: &models.InventoryItem{
			ID:       "inv-battery-partial",
			Category: models.CategoryBatteries,
			Quantity: 3,
		},
	}
	battery := &mockBatteryCreator{
		failOnCall: 2,
	}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/inventory", strings.NewReader(`{
		"name":"Tattu 1400mAh 6S",
		"category":"batteries",
		"quantity":3
	}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.addInventoryItem(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if gotID, _ := response["id"].(string); gotID != "inv-battery-partial" {
		t.Fatalf("response id = %q, want inv-battery-partial", gotID)
	}

	warning, ok := response["warning"].(string)
	if !ok || warning == "" {
		t.Fatalf("expected warning in response, got: %#v", response["warning"])
	}
	if !strings.Contains(warning, "1 of 3") {
		t.Fatalf("warning = %q, want partial creation detail", warning)
	}
}

func TestBatteryTrackingTimeoutScalesAndCaps(t *testing.T) {
	small := batteryTrackingTimeout(1)
	large := batteryTrackingTimeout(30)
	if large <= small {
		t.Fatalf("batteryTrackingTimeout(30) = %v, want > %v", large, small)
	}

	capped := batteryTrackingTimeout(500)
	if capped != batteryTrackingMaxTimeout {
		t.Fatalf("batteryTrackingTimeout(500) = %v, want %v", capped, batteryTrackingMaxTimeout)
	}
}

func TestUpdateInventoryItem_IncreasingBatteryQuantityCreatesDeltaTrackers(t *testing.T) {
	inventory := &mockInventoryManager{
		getItem: &models.InventoryItem{
			ID:       "inv-battery-1",
			Name:     "Tattu 1300mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 1,
		},
		updateItem: &models.InventoryItem{
			ID:       "inv-battery-1",
			Name:     "Tattu 1300mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 6,
		},
	}
	battery := &mockBatteryCreator{}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/inventory/inv-battery-1", strings.NewReader(`{"quantity":6}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.updateInventoryItem(w, req, "inv-battery-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(inventory.updateCalls) != 1 {
		t.Fatalf("update calls = %d, want 1", len(inventory.updateCalls))
	}
	if len(battery.createCalls) != 5 {
		t.Fatalf("battery Create calls = %d, want 5", len(battery.createCalls))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if _, hasWarning := response["warning"]; hasWarning {
		t.Fatalf("did not expect warning, got: %v", response["warning"])
	}
}

func TestUpdateInventoryItem_PartialDeltaBatteryCreationIncludesWarning(t *testing.T) {
	inventory := &mockInventoryManager{
		getItem: &models.InventoryItem{
			ID:       "inv-battery-2",
			Name:     "Tattu 1400mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 1,
		},
		updateItem: &models.InventoryItem{
			ID:       "inv-battery-2",
			Name:     "Tattu 1400mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 6,
		},
	}
	battery := &mockBatteryCreator{
		failOnCall: 3,
	}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/inventory/inv-battery-2", strings.NewReader(`{"quantity":6}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.updateInventoryItem(w, req, "inv-battery-2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(battery.createCalls) != 3 {
		t.Fatalf("battery Create calls = %d, want 3", len(battery.createCalls))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	warning, ok := response["warning"].(string)
	if !ok || warning == "" {
		t.Fatalf("expected warning in response, got: %#v", response["warning"])
	}
	if !strings.Contains(warning, "2 of 5") {
		t.Fatalf("warning = %q, want partial delta detail", warning)
	}
}

func TestUpdateInventoryItem_CategoryChangedToBatteriesCreatesTrackers(t *testing.T) {
	inventory := &mockInventoryManager{
		getItem: &models.InventoryItem{
			ID:       "inv-convert-1",
			Name:     "Tattu 1200mAh 6S",
			Category: models.CategoryAccessories,
			Quantity: 3,
		},
		updateItem: &models.InventoryItem{
			ID:       "inv-convert-1",
			Name:     "Tattu 1200mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 3,
		},
	}
	battery := &mockBatteryCreator{}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/inventory/inv-convert-1", strings.NewReader(`{"category":"batteries"}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.updateInventoryItem(w, req, "inv-convert-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(battery.createCalls) != 3 {
		t.Fatalf("battery Create calls = %d, want 3", len(battery.createCalls))
	}
}

func TestUpdateInventoryItem_DecreasingBatteryQuantityCreatesNoTrackers(t *testing.T) {
	inventory := &mockInventoryManager{
		getItem: &models.InventoryItem{
			ID:       "inv-battery-dec",
			Name:     "Tattu 1200mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 5,
		},
		updateItem: &models.InventoryItem{
			ID:       "inv-battery-dec",
			Name:     "Tattu 1200mAh 6S",
			Category: models.CategoryBatteries,
			Quantity: 3,
		},
	}
	battery := &mockBatteryCreator{}

	api := &EquipmentAPI{
		inventorySvc: inventory,
		batterySvc:   battery,
		logger:       logging.New(logging.LevelError),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/inventory/inv-battery-dec", strings.NewReader(`{"quantity":3}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
	w := httptest.NewRecorder()

	api.updateInventoryItem(w, req, "inv-battery-dec")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(battery.createCalls) != 0 {
		t.Fatalf("battery Create calls = %d, want 0", len(battery.createCalls))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if _, hasWarning := response["warning"]; hasWarning {
		t.Fatalf("did not expect warning, got: %v", response["warning"])
	}
}
