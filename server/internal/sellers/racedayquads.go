package sellers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/cache"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

// RaceDayQuads is the adapter for RaceDayQuads.com
type RaceDayQuads struct {
	limiter *ratelimit.Limiter
	cache   cache.Cache
	client  *http.Client
}

// NewRaceDayQuads creates a new RaceDayQuads adapter
func NewRaceDayQuads(limiter *ratelimit.Limiter, cache cache.Cache) *RaceDayQuads {
	return &RaceDayQuads{
		limiter: limiter,
		cache:   cache,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *RaceDayQuads) ID() string {
	return "racedayquads"
}

func (r *RaceDayQuads) Name() string {
	return "RaceDayQuads"
}

func (r *RaceDayQuads) BaseURL() string {
	return "https://www.racedayquads.com"
}

// categoryMapping maps our categories to RDQ collection handles
var rdqCategoryMapping = map[models.EquipmentCategory]string{
	models.CategoryFrames:      "frames",
	models.CategoryVTX:         "video-transmitters",
	models.CategoryFC:          "flight-controllers",
	models.CategoryESC:         "escs",
	models.CategoryStacks:      "stacks",
	models.CategoryMotors:      "motors",
	models.CategoryPropellers:  "propellers",
	models.CategoryReceivers:   "receivers",
	models.CategoryBatteries:   "batteries",
	models.CategoryCameras:     "cameras",
	models.CategoryAntennas:    "antennas",
	models.CategoryAccessories: "accessories",
}

func (r *RaceDayQuads) Search(ctx context.Context, query string, category models.EquipmentCategory, limit int) ([]models.EquipmentItem, error) {
	r.limiter.Wait(r.BaseURL())

	// Use Shopify search endpoint
	searchURL := fmt.Sprintf("%s/search/suggest.json?q=%s&resources[type]=product&resources[limit]=%d",
		r.BaseURL(), url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Return demo data for now
		return r.getDemoProducts(category, limit), nil
	}

	var result struct {
		Resources struct {
			Results struct {
				Products []struct {
					ID        int64  `json:"id"`
					Title     string `json:"title"`
					Handle    string `json:"handle"`
					Image     string `json:"image"`
					Price     string `json:"price"`
					PriceMin  string `json:"price_min"`
					Available bool   `json:"available"`
					Vendor    string `json:"vendor"`
				} `json:"products"`
			} `json:"results"`
		} `json:"resources"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return r.getDemoProducts(category, limit), nil
	}

	items := make([]models.EquipmentItem, 0, len(result.Resources.Results.Products))
	for _, p := range result.Resources.Results.Products {
		price := parsePrice(p.Price)
		if price == 0 {
			price = parsePrice(p.PriceMin)
		}

		item := models.EquipmentItem{
			ID:           fmt.Sprintf("rdq-%d", p.ID),
			Name:         p.Title,
			Seller:       r.Name(),
			SellerID:     r.ID(),
			Price:        price,
			Currency:     "USD",
			ProductURL:   fmt.Sprintf("%s/products/%s", r.BaseURL(), p.Handle),
			ImageURL:     p.Image,
			InStock:      p.Available,
			Manufacturer: p.Vendor,
			Category:     category,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return r.getDemoProducts(category, limit), nil
	}

	return items, nil
}

func (r *RaceDayQuads) GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) ([]models.EquipmentItem, error) {
	collectionHandle, ok := rdqCategoryMapping[category]
	if !ok {
		return nil, fmt.Errorf("unsupported category: %s", category)
	}

	r.limiter.Wait(r.BaseURL())

	// Use Shopify collections endpoint
	collectionURL := fmt.Sprintf("%s/collections/%s/products.json?limit=%d",
		r.BaseURL(), collectionHandle, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", collectionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return r.getDemoProducts(category, limit), nil
	}

	var result struct {
		Products []struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			Handle    string `json:"handle"`
			Vendor    string `json:"vendor"`
			Available bool   `json:"available"`
			Images    []struct {
				Src string `json:"src"`
			} `json:"images"`
			Variants []struct {
				Price string `json:"price"`
			} `json:"variants"`
		} `json:"products"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return r.getDemoProducts(category, limit), nil
	}

	items := make([]models.EquipmentItem, 0, len(result.Products))
	for _, p := range result.Products {
		var price float64
		if len(p.Variants) > 0 {
			price = parsePrice(p.Variants[0].Price)
		}

		var imageURL string
		if len(p.Images) > 0 {
			imageURL = p.Images[0].Src
		}

		item := models.EquipmentItem{
			ID:           fmt.Sprintf("rdq-%d", p.ID),
			Name:         p.Title,
			Seller:       r.Name(),
			SellerID:     r.ID(),
			Price:        price,
			Currency:     "USD",
			ProductURL:   fmt.Sprintf("%s/products/%s", r.BaseURL(), p.Handle),
			ImageURL:     imageURL,
			InStock:      p.Available,
			Manufacturer: p.Vendor,
			Category:     category,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return r.getDemoProducts(category, limit), nil
	}

	return items, nil
}

func (r *RaceDayQuads) GetProduct(ctx context.Context, productID string) (*models.EquipmentItem, error) {
	// Strip prefix
	productID = strings.TrimPrefix(productID, "rdq-")

	r.limiter.Wait(r.BaseURL())

	productURL := fmt.Sprintf("%s/products/%s.json", r.BaseURL(), productID)

	req, err := http.NewRequestWithContext(ctx, "GET", productURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product not found")
	}

	var result struct {
		Product struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			Handle    string `json:"handle"`
			Vendor    string `json:"vendor"`
			BodyHTML  string `json:"body_html"`
			Available bool   `json:"available"`
			Images    []struct {
				Src string `json:"src"`
			} `json:"images"`
			Variants []struct {
				Price string `json:"price"`
			} `json:"variants"`
		} `json:"product"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode product: %w", err)
	}

	p := result.Product
	var price float64
	if len(p.Variants) > 0 {
		price = parsePrice(p.Variants[0].Price)
	}

	var imageURL string
	if len(p.Images) > 0 {
		imageURL = p.Images[0].Src
	}

	return &models.EquipmentItem{
		ID:           fmt.Sprintf("rdq-%d", p.ID),
		Name:         p.Title,
		Seller:       r.Name(),
		SellerID:     r.ID(),
		Price:        price,
		Currency:     "USD",
		ProductURL:   fmt.Sprintf("%s/products/%s", r.BaseURL(), p.Handle),
		ImageURL:     imageURL,
		InStock:      p.Available,
		Manufacturer: p.Vendor,
		Description:  p.BodyHTML,
	}, nil
}

func (r *RaceDayQuads) SyncProducts(ctx context.Context) error {
	// For now, just return nil - full sync can be implemented later
	return nil
}

// getDemoProducts returns demo products for testing
func (r *RaceDayQuads) getDemoProducts(category models.EquipmentCategory, limit int) []models.EquipmentItem {
	demos := map[models.EquipmentCategory][]models.EquipmentItem{
		models.CategoryFrames: {
			{ID: "rdq-demo-1", Name: "ImpulseRC Apex 5\" Frame", Price: 89.99, Currency: "USD", Manufacturer: "ImpulseRC", InStock: true, Category: models.CategoryFrames},
			{ID: "rdq-demo-2", Name: "TBS Source One V5", Price: 34.99, Currency: "USD", Manufacturer: "TBS", InStock: true, Category: models.CategoryFrames},
			{ID: "rdq-demo-3", Name: "Armattan Badger 6\" Frame", Price: 149.99, Currency: "USD", Manufacturer: "Armattan", InStock: true, Category: models.CategoryFrames},
		},
		models.CategoryMotors: {
			{ID: "rdq-demo-4", Name: "EMAX ECO II 2306 2400KV", Price: 16.99, Currency: "USD", Manufacturer: "EMAX", InStock: true, Category: models.CategoryMotors},
			{ID: "rdq-demo-5", Name: "T-Motor F60 Pro IV 2207", Price: 24.99, Currency: "USD", Manufacturer: "T-Motor", InStock: true, Category: models.CategoryMotors},
		},
		models.CategoryFC: {
			{ID: "rdq-demo-6", Name: "SpeedyBee F405 V4", Price: 39.99, Currency: "USD", Manufacturer: "SpeedyBee", InStock: true, Category: models.CategoryFC},
			{ID: "rdq-demo-7", Name: "Diatone MAMBA F722", Price: 54.99, Currency: "USD", Manufacturer: "Diatone", InStock: true, Category: models.CategoryFC},
		},
	}

	items := demos[category]
	if items == nil {
		items = []models.EquipmentItem{
			{ID: "rdq-demo-0", Name: "Sample Product", Price: 19.99, Currency: "USD", Manufacturer: "Generic", InStock: true, Category: category},
		}
	}

	for i := range items {
		items[i].Seller = r.Name()
		items[i].SellerID = r.ID()
		items[i].ProductURL = fmt.Sprintf("%s/products/demo-%d", r.BaseURL(), i)
	}

	if len(items) > limit && limit > 0 {
		items = items[:limit]
	}

	return items
}

func parsePrice(priceStr string) float64 {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.TrimPrefix(priceStr, "$")
	priceStr = strings.ReplaceAll(priceStr, ",", "")

	var price float64
	_, _ = fmt.Sscanf(priceStr, "%f", &price)
	return price
}
