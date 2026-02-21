package models

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// GearType represents the type of gear in the catalog
type GearType string

const (
	GearTypeMotor    GearType = "motor"
	GearTypeESC      GearType = "esc"
	GearTypeFC       GearType = "fc"
	GearTypeAIO      GearType = "aio"
	GearTypeStack    GearType = "stack"
	GearTypeFrame    GearType = "frame"
	GearTypeVTX      GearType = "vtx"
	GearTypeReceiver GearType = "receiver"
	GearTypeAntenna  GearType = "antenna"
	GearTypeGPS      GearType = "gps"
	GearTypeBattery  GearType = "battery"
	GearTypeProp     GearType = "prop"
	GearTypeRadio    GearType = "radio"
	GearTypeCamera   GearType = "camera"
	GearTypeOther    GearType = "other"
)

// AllGearTypes returns all valid gear types
func AllGearTypes() []GearType {
	return []GearType{
		GearTypeMotor,
		GearTypeESC,
		GearTypeFC,
		GearTypeAIO,
		GearTypeStack,
		GearTypeFrame,
		GearTypeVTX,
		GearTypeReceiver,
		GearTypeAntenna,
		GearTypeGPS,
		GearTypeBattery,
		GearTypeProp,
		GearTypeRadio,
		GearTypeCamera,
		GearTypeOther,
	}
}

// GearTypeFromEquipmentCategory converts an EquipmentCategory to a GearType
func GearTypeFromEquipmentCategory(cat EquipmentCategory) GearType {
	switch cat {
	case CategoryMotors:
		return GearTypeMotor
	case CategoryESC:
		return GearTypeESC
	case CategoryFC:
		return GearTypeFC
	case CategoryAIO:
		return GearTypeAIO
	case CategoryStacks:
		return GearTypeStack
	case CategoryFrames:
		return GearTypeFrame
	case CategoryVTX:
		return GearTypeVTX
	case CategoryReceivers:
		return GearTypeReceiver
	case CategoryAntennas:
		return GearTypeAntenna
	case CategoryGPS:
		return GearTypeGPS
	case CategoryBatteries:
		return GearTypeBattery
	case CategoryPropellers:
		return GearTypeProp
	case CategoryCameras:
		return GearTypeCamera
	default:
		return GearTypeOther
	}
}

// ToEquipmentCategory converts a GearType back to EquipmentCategory
func (gt GearType) ToEquipmentCategory() EquipmentCategory {
	switch gt {
	case GearTypeMotor:
		return CategoryMotors
	case GearTypeESC:
		return CategoryESC
	case GearTypeFC:
		return CategoryFC
	case GearTypeAIO:
		return CategoryAIO
	case GearTypeStack:
		return CategoryStacks
	case GearTypeFrame:
		return CategoryFrames
	case GearTypeVTX:
		return CategoryVTX
	case GearTypeReceiver:
		return CategoryReceivers
	case GearTypeAntenna:
		return CategoryAntennas
	case GearTypeGPS:
		return CategoryGPS
	case GearTypeBattery:
		return CategoryBatteries
	case GearTypeProp:
		return CategoryPropellers
	case GearTypeRadio:
		return CategoryAccessories
	case GearTypeCamera:
		return CategoryCameras
	default:
		return CategoryAccessories
	}
}

// CatalogItemStatus represents the moderation status of a catalog item
type CatalogItemStatus string

const (
	CatalogStatusPublished CatalogItemStatus = "published"
	CatalogStatusPending   CatalogItemStatus = "pending"
	CatalogStatusRemoved   CatalogItemStatus = "removed"

	// Backward-compatible aliases for legacy values still present in old data/clients.
	CatalogStatusActive   CatalogItemStatus = CatalogStatusPublished
	CatalogStatusFlagged  CatalogItemStatus = CatalogStatusRemoved
	CatalogStatusRejected CatalogItemStatus = CatalogStatusRemoved
)

// NormalizeCatalogStatus maps legacy status values to current canonical values.
func NormalizeCatalogStatus(status CatalogItemStatus) CatalogItemStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "active", "published":
		return CatalogStatusPublished
	case "pending":
		return CatalogStatusPending
	case "flagged", "rejected", "removed":
		return CatalogStatusRemoved
	default:
		return status
	}
}

// IsValidCatalogStatus reports whether status is one of the supported catalog states.
func IsValidCatalogStatus(status CatalogItemStatus) bool {
	switch NormalizeCatalogStatus(status) {
	case CatalogStatusPublished, CatalogStatusPending, CatalogStatusRemoved:
		return true
	default:
		return false
	}
}

// ImageStatus represents the curation status of a gear item's image
type ImageStatus string

const (
	ImageStatusMissing  ImageStatus = "missing"
	ImageStatusScanned  ImageStatus = "scanned"
	ImageStatusApproved ImageStatus = "approved"
	// ImageStatusRecentlyCurated is a special filter value (not stored in DB)
	// Used by admin to find items curated within last 24 hours
	ImageStatusRecentlyCurated ImageStatus = "recently-curated"
	// ImageStatusAll is a special filter value (not stored in DB)
	// Used by admin to fetch all catalog records without curation-status filtering
	ImageStatusAll ImageStatus = "all"
)

// CatalogItemSource represents how the item was added
type CatalogItemSource string

const (
	CatalogSourceUserSubmitted CatalogItemSource = "user-submitted"
	CatalogSourceAdmin         CatalogItemSource = "admin"
	CatalogSourceImport        CatalogItemSource = "import"
	CatalogSourceMigration     CatalogItemSource = "migration"
)

// GearCatalogItem represents a canonical gear item in the shared catalog
type GearCatalogItem struct {
	ID                string            `json:"id"`
	GearType          GearType          `json:"gearType"`
	Brand             string            `json:"brand"`
	Model             string            `json:"model"`
	Variant           string            `json:"variant,omitempty"`
	Specs             json.RawMessage   `json:"specs,omitempty"`
	BestFor           []string          `json:"bestFor,omitempty"` // Drone types: freestyle, long-range, cinematic, etc.
	MSRP              *float64          `json:"msrp,omitempty"`    // Manufacturer suggested retail price
	Source            CatalogItemSource `json:"source"`
	CreatedByUserID   string            `json:"createdByUserId,omitempty"`
	Status            CatalogItemStatus `json:"status"`
	CanonicalKey      string            `json:"canonicalKey"`
	ImageURL          string            `json:"imageUrl,omitempty"`
	ImageSourceDomain string            `json:"imageSourceDomain,omitempty"`
	ShoppingLinks     []string          `json:"shoppingLinks,omitempty"`
	HasStoredImage    bool              `json:"hasStoredImage"`
	Description       string            `json:"description,omitempty"`
	UsageCount        int               `json:"usageCount"` // How many users have this in inventory
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`

	// Image curation fields
	ImageStatus          ImageStatus `json:"imageStatus"`
	ImageCuratedByUserID string      `json:"imageCuratedByUserId,omitempty"`
	ImageCuratedAt       *time.Time  `json:"imageCuratedAt,omitempty"`

	// Description curation fields
	DescriptionStatus          ImageStatus `json:"descriptionStatus"`
	DescriptionCuratedByUserID string      `json:"descriptionCuratedByUserId,omitempty"`
	DescriptionCuratedAt       *time.Time  `json:"descriptionCuratedAt,omitempty"`
}

// DisplayName returns a formatted display name for the catalog item
func (g *GearCatalogItem) DisplayName() string {
	name := g.Brand + " " + g.Model
	if g.Variant != "" {
		name += " " + g.Variant
	}
	return strings.TrimSpace(name)
}

// CreateGearCatalogParams represents the parameters for creating a catalog item
// Note: imageUrl is NOT included - images are added by admin only
type CreateGearCatalogParams struct {
	GearType    GearType        `json:"gearType"`
	Brand       string          `json:"brand"`
	Model       string          `json:"model"`
	Variant     string          `json:"variant,omitempty"`
	Specs       json.RawMessage `json:"specs,omitempty"`
	BestFor     []string        `json:"bestFor,omitempty"` // Drone types this gear is best suited for
	MSRP        *float64        `json:"msrp,omitempty"`    // Manufacturer suggested retail price
	Description string          `json:"description,omitempty"`
}

// AdminUpdateGearCatalogParams represents admin-only update parameters
type AdminUpdateGearCatalogParams struct {
	GearType          *GearType          `json:"gearType,omitempty"`
	Brand             *string            `json:"brand,omitempty"`
	Model             *string            `json:"model,omitempty"`
	Variant           *string            `json:"variant,omitempty"`
	Specs             json.RawMessage    `json:"specs,omitempty"`
	Description       *string            `json:"description,omitempty"`
	ImageURL          *string            `json:"imageUrl,omitempty"` // Optional external override; falls back to stored image asset when empty
	ImageSourceDomain *string            `json:"imageSourceDomain,omitempty"`
	ShoppingLinks     []string           `json:"shoppingLinks,omitempty"`
	MSRP              *float64           `json:"msrp,omitempty"`
	ClearMSRP         bool               `json:"clearMsrp,omitempty"` // Explicitly clear MSRP when true
	ImageStatus       *ImageStatus       `json:"imageStatus,omitempty"`
	BestFor           []string           `json:"bestFor,omitempty"` // Drone types this gear is best suited for
	Status            *CatalogItemStatus `json:"status,omitempty"`
}

// ExtractImageSourceDomain extracts a normalized host domain from an image URL.
// Example: https://cdn-v2.getfpv.com/path.jpg -> getfpv.com
func ExtractImageSourceDomain(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}

	return host
}

// AdminGearSearchParams represents admin search parameters with curation filters
type AdminGearSearchParams struct {
	Query       string            `json:"query,omitempty"`
	GearType    GearType          `json:"gearType,omitempty"`
	Brand       string            `json:"brand,omitempty"`
	Status      CatalogItemStatus `json:"status,omitempty"`      // Filter by overall catalog status
	ImageStatus ImageStatus       `json:"imageStatus,omitempty"` // Filter by image status
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
}

// GearCatalogSearchParams represents search parameters for the catalog
type GearCatalogSearchParams struct {
	Query    string            `json:"query,omitempty"`
	GearType GearType          `json:"gearType,omitempty"`
	Brand    string            `json:"brand,omitempty"`
	Status   CatalogItemStatus `json:"status,omitempty"`
	Limit    int               `json:"limit,omitempty"`
	Offset   int               `json:"offset,omitempty"`
}

// GearCatalogSearchResponse represents the response from a catalog search
type GearCatalogSearchResponse struct {
	Items      []GearCatalogItem `json:"items"`
	TotalCount int               `json:"totalCount"`
	Query      string            `json:"query,omitempty"`
}

// GearCatalogCreateResponse represents the response when creating/finding a catalog item
type GearCatalogCreateResponse struct {
	Item     *GearCatalogItem `json:"item"`
	Existing bool             `json:"existing"` // True if we found an existing match instead of creating new
}

// NearMatch represents a potential duplicate found during catalog creation
type NearMatch struct {
	Item       GearCatalogItem `json:"item"`
	Similarity float64         `json:"similarity"`
}

// NearMatchResponse represents the response when near matches are found
type NearMatchResponse struct {
	Matches []NearMatch `json:"matches"`
}

// BuildCanonicalKey creates a normalized key for deduplication
// Format: gear_type|brand|model|variant (all lowercase, normalized)
func BuildCanonicalKey(gearType GearType, brand, model, variant string) string {
	parts := []string{
		string(gearType),
		normalizeString(brand),
		normalizeString(model),
	}
	// Trim variant before checking - ensures " " and "" are treated the same
	variant = strings.TrimSpace(variant)
	if variant != "" {
		parts = append(parts, normalizeString(variant))
	}
	return strings.Join(parts, "|")
}

// normalizeString normalizes a string for canonical key generation
func normalizeString(s string) string {
	// 1. Normalize unicode (NFC form)
	s = norm.NFC.String(s)

	// 2. Convert to lowercase
	s = strings.ToLower(s)

	// 3. Replace punctuation with spaces
	punctuationRegex := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	s = punctuationRegex.ReplaceAllString(s, " ")

	// 4. Remove accents/diacritics
	s = removeDiacritics(s)

	// 5. Collapse multiple spaces into single space
	spaceRegex := regexp.MustCompile(`\s+`)
	s = spaceRegex.ReplaceAllString(s, " ")

	// 6. Trim whitespace
	s = strings.TrimSpace(s)

	return s
}

// removeDiacritics removes diacritical marks from a string
func removeDiacritics(s string) string {
	t := norm.NFD.String(s)
	var result strings.Builder
	for _, r := range t {
		if unicode.Is(unicode.Mn, r) {
			continue // Skip combining marks
		}
		result.WriteRune(r)
	}
	return result.String()
}

// ExtractBrandModelFromName attempts to extract brand and model from a combined name
// This is useful for migrating existing inventory items
func ExtractBrandModelFromName(name, manufacturer string) (brand, model, variant string) {
	// If manufacturer is provided, use it as brand
	if manufacturer != "" {
		brand = manufacturer
		// The remaining name is the model
		// Try to extract variant if present
		name = strings.TrimSpace(name)
		// Remove the brand prefix if it exists in the name
		nameLower := strings.ToLower(name)
		brandLower := strings.ToLower(manufacturer)
		if strings.HasPrefix(nameLower, brandLower) {
			name = strings.TrimSpace(name[len(manufacturer):])
		}
		model, variant = extractVariant(name)
		return
	}

	// Try to split name into brand + model
	// Common patterns: "TMotor F60 Pro IV 1950KV", "BetaFPV 1102 18000KV"
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", "", ""
	}

	// First part is usually the brand
	brand = parts[0]
	if len(parts) > 1 {
		remaining := strings.Join(parts[1:], " ")
		model, variant = extractVariant(remaining)
	}

	return brand, model, variant
}

// extractVariant tries to identify a variant suffix from a model name
// Common patterns: "V2", "Pro", "LR", "HV", "2024", version numbers
func extractVariant(s string) (model, variant string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}

	// Common variant patterns
	variantPatterns := []string{
		`\s+(V\d+)$`,                   // V1, V2, V3
		`\s+(Pro|Lite|Mini|Max|Plus)$`, // Edition names
		`\s+(LR|HV|LV)$`,               // Voltage/range variants
		`\s+(\d{4})$`,                  // Year versions
		`\s+(Mark\s*\d+|MK\d+)$`,       // Mark versions
		`\s+(Rev\s*[A-Z0-9]+)$`,        // Revision
		`\s+(\d+KV|\d+mAh)$`,           // Motor KV or battery capacity
	}

	for _, pattern := range variantPatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(s)
		if len(matches) > 1 {
			variant = strings.TrimSpace(matches[1])
			model = strings.TrimSpace(re.ReplaceAllString(s, ""))
			return model, variant
		}
	}

	// No variant found, entire string is the model
	return s, ""
}
