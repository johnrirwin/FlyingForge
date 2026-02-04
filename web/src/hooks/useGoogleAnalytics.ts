import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

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

  const eventArgs = [
    'event',
    'page_view',
    {
      page_path: path,
      page_title: title || document.title,
    },
  ];

  // If GA is ready, send immediately; otherwise queue
  if (isGAReady && window.gtag) {
    window.gtag(...eventArgs);
  } else {
    eventQueue.push({ type: 'page_view', args: eventArgs });
  }
}

// Track custom events
export function trackEvent(
  eventName: string,
  params?: Record<string, string | number | boolean>
) {
  if (!GA_MEASUREMENT_ID) return;

  const eventArgs = ['event', eventName, params];

  // If GA is ready, send immediately; otherwise queue
  if (isGAReady && window.gtag) {
    window.gtag(...eventArgs);
  } else {
    eventQueue.push({ type: 'event', args: eventArgs });
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
