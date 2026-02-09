import { useEffect, useId, useRef } from 'react';
import type { ReactNode } from 'react';

interface MobileFloatingControlsProps {
  label: string;
  isOpen: boolean;
  onToggle: () => void;
  children: ReactNode;
  className?: string;
  panelClassName?: string;
}

export function MobileFloatingControls({
  label,
  isOpen,
  onToggle,
  children,
  className,
  panelClassName,
}: MobileFloatingControlsProps) {
  const panelId = useId();
  const panelRef = useRef<HTMLDivElement | null>(null);
  const isClosingRef = useRef(false);

  useEffect(() => {
    if (!isOpen) {
      isClosingRef.current = false;
    }
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) return;

    const closeControls = () => {
      if (isClosingRef.current) return;
      isClosingRef.current = true;
      onToggle();
    };

    const handlePointerScroll = (event: WheelEvent | TouchEvent) => {
      const target = event.target;
      if (target instanceof Node && panelRef.current?.contains(target)) {
        return;
      }
      closeControls();
    };

    window.addEventListener('scroll', closeControls, { passive: true });
    window.addEventListener('touchmove', handlePointerScroll, { passive: true });
    window.addEventListener('wheel', handlePointerScroll, { passive: true });

    return () => {
      window.removeEventListener('scroll', closeControls);
      window.removeEventListener('touchmove', handlePointerScroll);
      window.removeEventListener('wheel', handlePointerScroll);
    };
  }, [isOpen, onToggle]);

  return (
    <div className={`md:hidden absolute top-5 left-4 right-4 z-20 pointer-events-none ${className ?? ''}`}>
      <div className="pointer-events-auto">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={isOpen}
          aria-controls={panelId}
          className="w-full flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-slate-700 bg-slate-800/95 backdrop-blur text-white font-medium shadow-lg shadow-slate-950/30"
        >
          <span className="truncate">{label}</span>
          <svg
            className={`w-5 h-5 text-slate-300 transition-transform ${isOpen ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {isOpen && (
          <div
            id={panelId}
            ref={panelRef}
            role="region"
            aria-label={label}
            className={`mt-2 rounded-xl border border-slate-700 overflow-hidden shadow-2xl shadow-slate-950/40 bg-slate-900/95 backdrop-blur max-h-[70vh] overflow-y-auto ${panelClassName ?? ''}`}
          >
            {children}
          </div>
        )}
      </div>
    </div>
  );
}
