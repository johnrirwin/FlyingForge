import { useEffect, useRef } from 'react';
import type { GearCatalogItem } from '../gearCatalogTypes';
import {
  GEAR_TYPES,
  DRONE_TYPES,
  extractDomainFromUrl,
  getCatalogItemDisplayName,
  getShoppingLinkDisplayName,
} from '../gearCatalogTypes';

interface GearDetailModalProps {
  item: GearCatalogItem;
  isOpen: boolean;
  onClose: () => void;
  onAddToInventory?: (item: GearCatalogItem) => void;
  isAuthenticated: boolean;
}

export function GearDetailModal({
  item,
  isOpen,
  onClose,
  onAddToInventory,
  isAuthenticated,
}: GearDetailModalProps) {
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const modalRef = useRef<HTMLDivElement>(null);

  // Handle focus management
  useEffect(() => {
    if (isOpen) {
      // Save the currently focused element
      previousFocusRef.current = document.activeElement as HTMLElement;
      // Focus the modal container
      modalRef.current?.focus();
    } else {
      // Return focus to the trigger element
      previousFocusRef.current?.focus();
    }
  }, [isOpen]);

  // Handle Escape key
  useEffect(() => {
    if (!isOpen) return;
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const typeLabel = GEAR_TYPES.find(t => t.value === item.gearType)?.label || item.gearType;
  const displayName = getCatalogItemDisplayName(item);
  const imageSourceDomain = (item.imageSourceDomain || extractDomainFromUrl(item.imageUrl)).trim();
  const shoppingLinks = (item.shoppingLinks || []).map((link) => link.trim()).filter(Boolean);
  const titleId = `gear-detail-title-${item.id}`;

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const handleAddClick = () => {
    onAddToInventory?.(item);
    onClose();
  };

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      onClick={handleBackdropClick}
    >
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div 
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className="relative bg-slate-800 border border-slate-700 rounded-2xl w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col focus:outline-none"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <h2 id={titleId} className="text-lg font-semibold text-white">{displayName}</h2>
            <span className="px-2 py-0.5 bg-slate-700 text-slate-300 text-xs rounded-full">
              {typeLabel}
            </span>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="flex flex-col md:flex-row gap-6">
            {/* Image */}
            <div className="flex-shrink-0">
              {item.imageUrl ? (
                <div>
                  <img
                    src={item.imageUrl}
                    alt={displayName}
                    className="w-full md:w-48 h-48 rounded-xl object-cover"
                  />
                  {imageSourceDomain && (
                    <p className="mt-2 text-xs text-slate-400">Image via {imageSourceDomain}</p>
                  )}
                </div>
              ) : (
                <div className="w-full md:w-48 h-48 bg-slate-700 rounded-xl flex items-center justify-center">
                  <svg className="w-16 h-16 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                  </svg>
                </div>
              )}
            </div>

            {/* Details */}
            <div className="flex-1 min-w-0">
              {/* Brand */}
              <p className="text-slate-400 text-sm mb-2">{item.brand}</p>

              {/* Usage stats */}
              <div className="flex items-center gap-3 text-sm text-slate-500 mb-4">
                <span className="flex items-center gap-1">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                  {item.usageCount} {item.usageCount === 1 ? 'pilot uses this' : 'pilots use this'}
                </span>
              </div>

              {/* Best For badges */}
              {item.bestFor && item.bestFor.length > 0 && (
                <div className="mb-4">
                  <p className="text-sm text-slate-400 mb-2">Best for:</p>
                  <div className="flex flex-wrap gap-2">
                    {item.bestFor.map(droneType => {
                      const label = DRONE_TYPES.find(t => t.value === droneType)?.label || droneType;
                      return (
                        <span 
                          key={droneType}
                          className="px-3 py-1 bg-primary-600/20 text-primary-400 text-sm rounded-full"
                        >
                          {label}
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* MSRP */}
              {item.msrp !== undefined && item.msrp > 0 && (
                <div className="mb-4">
                  <p className="text-sm text-slate-400">MSRP:</p>
                  <p className="text-white font-medium">${item.msrp.toFixed(2)}</p>
                </div>
              )}
            </div>
          </div>

          {/* Description */}
          {item.description && (
            <div className="mt-6">
              <h3 className="text-sm font-medium text-slate-400 mb-2">Description</h3>
              <p className="text-slate-300 text-sm leading-relaxed whitespace-pre-wrap">
                {item.description}
              </p>
            </div>
          )}

          {/* Specs */}
          {item.specs && Object.keys(item.specs).length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-medium text-slate-400 mb-3">Specifications</h3>
              <div className="bg-slate-900/50 rounded-lg p-4">
                <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                  {Object.entries(item.specs).map(([key, value]) => (
                    <div key={key} className="contents">
                      <dt className="text-slate-500 capitalize">{key.replace(/_/g, ' ')}</dt>
                      <dd className="text-slate-300">{String(value)}</dd>
                    </div>
                  ))}
                </dl>
              </div>
            </div>
          )}

          {shoppingLinks.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-medium text-slate-400 mb-2">Shopping Links</h3>
              <div className="space-y-2">
                {shoppingLinks.map((link, index) => {
                  const linkDomain = getShoppingLinkDisplayName(link);
                  const linkLabel = linkDomain || `Buy link ${index + 1}`;
                  return (
                  <div key={`${link}-${index}`}>
                    <a
                      href={link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex text-sm text-primary-300 hover:text-primary-200"
                      aria-label={`Visit ${linkLabel} (opens in new tab)`}
                    >
                      {linkLabel}
                    </a>
                  </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        {onAddToInventory && (
          <div className="flex items-center justify-end px-6 py-4 border-t border-slate-700 bg-slate-800/50">
            <button
              onClick={handleAddClick}
              disabled={!isAuthenticated}
              title={isAuthenticated ? 'Add to your inventory' : 'Sign in to add to inventory'}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-slate-700 disabled:text-slate-500 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add to My Inventory
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
