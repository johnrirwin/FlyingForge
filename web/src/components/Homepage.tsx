interface HomepageProps {
  onSignIn: () => void;
  onExploreNews: () => void;
}

// Feature card component
function FeatureCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-6 hover:border-slate-600 transition-colors">
      <div className="w-12 h-12 bg-primary-600/20 rounded-xl flex items-center justify-center mb-4">
        {icon}
      </div>
      <h3 className="text-lg font-semibold text-white mb-2">{title}</h3>
      <p className="text-slate-400 text-sm leading-relaxed">{description}</p>
    </div>
  );
}

// Step component for "How it works"
function Step({
  number,
  title,
  description,
}: {
  number: number;
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="w-14 h-14 bg-primary-600 rounded-full flex items-center justify-center mb-4 shadow-lg shadow-primary-600/25">
        <span className="text-xl font-bold text-white">{number}</span>
      </div>
      <h4 className="text-lg font-semibold text-white mb-2">{title}</h4>
      <p className="text-slate-400 text-sm max-w-xs">{description}</p>
    </div>
  );
}

export function Homepage({ onSignIn, onExploreNews }: HomepageProps) {
  return (
    <div className="flex-1 overflow-y-auto bg-slate-900">
      {/* Hero Section */}
      <section className="relative px-6 py-20 md:py-32 overflow-hidden">
        {/* Background gradient */}
        <div className="absolute inset-0 bg-gradient-to-br from-primary-900/20 via-slate-900 to-slate-900" />
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-primary-600/10 rounded-full blur-3xl" />
        
        <div className="relative max-w-5xl mx-auto text-center">
          {/* Logo */}
          <div className="flex items-center justify-center gap-3 mb-8">
            <div className="w-14 h-14 bg-primary-600 rounded-xl flex items-center justify-center shadow-lg shadow-primary-600/30">
              <svg className="w-8 h-8 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                <circle cx="12" cy="12" r="3"/>
                <path d="M12 1v4M12 19v4M1 12h4M19 12h4"/>
                <path d="M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83"/>
                <path d="M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
              </svg>
            </div>
            <span className="text-3xl font-bold text-white">FlyingForge</span>
          </div>

          {/* Headline */}
          <h1 className="text-4xl md:text-6xl font-bold text-white mb-6 leading-tight">
            Build it. Fly it.<br />
            <span className="text-primary-400">Refine it.</span>
          </h1>

          {/* Subheadline */}
          <p className="text-xl text-slate-300 mb-10 max-w-2xl mx-auto leading-relaxed">
            The home base for drone and fixed-wing hobbyists. Track your aircraft, gear, radios, and batteries—all in one place. Stay current with the hobby and spend more time flying.
          </p>

          {/* CTAs */}
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <button
              onClick={onSignIn}
              className="w-full sm:w-auto px-8 py-4 bg-primary-600 hover:bg-primary-700 text-white font-semibold rounded-xl transition-colors shadow-lg shadow-primary-600/25 flex items-center justify-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
              </svg>
              Get Started Free
            </button>
            <button
              onClick={onExploreNews}
              className="w-full sm:w-auto px-8 py-4 bg-slate-800 hover:bg-slate-700 text-white font-semibold rounded-xl transition-colors border border-slate-700 flex items-center justify-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
              </svg>
              Explore News Feed
            </button>
          </div>
        </div>
      </section>

      {/* What FlyingForge Does - Feature Cards */}
      <section className="px-6 py-20 border-t border-slate-800">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold text-white mb-4">Everything you need for the hobby</h2>
            <p className="text-slate-400 max-w-2xl mx-auto">
              From tracking your builds to staying up-to-date with the community—FlyingForge has you covered.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {/* Track Aircraft */}
            <FeatureCard
              icon={
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
              }
              title="Track Your Aircraft"
              description="Document your builds with photos, specs, and component configurations. Keep a history of every upgrade and repair."
            />

            {/* Track Gear */}
            <FeatureCard
              icon={
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                </svg>
              }
              title="Track Your Gear"
              description="Inventory all your equipment—frames, flight controllers, VTX, receivers, goggles, and more. Know what you have and what's installed where."
            />

            {/* Track Radio */}
            <FeatureCard
              icon={
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.14 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
                </svg>
              }
              title="Track Your Radio"
              description="Keep your transmitter configs organized. Track receiver settings, model match IDs, and have backup configs ready to go."
            />

            {/* Track Batteries */}
            <FeatureCard
              icon={
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h14a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4a2 2 0 012-2zm16 3h2m-2 0v2" />
                </svg>
              }
              title="Track Batteries"
              description="Monitor battery health, log charge cycles, track internal resistance over time. Print labels with QR codes for quick scanning."
            />

            {/* News Feed */}
            <FeatureCard
              icon={
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
                </svg>
              }
              title="News Feed"
              description="Stay plugged into the hobby with a curated feed from top drone news sites, subreddits, and community forums—all in one place."
            />

            {/* Shop */}
            <FeatureCard
              icon={
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
                </svg>
              }
              title="Shop"
              description="Browse equipment from popular FPV retailers. Find the parts you need and add them straight to your inventory when they arrive."
            />
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="px-6 py-20 bg-slate-800/30 border-t border-slate-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold text-white mb-4">How it works</h2>
            <p className="text-slate-400">Get organized in three simple steps</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 md:gap-12">
            <Step
              number={1}
              title="Add Your Gear"
              description="Create your aircraft, add components from your inventory, and upload photos of your builds."
            />
            <Step
              number={2}
              title="Link & Configure"
              description="Connect parts to specific aircraft, set up receiver bindings, and organize your radio model configs."
            />
            <Step
              number={3}
              title="Track & Fly"
              description="Monitor battery health, log repairs and upgrades, and spend less time managing—more time flying."
            />
          </div>
        </div>
      </section>

      {/* For Drone & Fixed Wing */}
      <section className="px-6 py-20 border-t border-slate-800">
        <div className="max-w-4xl mx-auto text-center">
          <div className="flex items-center justify-center gap-8 mb-8">
            {/* Drone icon */}
            <div className="w-20 h-20 bg-slate-800 border border-slate-700 rounded-2xl flex items-center justify-center">
              <svg className="w-10 h-10 text-primary-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                <circle cx="12" cy="12" r="3"/>
                <circle cx="5" cy="5" r="2"/>
                <circle cx="19" cy="5" r="2"/>
                <circle cx="5" cy="19" r="2"/>
                <circle cx="19" cy="19" r="2"/>
                <path d="M9 9L6 6M15 9l3-3M9 15l-3 3M15 15l3 3"/>
              </svg>
            </div>
            <span className="text-4xl text-slate-600 font-light">+</span>
            {/* Fixed wing icon */}
            <div className="w-20 h-20 bg-slate-800 border border-slate-700 rounded-2xl flex items-center justify-center">
              <svg className="w-10 h-10 text-primary-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                <path d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
              </svg>
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white mb-4">
            Built for all RC pilots
          </h2>
          <p className="text-slate-400 max-w-2xl mx-auto leading-relaxed mb-8">
            Whether you're flying FPV quads, cinewhoops, long-range wings, or classic fixed-wing aircraft—FlyingForge adapts to how you fly. Track multirotors, planes, and everything in between.
          </p>

          <div className="flex flex-wrap items-center justify-center gap-3">
            {['FPV Freestyle', 'Racing Quads', 'Cinewhoops', 'Long Range', 'Fixed Wing', 'Planes', 'VTOL'].map((tag) => (
              <span
                key={tag}
                className="px-4 py-2 bg-slate-800 border border-slate-700 rounded-full text-sm text-slate-300"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="px-6 py-20 border-t border-slate-800">
        <div className="max-w-2xl mx-auto text-center">
          <h2 className="text-3xl font-bold text-white mb-4">
            Ready to organize your flying life?
          </h2>
          <p className="text-slate-400 mb-8">
            Join FlyingForge today and get your aircraft, gear, and batteries under control.
          </p>
          <button
            onClick={onSignIn}
            className="px-8 py-4 bg-primary-600 hover:bg-primary-700 text-white font-semibold rounded-xl transition-colors shadow-lg shadow-primary-600/25"
          >
            Create Your Free Account
          </button>
        </div>
      </section>

      {/* Footer */}
      <footer className="px-6 py-12 border-t border-slate-800 bg-slate-900">
        <div className="max-w-6xl mx-auto">
          <div className="flex flex-col md:flex-row items-center justify-between gap-6">
            {/* Logo & tagline */}
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-primary-600 rounded-lg flex items-center justify-center">
                <svg className="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <circle cx="12" cy="12" r="3"/>
                  <path d="M12 1v4M12 19v4M1 12h4M19 12h4"/>
                  <path d="M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83"/>
                  <path d="M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
                </svg>
              </div>
              <div>
                <div className="text-lg font-semibold text-white">FlyingForge</div>
                <div className="text-xs text-slate-500">Build it. Fly it. Refine it.</div>
              </div>
            </div>

            {/* Copyright */}
            <div className="text-sm text-slate-500">
              © {new Date().getFullYear()} FlyingForge
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
