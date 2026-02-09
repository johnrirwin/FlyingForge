import { useState } from 'react';
import { MobileFloatingControls } from './MobileFloatingControls';

interface ShopStore {
  name: string;
  description: string;
  url: string;
  logo: string;
  tags: string[];
  color: string;
}

const DRONE_SHOPS: ShopStore[] = [
  {
    name: 'GetFPV',
    description: 'One of the largest FPV drone retailers with a huge selection of frames, motors, electronics, and ready-to-fly drones.',
    url: 'https://www.getfpv.com',
    logo: '🚁',
    tags: ['FPV', 'Parts', 'RTF', 'Electronics'],
    color: 'from-orange-500/20 to-red-500/20',
  },
  {
    name: 'RaceDayQuads',
    description: 'Premium FPV equipment retailer known for quality products, fast shipping, and excellent customer service.',
    url: 'https://www.racedayquads.com',
    logo: '🏁',
    tags: ['FPV', 'Racing', 'Freestyle', 'Parts'],
    color: 'from-blue-500/20 to-purple-500/20',
  },
];

export function ShopSection() {
  const [isMobileControlsOpen, setIsMobileControlsOpen] = useState(false);

  const controls = (
    <div className="px-4 md:px-6 py-4 border-b border-slate-800 bg-slate-900">
      <h1 className="text-xl font-semibold text-white">Shop Equipment</h1>
      <p className="text-sm text-slate-400">
        Trusted retailers for drone parts and gear
      </p>
    </div>
  );

  return (
    <div className="relative flex-1 min-h-0 flex flex-col overflow-hidden">
      <div className="hidden md:block flex-shrink-0">{controls}</div>

      {/* Shop Cards */}
      <div
        className="flex-1 min-h-0 overflow-y-auto p-4 md:p-6 pt-24 md:pt-6"
        onScroll={() => setIsMobileControlsOpen((prev) => (prev ? false : prev))}
      >
        <div className="grid gap-4 md:gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 items-stretch">
          {DRONE_SHOPS.map((shop) => (
            <a
              key={shop.name}
              href={shop.url}
              target="_blank"
              rel="noopener noreferrer"
              className="group block h-full"
            >
              <div className={`relative overflow-hidden rounded-xl border border-slate-700 bg-gradient-to-br ${shop.color} backdrop-blur-sm transition-all duration-300 hover:border-slate-500 hover:shadow-lg hover:shadow-slate-900/50 hover:-translate-y-1 h-full min-h-[280px] flex flex-col`}>
                {/* Card Content */}
                <div className="p-6 h-full flex flex-col">
                  {/* Logo and Name */}
                  <div className="flex items-center gap-4 mb-4">
                    <div className="w-14 h-14 rounded-xl bg-slate-800/80 flex items-center justify-center text-3xl">
                      {shop.logo}
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-white group-hover:text-primary-400 transition-colors">
                        {shop.name}
                      </h3>
                      <p className="text-xs text-slate-500 truncate max-w-[180px]">
                        {shop.url.replace('https://www.', '')}
                      </p>
                    </div>
                  </div>

                  {/* Description */}
                  <p className="text-sm text-slate-300 leading-relaxed mb-4 line-clamp-3 min-h-[4.5rem]">
                    {shop.description}
                  </p>

                  {/* Tags */}
                  <div className="flex flex-wrap gap-2 mb-4 min-h-[2rem]">
                    {shop.tags.map((tag) => (
                      <span
                        key={tag}
                        className="px-2 py-1 text-xs font-medium rounded-md bg-slate-800/60 text-slate-400"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>

                  {/* Visit Button */}
                  <div className="flex items-center justify-between pt-4 border-t border-slate-700/50 mt-auto">
                    <span className="text-sm text-slate-400">Visit Store</span>
                    <svg 
                      className="w-5 h-5 text-slate-400 group-hover:text-primary-400 group-hover:translate-x-1 transition-all" 
                      fill="none" 
                      stroke="currentColor" 
                      viewBox="0 0 24 24"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
                    </svg>
                  </div>
                </div>

                {/* Hover Glow Effect */}
                <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                  <div className="absolute inset-0 bg-gradient-to-t from-primary-500/5 to-transparent" />
                </div>
              </div>
            </a>
          ))}

          {/* Add More Card */}
          <div className="relative overflow-hidden rounded-xl border border-dashed border-slate-700 bg-slate-800/20 p-6 flex flex-col items-center justify-center text-center min-h-[280px] h-full">
            <div className="w-14 h-14 rounded-xl bg-slate-800/50 flex items-center justify-center mb-4">
              <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 4v16m8-8H4" />
              </svg>
            </div>
            <h3 className="text-sm font-medium text-slate-400 mb-1">More Coming Soon</h3>
            <p className="text-xs text-slate-500">
              Additional trusted retailers will be added
            </p>
          </div>
        </div>
      </div>

      <MobileFloatingControls
        label="Shop Info"
        isOpen={isMobileControlsOpen}
        onToggle={() => setIsMobileControlsOpen((prev) => !prev)}
      >
        {controls}
      </MobileFloatingControls>
    </div>
  );
}
