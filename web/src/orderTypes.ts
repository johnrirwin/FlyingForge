// Order types for the frontend

export type Carrier = 'fedex' | 'usps' | 'ups' | 'dhl' | 'other';

export type ShipmentStatus = 
  | 'label_created'
  | 'in_transit'
  | 'out_for_delivery'
  | 'delivered'
  | 'exception'
  | 'unknown';

export interface Order {
  id: string;
  userId: string;
  carrier: Carrier;
  trackingNumber: string;
  label: string;
  status: ShipmentStatus;
  statusDetails: string;
  estimatedDate?: string;
  deliveredAt?: string;
  lastCheckedAt?: string;
  archived: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AddOrderParams {
  carrier: Carrier;
  trackingNumber: string;
  label?: string;
}

export interface UpdateOrderParams {
  carrier?: Carrier;
  trackingNumber?: string;
  label?: string;
  status?: ShipmentStatus;
  statusDetails?: string;
  estimatedDate?: string;
  deliveredAt?: string;
  archived?: boolean;
}

export interface OrderListResponse {
  orders: Order[];
  total_count: number;
}

// Helper functions for display

export const carrierDisplayNames: Record<Carrier, string> = {
  fedex: 'FedEx',
  usps: 'USPS',
  ups: 'UPS',
  dhl: 'DHL',
  other: 'Other',
};

export const statusDisplayNames: Record<ShipmentStatus, string> = {
  label_created: 'Label Created',
  in_transit: 'In Transit',
  out_for_delivery: 'Out for Delivery',
  delivered: 'Delivered',
  exception: 'Exception',
  unknown: 'Unknown',
};

export const statusColors: Record<ShipmentStatus, { bg: string; text: string }> = {
  label_created: { bg: 'bg-gray-100', text: 'text-gray-700' },
  in_transit: { bg: 'bg-blue-100', text: 'text-blue-700' },
  out_for_delivery: { bg: 'bg-yellow-100', text: 'text-yellow-700' },
  delivered: { bg: 'bg-green-100', text: 'text-green-700' },
  exception: { bg: 'bg-red-100', text: 'text-red-700' },
  unknown: { bg: 'bg-gray-100', text: 'text-gray-500' },
};

export function getCarrierTrackingUrl(carrier: Carrier, trackingNumber: string): string | null {
  const urls: Record<Carrier, (tn: string) => string | null> = {
    fedex: (tn) => `https://www.fedex.com/fedextrack/?trknbr=${tn}`,
    usps: (tn) => `https://tools.usps.com/go/TrackConfirmAction?tLabels=${tn}`,
    ups: (tn) => `https://www.ups.com/track?tracknum=${tn}`,
    dhl: (tn) => `https://www.dhl.com/us-en/home/tracking/tracking-global-forwarding.html?submit=1&tracking-id=${tn}`,
    other: () => null,
  };

  return urls[carrier](trackingNumber);
}

export function isActiveOrder(order: Order): boolean {
  return order.status !== 'delivered' && !order.archived;
}

export function maskTrackingNumber(trackingNumber: string): string {
  if (trackingNumber.length <= 8) {
    return trackingNumber;
  }
  const lastFour = trackingNumber.slice(-4);
  return `****${lastFour}`;
}
