package sellers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/cache"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

// GetFPV is the adapter for GetFPV.com
type GetFPV struct {
	limiter *ratelimit.Limiter
	cache   cache.Cache
	client  *http.Client
}

// NewGetFPV creates a new GetFPV adapter
func NewGetFPV(limiter *ratelimit.Limiter, cache cache.Cache) *GetFPV {
	return &GetFPV{
		limiter: limiter,
		cache:   cache,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *GetFPV) ID() string {
	return "getfpv"
}

func (g *GetFPV) Name() string {
	return "GetFPV"
}

func (g *GetFPV) BaseURL() string {
	return "https://www.getfpv.com"
}

// categoryMapping maps our categories to GetFPV category slugs
var getfpvCategoryMapping = map[models.EquipmentCategory]string{
	models.CategoryFrames:      "frames",
	models.CategoryVTX:         "video-transmitters",
	models.CategoryFC:          "flight-controllers",
	models.CategoryESC:         "escs",
	models.CategoryStacks:      "stacks",
	models.CategoryMotors:      "motors",
	models.CategoryPropellers:  "propellers",
	models.CategoryReceivers:   "receivers",
	models.CategoryBatteries:   "batteries",
	models.CategoryCameras:     "fpv-cameras",
	models.CategoryAntennas:    "antennas",
	models.CategoryAccessories: "accessories",
}

func (g *GetFPV) Search(ctx context.Context, query string, category models.EquipmentCategory, limit int) ([]models.EquipmentItem, error) {
	g.limiter.Wait(g.BaseURL())

	// GetFPV uses Magento-style search
	searchURL := fmt.Sprintf("%s/catalogsearch/result/?q=%s",
		g.BaseURL(), url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	// For now, return demo data since parsing HTML would require additional libraries
	return g.getDemoProducts(category, limit), nil
}

func (g *GetFPV) GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) ([]models.EquipmentItem, error) {
	_, ok := getfpvCategoryMapping[category]
	if !ok {
		return nil, fmt.Errorf("unsupported category: %s", category)
	}

	// For now, return demo data
	return g.getDemoProducts(category, limit), nil
}

func (g *GetFPV) GetProduct(ctx context.Context, productID string) (*models.EquipmentItem, error) {
	// Strip prefix for future use
	_ = strings.TrimPrefix(productID, "gfpv-")

	g.limiter.Wait(g.BaseURL())

	// For now, return nil as we'd need to parse HTML
	return nil, fmt.Errorf("product not found")
}

func (g *GetFPV) SyncProducts(ctx context.Context) error {
	// For now, just return nil - full sync can be implemented later
	return nil
}

// getDemoProducts returns demo products for testing
func (g *GetFPV) getDemoProducts(category models.EquipmentCategory, limit int) []models.EquipmentItem {
	demos := map[models.EquipmentCategory][]models.EquipmentItem{
		models.CategoryFrames: {
			{ID: "gfpv-demo-1", Name: "Lumenier QAV-S JohnnyFPV 5\"", Price: 69.99, Currency: "USD", Manufacturer: "Lumenier", InStock: true, Category: models.CategoryFrames},
			{ID: "gfpv-demo-2", Name: "iFlight Nazgul5 V2 Frame", Price: 49.99, Currency: "USD", Manufacturer: "iFlight", InStock: true, Category: models.CategoryFrames},
		},
		models.CategoryMotors: {
			{ID: "gfpv-demo-3", Name: "iFlight XING2 2207 2755KV", Price: 19.99, Currency: "USD", Manufacturer: "iFlight", InStock: true, Category: models.CategoryMotors},
			{ID: "gfpv-demo-4", Name: "BrotherHobby Avenger 2806.5", Price: 29.99, Currency: "USD", Manufacturer: "BrotherHobby", InStock: true, Category: models.CategoryMotors},
		},
		models.CategoryFC: {
			{ID: "gfpv-demo-5", Name: "BetaFPV Toothpick F4 2-4S", Price: 29.99, Currency: "USD", Manufacturer: "BetaFPV", InStock: true, Category: models.CategoryFC},
			{ID: "gfpv-demo-6", Name: "Holybro Kakute H7 Mini", Price: 49.99, Currency: "USD", Manufacturer: "Holybro", InStock: true, Category: models.CategoryFC},
		},
		models.CategoryVTX: {
			{ID: "gfpv-demo-7", Name: "Rush Tank Solo 5.8GHz", Price: 39.99, Currency: "USD", Manufacturer: "Rush", InStock: true, Category: models.CategoryVTX},
			{ID: "gfpv-demo-8", Name: "TBS Unify Pro32 HV", Price: 49.99, Currency: "USD", Manufacturer: "TBS", InStock: true, Category: models.CategoryVTX},
		},
		models.CategoryCameras: {
			{ID: "gfpv-demo-9", Name: "Caddx Ratel 2", Price: 29.99, Currency: "USD", Manufacturer: "Caddx", InStock: true, Category: models.CategoryCameras},
			{ID: "gfpv-demo-10", Name: "RunCam Phoenix 2", Price: 34.99, Currency: "USD", Manufacturer: "RunCam", InStock: true, Category: models.CategoryCameras},
		},
	}

	items := demos[category]
	if items == nil {
		items = []models.EquipmentItem{
			{ID: "gfpv-demo-0", Name: "Sample GetFPV Product", Price: 24.99, Currency: "USD", Manufacturer: "Generic", InStock: true, Category: category},
		}
	}

	for i := range items {
		items[i].Seller = g.Name()
		items[i].SellerID = g.ID()
		items[i].ProductURL = fmt.Sprintf("%s/demo-%d", g.BaseURL(), i)
	}

	if len(items) > limit && limit > 0 {
		items = items[:limit]
	}

	return items
}
