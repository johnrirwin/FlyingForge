import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

/**
 * Google Analytics 4 Integration Hook
 *
 * This hook provides GA4 tracking functionality for the application.
 *
 * THIRD-PARTY DEPENDENCY NOTICE:
 * This integration loads the gtag.js script dynamically from Google's servers
 * (googletagmanager.com). This is standard practice for GA4 and required for
 * the service to function, as Google frequently updates the script.
 *
 * CONTENT SECURITY POLICY (CSP):
 * If using CSP headers, ensure the following domains are allowed:
 * - script-src: https://www.googletagmanager.com
 * - connect-src: https://www.google-analytics.com https://analytics.google.com
 *
 * PRIVACY CONSIDERATIONS:
 * - GA collects user interaction data (page views, events, timing)
 * - IP addresses are anonymized by default in GA4
 * - Consider adding a cookie consent banner for GDPR/CCPA compliance
 * - Review Google's data processing terms: https://business.safety.google/adsprocessorterms/
 *
 * SUBRESOURCE INTEGRITY (SRI):
 * SRI is not used because Google updates the gtag.js script without changing
 * the URL. This is standard for all GA implementations.
 */

// Declare gtag on window for TypeScript
declare global {
  interface Window {
    dataLayer: unknown[];
    gtag: (...args: unknown[]) => void;
  }
}

const GA_MEASUREMENT_ID = import.meta.env.VITE_GA_MEASUREMENT_ID;

// Delay before tracking page view to allow React to update document.title
// after route changes. This ensures we capture the correct page title.
const PAGE_TITLE_UPDATE_DELAY_MS = 100;

// Queue to hold events until GA is fully initialized
// This prevents race conditions where trackPageView is called before gtag loads
type QueuedEvent = { type: 'page_view' | 'event'; args: unknown[] };
const eventQueue: QueuedEvent[] = [];
let isGAReady = false;

// Process queued events once GA is ready
function flushEventQueue() {
  if (!window.gtag) return;
  
  while (eventQueue.length > 0) {
    const event = eventQueue.shift();
    if (event) {
      window.gtag(...event.args);
    }
  }
}

// Initialize Google Analytics
function initGA() {
  if (!GA_MEASUREMENT_ID || typeof window === 'undefined') return;

  // Don't initialize if already done
  if (isGAReady) return;

  // Add gtag.js script
  const script = document.createElement('script');
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${GA_MEASUREMENT_ID}`;
  
  // Mark as ready and flush queue when script loads
  script.onload = () => {
    isGAReady = true;
    flushEventQueue();
  };
  
  document.head.appendChild(script);

  // Initialize dataLayer and gtag function immediately
  // Events will be queued in dataLayer even before script loads
  window.dataLayer = window.dataLayer || [];
  window.gtag = function gtag(...args: unknown[]) {
    window.dataLayer.push(args);
  };

  window.gtag('js', new Date());
  window.gtag('config', GA_MEASUREMENT_ID, {
    send_page_view: false, // We'll send page views manually on route changes
  });
}

// Track page view
export function trackPageView(path: string, title?: string) {
  if (!GA_MEASUREMENT_ID) return;

  // gtag function is created during initGA and pushes to dataLayer,
  // which GA will process once the script loads
  if (window.gtag) {
    window.gtag('event', 'page_view', {
      page_path: path,
      page_title: title || document.title,
    });
  } else {
    // Queue for when GA initializes
    eventQueue.push({
      type: 'page_view',
      args: ['event', 'page_view', { page_path: path, page_title: title || document.title }],
    });
  }
}

// Track custom events
export function trackEvent(
  eventName: string,
  params?: Record<string, string | number | boolean>
) {
  if (!GA_MEASUREMENT_ID) return;

  if (window.gtag) {
    window.gtag('event', eventName, params);
  } else {
    eventQueue.push({ type: 'event', args: ['event', eventName, params] });
  }
}

// Hook to initialize GA and track page views on route changes
export function useGoogleAnalytics() {
  const location = useLocation();

  // Initialize GA on mount
  useEffect(() => {
    initGA();
  }, []);

  // Track page views on route changes
  useEffect(() => {
    if (!GA_MEASUREMENT_ID) return;

    const timeoutId = setTimeout(() => {
      trackPageView(location.pathname + location.search);
    }, PAGE_TITLE_UPDATE_DELAY_MS);

    return () => clearTimeout(timeoutId);
  }, [location.pathname, location.search]);

  return { trackEvent, trackPageView };
}
