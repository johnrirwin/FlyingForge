import { useState, useEffect } from 'react';
import type { Order, Carrier, AddOrderParams } from '../orderTypes';
import { 
  carrierDisplayNames, 
  statusDisplayNames, 
  statusColors, 
  getCarrierTrackingUrl,
  isActiveOrder 
} from '../orderTypes';
import { getOrders, createOrder, updateOrder, deleteOrder } from '../orderApi';

// Add Order Modal
function AddOrderModal({
  onClose,
  onAdd,
}: {
  onClose: () => void;
  onAdd: (params: AddOrderParams) => Promise<void>;
}) {
  const [carrier, setCarrier] = useState<Carrier>('usps');
  const [trackingNumber, setTrackingNumber] = useState('');
  const [label, setLabel] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!trackingNumber.trim()) {
      setError('Tracking number is required');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await onAdd({
        carrier,
        trackingNumber: trackingNumber.trim(),
        label: label.trim() || undefined,
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add order');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-slate-800 rounded-2xl w-full max-w-md shadow-xl">
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">Add Order</h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {error && (
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-3">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1.5">
              Carrier *
            </label>
            <select
              value={carrier}
              onChange={(e) => setCarrier(e.target.value as Carrier)}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {Object.entries(carrierDisplayNames).map(([value, name]) => (
                <option key={value} value={value}>{name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1.5">
              Tracking Number *
            </label>
            <input
              type="text"
              value={trackingNumber}
              onChange={(e) => setTrackingNumber(e.target.value)}
              placeholder="Enter tracking number"
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1.5">
              Label (optional)
            </label>
            <input
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="e.g., Motors from GetFPV"
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
            <p className="text-xs text-slate-500 mt-1">
              Give this shipment a name to help you remember what's in it
            </p>
          </div>

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 text-white rounded-lg transition-colors"
            >
              {isSubmitting ? 'Adding...' : 'Add Order'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Order Card
function OrderCard({
  order,
  onArchive,
  onDelete,
}: {
  order: Order;
  onArchive: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const { bg, text } = statusColors[order.status];
  const trackingUrl = getCarrierTrackingUrl(order.carrier, order.trackingNumber);
  const isActive = isActiveOrder(order);
  const [showMenu, setShowMenu] = useState(false);

  return (
    <div 
      className={`bg-slate-800 border border-slate-700 rounded-xl p-4 ${isActive ? 'ring-1 ring-primary-500/30' : ''}`}
    >
      <div className="flex gap-4">
        <div className="w-14 h-14 bg-slate-700 rounded-lg flex items-center justify-center flex-shrink-0">
          <svg className="w-7 h-7 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div>
              <h3 className="text-base font-medium text-white truncate">
                {order.label || `${carrierDisplayNames[order.carrier]} Package`}
              </h3>
              <p className="text-sm text-slate-400">
                {carrierDisplayNames[order.carrier]} • {order.trackingNumber}
              </p>
            </div>
            
            <div className="relative">
              <button
                onClick={() => setShowMenu(!showMenu)}
                className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                </svg>
              </button>
              
              {showMenu && (
                <>
                  <div 
                    className="fixed inset-0" 
                    onClick={() => setShowMenu(false)}
                  />
                  <div className="absolute right-0 mt-1 w-40 bg-slate-700 border border-slate-600 rounded-lg shadow-xl z-10">
                    <button
                      onClick={() => {
                        onArchive(order.id);
                        setShowMenu(false);
                      }}
                      className="w-full px-3 py-2 text-left text-sm text-slate-200 hover:bg-slate-600 rounded-t-lg transition-colors"
                    >
                      {order.archived ? 'Unarchive' : 'Archive'}
                    </button>
                    <button
                      onClick={() => {
                        onDelete(order.id);
                        setShowMenu(false);
                      }}
                      className="w-full px-3 py-2 text-left text-sm text-red-400 hover:bg-slate-600 rounded-b-lg transition-colors"
                    >
                      Delete
                    </button>
                  </div>
                </>
              )}
            </div>
          </div>
          
          <div className="flex items-center gap-3 mt-2">
            <span className={`inline-block px-2.5 py-1 ${bg} ${text} rounded-lg text-xs font-medium`}>
              {statusDisplayNames[order.status]}
            </span>
            
            {order.estimatedDate && order.status !== 'delivered' && (
              <span className="text-xs text-slate-400">
                Est. {new Date(order.estimatedDate).toLocaleDateString()}
              </span>
            )}
            
            {order.deliveredAt && (
              <span className="text-xs text-green-400">
                Delivered {new Date(order.deliveredAt).toLocaleDateString()}
              </span>
            )}
          </div>
          
          {order.statusDetails && (
            <p className="text-xs text-slate-500 mt-2">{order.statusDetails}</p>
          )}
        </div>
        
        {trackingUrl && (
          <a
            href={trackingUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="self-center p-2.5 text-slate-400 hover:text-primary-400 hover:bg-slate-700 rounded-lg transition-colors"
            title="Track on carrier website"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </a>
        )}
      </div>
    </div>
  );
}

// Empty State
function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="bg-slate-800/30 border border-slate-700/50 border-dashed rounded-2xl p-8 text-center">
      <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-4">
        <svg className="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
        </svg>
      </div>
      <h3 className="text-lg font-medium text-white mb-2">No Orders Yet</h3>
      <p className="text-slate-400 mb-4">
        Track your FPV shipments and see delivery status at a glance.
      </p>
      <button
        onClick={onAdd}
        className="inline-flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Add Your First Order
      </button>
    </div>
  );
}

// Main Orders Page
export function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showArchived, setShowArchived] = useState(false);

  const loadOrders = async () => {
    setIsLoading(true);
    try {
      const response = await getOrders({ includeArchived: showArchived, limit: 50 });
      setOrders(response.orders);
    } catch (error) {
      console.error('Failed to load orders:', error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadOrders();
  }, [showArchived]);

  const handleAddOrder = async (params: AddOrderParams) => {
    await createOrder(params);
    await loadOrders();
  };

  const handleArchive = async (id: string) => {
    const order = orders.find(o => o.id === id);
    if (!order) return;
    
    try {
      await updateOrder(id, { archived: !order.archived });
      await loadOrders();
    } catch (error) {
      console.error('Failed to archive order:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this order?')) return;
    
    try {
      await deleteOrder(id);
      await loadOrders();
    } catch (error) {
      console.error('Failed to delete order:', error);
    }
  };

  const activeOrders = orders.filter(o => !o.archived && o.status !== 'delivered');
  const completedOrders = orders.filter(o => !o.archived && o.status === 'delivered');
  const archivedOrders = orders.filter(o => o.archived);

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="p-6 max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white mb-1">Orders</h1>
            <p className="text-slate-400">Track your shipments and deliveries</p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Order
          </button>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3 mb-6">
          <label className="flex items-center gap-2 text-sm text-slate-400">
            <input
              type="checkbox"
              checked={showArchived}
              onChange={(e) => setShowArchived(e.target.checked)}
              className="rounded border-slate-600 bg-slate-700 text-primary-600 focus:ring-primary-500"
            />
            Show archived
          </label>
        </div>

        {/* Content */}
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map(i => (
              <div key={i} className="bg-slate-800 border border-slate-700 rounded-xl p-4 animate-pulse">
                <div className="flex gap-4">
                  <div className="w-14 h-14 bg-slate-700 rounded-lg" />
                  <div className="flex-1 space-y-2">
                    <div className="h-5 bg-slate-700 rounded w-1/3" />
                    <div className="h-4 bg-slate-700 rounded w-1/2" />
                    <div className="h-6 bg-slate-700 rounded w-24" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : orders.length === 0 && !showArchived ? (
          <EmptyState onAdd={() => setShowAddModal(true)} />
        ) : (
          <div className="space-y-8">
            {/* Active Orders */}
            {activeOrders.length > 0 && (
              <section>
                <h2 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">
                  In Transit ({activeOrders.length})
                </h2>
                <div className="space-y-3">
                  {activeOrders.map(order => (
                    <OrderCard
                      key={order.id}
                      order={order}
                      onArchive={handleArchive}
                      onDelete={handleDelete}
                    />
                  ))}
                </div>
              </section>
            )}

            {/* Completed Orders */}
            {completedOrders.length > 0 && (
              <section>
                <h2 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">
                  Delivered ({completedOrders.length})
                </h2>
                <div className="space-y-3">
                  {completedOrders.map(order => (
                    <OrderCard
                      key={order.id}
                      order={order}
                      onArchive={handleArchive}
                      onDelete={handleDelete}
                    />
                  ))}
                </div>
              </section>
            )}

            {/* Archived Orders */}
            {showArchived && archivedOrders.length > 0 && (
              <section>
                <h2 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">
                  Archived ({archivedOrders.length})
                </h2>
                <div className="space-y-3 opacity-60">
                  {archivedOrders.map(order => (
                    <OrderCard
                      key={order.id}
                      order={order}
                      onArchive={handleArchive}
                      onDelete={handleDelete}
                    />
                  ))}
                </div>
              </section>
            )}

            {orders.length === 0 && showArchived && (
              <div className="text-center py-8 text-slate-500">
                No archived orders
              </div>
            )}
          </div>
        )}
      </div>

      {/* Add Order Modal */}
      {showAddModal && (
        <AddOrderModal
          onClose={() => setShowAddModal(false)}
          onAdd={handleAddOrder}
        />
      )}
    </div>
  );
}
