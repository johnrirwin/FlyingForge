import { useState, useEffect } from 'react';
import { fetchPublicAircraftImage } from '../socialApi';

interface AircraftImageProps {
  aircraftId: string;
  aircraftName: string;
  hasImage: boolean;
  className?: string;
  fallbackIcon?: React.ReactNode;
}

/**
 * Secure aircraft image component that fetches images using Authorization headers
 * instead of exposing tokens in query parameters.
 */
export function AircraftImage({ 
  aircraftId, 
  aircraftName, 
  hasImage, 
  className = '',
  fallbackIcon
}: AircraftImageProps) {
  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    let isMounted = true;
    let blobUrl: string | null = null;

    const loadImage = async () => {
      if (!hasImage) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(false);

      try {
        blobUrl = await fetchPublicAircraftImage(aircraftId);
        
        if (isMounted) {
          if (blobUrl) {
            setImageSrc(blobUrl);
          } else {
            setError(true);
          }
          setLoading(false);
        }
      } catch {
        if (isMounted) {
          setError(true);
          setLoading(false);
        }
      }
    };

    loadImage();

    // Cleanup: revoke blob URL when component unmounts or aircraftId changes
    return () => {
      isMounted = false;
      if (blobUrl) {
        URL.revokeObjectURL(blobUrl);
      }
    };
  }, [aircraftId, hasImage]);

  // Default fallback icon
  const defaultFallback = (
    <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
    </svg>
  );

  if (loading) {
    return (
      <div className={`flex items-center justify-center bg-slate-700 ${className}`}>
        <div className="animate-pulse text-slate-500">Loading...</div>
      </div>
    );
  }

  if (!hasImage || error || !imageSrc) {
    return (
      <div className={`flex items-center justify-center bg-slate-700 ${className}`}>
        {fallbackIcon || defaultFallback}
      </div>
    );
  }

  return (
    <img
      src={imageSrc}
      alt={aircraftName}
      className={className}
      onError={() => setError(true)}
    />
  );
}
