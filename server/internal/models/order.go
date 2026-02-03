package models

import (
	"time"
)

// Carrier represents a shipping carrier
type Carrier string

const (
	CarrierFedEx Carrier = "fedex"
	CarrierUSPS  Carrier = "usps"
	CarrierUPS   Carrier = "ups"
	CarrierDHL   Carrier = "dhl"
	CarrierOther Carrier = "other"
)

// ShipmentStatus represents the current status of a shipment
type ShipmentStatus string

const (
	StatusLabelCreated   ShipmentStatus = "label_created"
	StatusInTransit      ShipmentStatus = "in_transit"
	StatusOutForDelivery ShipmentStatus = "out_for_delivery"
	StatusDelivered      ShipmentStatus = "delivered"
	StatusException      ShipmentStatus = "exception"
	StatusUnknown        ShipmentStatus = "unknown"
)

// Order represents a shipment tracking entry
type Order struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId,omitempty"`
	Carrier        Carrier        `json:"carrier"`
	TrackingNumber string         `json:"trackingNumber"`
	Label          string         `json:"label,omitempty"` // Optional friendly name (e.g., "DJI Goggles")
	Status         ShipmentStatus `json:"status"`
	StatusDetails  string         `json:"statusDetails,omitempty"` // Human-readable status description
	EstimatedDate  *time.Time     `json:"estimatedDate,omitempty"` // Estimated delivery date
	DeliveredAt    *time.Time     `json:"deliveredAt,omitempty"`   // Actual delivery date
	LastCheckedAt  *time.Time     `json:"lastCheckedAt,omitempty"` // When status was last fetched
	Archived       bool           `json:"archived"`                // Hide from active list
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// AddOrderParams represents the parameters for adding a new order
type AddOrderParams struct {
	Carrier        Carrier `json:"carrier"`
	TrackingNumber string  `json:"trackingNumber"`
	Label          string  `json:"label,omitempty"`
}

// UpdateOrderParams represents the parameters for updating an order
type UpdateOrderParams struct {
	ID             string          `json:"id"`
	Carrier        *Carrier        `json:"carrier,omitempty"`
	TrackingNumber *string         `json:"trackingNumber,omitempty"`
	Label          *string         `json:"label,omitempty"`
	Status         *ShipmentStatus `json:"status,omitempty"`
	StatusDetails  *string         `json:"statusDetails,omitempty"`
	EstimatedDate  *time.Time      `json:"estimatedDate,omitempty"`
	DeliveredAt    *time.Time      `json:"deliveredAt,omitempty"`
	Archived       *bool           `json:"archived,omitempty"`
}

// OrderListResponse contains a list of orders with pagination info
type OrderListResponse struct {
	Orders     []Order `json:"orders"`
	TotalCount int     `json:"total_count"`
}

// ValidCarriers returns all valid carrier values
func ValidCarriers() []Carrier {
	return []Carrier{CarrierFedEx, CarrierUSPS, CarrierUPS, CarrierDHL, CarrierOther}
}

// IsValid checks if a carrier value is valid (method on Carrier)
func (c Carrier) IsValid() bool {
	for _, valid := range ValidCarriers() {
		if c == valid {
			return true
		}
	}
	return false
}

// IsValidCarrier checks if a carrier value is valid
func IsValidCarrier(c Carrier) bool {
	return c.IsValid()
}

// ValidStatuses returns all valid shipment status values
func ValidStatuses() []ShipmentStatus {
	return []ShipmentStatus{StatusLabelCreated, StatusInTransit, StatusOutForDelivery, StatusDelivered, StatusException, StatusUnknown}
}

// IsValid checks if a shipment status value is valid (method on ShipmentStatus)
func (s ShipmentStatus) IsValid() bool {
	for _, valid := range ValidStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

// CarrierDisplayName returns the display name for a carrier
func CarrierDisplayName(c Carrier) string {
	switch c {
	case CarrierFedEx:
		return "FedEx"
	case CarrierUSPS:
		return "USPS"
	case CarrierUPS:
		return "UPS"
	case CarrierDHL:
		return "DHL"
	case CarrierOther:
		return "Other"
	default:
		return string(c)
	}
}

// StatusDisplayName returns the display name for a shipment status
func StatusDisplayName(s ShipmentStatus) string {
	switch s {
	case StatusLabelCreated:
		return "Label Created"
	case StatusInTransit:
		return "In Transit"
	case StatusOutForDelivery:
		return "Out for Delivery"
	case StatusDelivered:
		return "Delivered"
	case StatusException:
		return "Exception"
	case StatusUnknown:
		return "Unknown"
	default:
		return string(s)
	}
}

// MaskedTrackingNumber returns a partially masked tracking number for display
func (o *Order) MaskedTrackingNumber() string {
	if len(o.TrackingNumber) <= 6 {
		return o.TrackingNumber
	}
	// Show first 4 and last 4 characters
	return o.TrackingNumber[:4] + "..." + o.TrackingNumber[len(o.TrackingNumber)-4:]
}

// IsActive returns true if the order is not archived and not yet delivered
func (o *Order) IsActive() bool {
	return !o.Archived && o.Status != StatusDelivered
}
