// ============================================================================
// Getting Started Page - Public education page for newcomers
// ============================================================================
// Content is structured as constants for easy future migration to CMS/JSON
import { useEffect, useState } from 'react';

interface GettingStartedProps {
  onSignIn: () => void;
}

// ----------------------------------------------------------------------------
// Content Data (structured for future CMS migration)
// ----------------------------------------------------------------------------

interface PathCard {
  id: string;
  title: string;
  icon: 'fpv' | 'camera' | 'fixedwing';
  description: string;
  whoItsFor: string;
  whatYouLearn: string[];
  checklist: string[];
}

const PATHS: PathCard[] = [
  {
    id: 'fpv',
    title: 'FPV Drones',
    icon: 'fpv',
    description: 'First-person view flying with goggles—racing, freestyle, or cinematic.',
    whoItsFor: 'Thrill-seekers who want an immersive experience and full manual control.',
    whatYouLearn: [
      'Manual (acro) flight mode',
      'Building and tuning quads',
      'FPV video systems',
    ],
    checklist: [
      'Radio controller',
      'FPV goggles',
      'Quad (prebuilt or DIY)',
      'Batteries + charger',
    ],
  },
  {
    id: 'camera',
    title: 'Camera Drones',
    icon: 'camera',
    description: 'Aerial photography and videography with GPS-stabilized platforms.',
    whoItsFor: 'Creators, photographers, and hobbyists wanting smooth, easy flight.',
    whatYouLearn: [
      'GPS-assisted flight modes',
      'Camera settings and composition',
      'Safe flying practices',
    ],
    checklist: [
      'Ready-to-fly drone (DJI, etc.)',
      'Spare batteries',
      'ND filters (optional)',
      'FAA registration (if required)',
    ],
  },
  {
    id: 'fixedwing',
    title: 'Fixed Wing',
    icon: 'fixedwing',
    description: 'Traditional RC planes—gliders, trainers, aerobatic, or long-range wings.',
    whoItsFor: 'Classic RC enthusiasts who love the feel of real flight dynamics.',
    whatYouLearn: [
      'Takeoff and landing techniques',
      'Trimming and control surfaces',
      'Thermal soaring (gliders)',
    ],
    checklist: [
      'Radio controller',
      'Trainer plane or foam wing',
      'Batteries + charger',
      'Open flying field',
    ],
  },
];

interface BasicItem {
  id: string;
  title: string;
  description: string;
  icon: 'radio' | 'sim' | 'battery' | 'safety';
}

const BASICS: BasicItem[] = [
  {
    id: 'radio',
    title: 'Controller / Radio',
    description: 'Your transmitter is the most important investment—it works with simulators and real aircraft. Popular choices: RadioMaster, TBS, FrSky.',
    icon: 'radio',
  },
  {
    id: 'sim',
    title: 'Simulator Practice',
    description: 'Crash in the sim, not in real life. A few hours of sim time before your first flight saves hundreds of dollars in repairs.',
    icon: 'sim',
  },
  {
    id: 'battery',
    title: 'Battery + Charging Basics',
    description: 'LiPo batteries need proper care: never over-discharge, use a balance charger, and store at storage voltage when not in use.',
    icon: 'battery',
  },
  {
    id: 'safety',
    title: 'Safety Mindset',
    description: 'Propellers spin fast. Always arm away from people, disarm immediately after landing, and never fly over crowds.',
    icon: 'safety',
  },
];

interface Simulator {
  id: string;
  name: string;
  description: string;
  bestFor: string;
  category: 'fpv' | 'fixedwing';
  link: string;
  linkType: 'steam' | 'official';
}

interface CreatorSpotlight {
  id: string;
  name: string;
  description: string;
  tags: string[];
  youtubeUrl?: string;
  instagramUrl?: string;
  subscribers?: string;
  featured?: boolean;
}

const SIMULATORS: Simulator[] = [
  // FPV Simulators
  {
    id: 'velocidrone',
    name: 'VelociDrone',
    description: 'Physics-accurate FPV sim with tons of maps and active multiplayer racing community.',
    bestFor: 'Realism & Racing',
    category: 'fpv',
    link: 'https://www.velocidrone.com',
    linkType: 'official',
  },
  {
    id: 'liftoff',
    name: 'Liftoff: FPV Drone Racing',
    description: 'Polished graphics and physics with a great tutorial system for beginners.',
    bestFor: 'Learning Basics',
    category: 'fpv',
    link: 'https://store.steampowered.com/app/410340/Liftoff_FPV_Drone_Racing/',
    linkType: 'steam',
  },
  {
    id: 'uncrashed',
    name: 'Uncrashed: FPV Drone Simulator',
    description: 'Modern Unreal Engine graphics with realistic environments and physics.',
    bestFor: 'Freestyle & Visuals',
    category: 'fpv',
    link: 'https://store.steampowered.com/app/1682970/Uncrashed__FPV_Drone_Simulator/',
    linkType: 'steam',
  },
  {
    id: 'drl',
    name: 'The Drone Racing League Simulator',
    description: 'Official Drone Racing League sim with official tracks and competitive modes.',
    bestFor: 'Multiplayer Racing',
    category: 'fpv',
    link: 'https://store.steampowered.com/app/641780/The_Drone_Racing_League_Simulator/',
    linkType: 'steam',
  },
  {
    id: 'tryp',
    name: 'TRYP FPV',
    description: 'Lightweight sim focused on freestyle with creative maps and smooth physics.',
    bestFor: 'Freestyle Practice',
    category: 'fpv',
    link: 'https://store.steampowered.com/app/1881200/TRYP_FPV_Drone_Racer_Simulator/',
    linkType: 'steam',
  },
  // Fixed-Wing Simulators
  {
    id: 'aerofly',
    name: 'aerofly RC 10',
    description: 'Beautiful scenery and smooth flight models, great for casual practice.',
    bestFor: 'Visuals & Ease of Use',
    category: 'fixedwing',
    link: 'https://store.steampowered.com/app/2394350/aerofly_RC_10__RC_Flight_Simulator/',
    linkType: 'steam',
  },
  {
    id: 'realflight',
    name: 'RealFlight',
    description: 'Industry-standard RC flight sim with extensive aircraft library and realistic physics.',
    bestFor: 'Realism & Variety',
    category: 'fixedwing',
    link: 'https://www.realflight.com',
    linkType: 'official',
  },
  {
    id: 'picasim',
    name: 'PicaSim',
    description: 'Free, lightweight glider and slope soaring simulator—perfect for beginners.',
    bestFor: 'Free & Gliders',
    category: 'fixedwing',
    link: 'https://www.rowlhouse.co.uk/PicaSim/',
    linkType: 'official',
  },
];

interface ProgressionStep {
  step: number;
  title: string;
  description: string;
}

const PROGRESSION: ProgressionStep[] = [
  {
    step: 1,
    title: 'Sim Basics',
    description: 'Learn to hover, maintain orientation, and recover from tumbles without breaking anything real.',
  },
  {
    step: 2,
    title: 'Orientation Mastery',
    description: 'Practice flying toward yourself (nose-in) until the controls feel natural from any angle.',
  },
  {
    step: 3,
    title: 'Circuits & Patterns',
    description: 'Fly figure-8s, circuits, and smooth turns. Build muscle memory for controlled flight.',
  },
  {
    step: 4,
    title: 'Skills & Drills',
    description: 'Freestyle tricks, racing gates, precision landings—whatever matches your flying style.',
  },
];

interface SafetyRule {
  id: string;
  rule: string;
  detail: string;
}

const SAFETY_RULES: SafetyRule[] = [
  {
    id: 'los',
    rule: 'Fly line of sight',
    detail: 'Keep your aircraft visible at all times unless you have proper authorization for beyond-visual-line-of-sight (BVLOS) operations.',
  },
  {
    id: 'local',
    rule: 'Check local rules',
    detail: 'Regulations vary by country and region. In the US, register with the FAA if required. Check for no-fly zones before each flight.',
  },
  {
    id: 'respect',
    rule: 'Respect people & property',
    detail: 'Never fly over crowds, near airports, or over private property without permission. Be a good ambassador for the hobby.',
  },
];

const TAKING_OFF_CREATORS: CreatorSpotlight[] = [
  {
    id: 'joshua-bardwell',
    name: 'Joshua Bardwell',
    description: 'The go-to source for FPV building, repair, configuration, and troubleshooting. Thousands of detailed tutorials.',
    tags: ['Tutorials', 'Builds', 'Troubleshooting'],
    youtubeUrl: 'https://www.youtube.com/@JoshuaBardwell',
    instagramUrl: undefined,
    subscribers: '500K+',
    featured: true,
  },
  {
    id: 'mr-steele',
    name: 'Mr Steele',
    description: 'Cinematic freestyle, travel flying, mindset, and high-level setup insights from one of the most recognized pilots.',
    tags: ['Freestyle', 'Cinematic'],
    youtubeUrl: 'https://www.youtube.com/@MrSteeleFPV',
    instagramUrl: undefined,
    subscribers: '400K+',
  },
  {
    id: 'drl',
    name: 'Drone Racing League',
    description: 'Professional FPV racing content, competitive flying highlights, and simulator crossover from the official league.',
    tags: ['Racing', 'Competition'],
    youtubeUrl: 'https://www.youtube.com/@TheDroneRacingLeague',
    instagramUrl: undefined,
    subscribers: '1M+',
  },
  {
    id: 'nick-burns',
    name: 'Nick Burns',
    description: 'Tiny whoops, small drones, and accessible freestyle flying. Great for beginners getting into micro quads.',
    tags: ['Reviews', 'Whoops', 'Beginner Friendly'],
    youtubeUrl: 'https://www.youtube.com/@NickBurnsFPV',
    instagramUrl: undefined,
    subscribers: '100K+',
  },
  {
    id: 'botgrinder',
    name: 'Botgrinder',
    description: 'High-energy freestyle flying with entertaining commentary and urban exploration.',
    tags: ['Freestyle', 'Entertainment'],
    youtubeUrl: 'https://www.youtube.com/@BOTGRINDER',
    instagramUrl: undefined,
    subscribers: '200K+',
  },
  {
    id: 'mads',
    name: 'Mads',
    description: 'Deep technical reviews, builds, and component analysis with thorough testing methodology.',
    tags: ['Technical', 'Reviews'],
    youtubeUrl: 'https://youtube.com/@madrc?si=09AIH8jg7Hbxun_A',
    instagramUrl: undefined,
    subscribers: '82.4K',
  },
  {
    id: 'colibri',
    name: 'Colibri',
    description: 'Creative FPV flying and community-focused content with a strong freestyle style.',
    tags: ['Freestyle', 'Community'],
    youtubeUrl: 'https://youtube.com/@colibrifpv?si=4VrnckgwbuExB7rC',
    instagramUrl: undefined,
    subscribers: '33.8K',
  },
  {
    id: 'kai-vertigoh',
    name: 'Kai Vertigoh',
    description: 'Precision freestyle and dynamic flying lines with educational insights.',
    tags: ['Freestyle', 'Technique'],
    youtubeUrl: 'https://youtube.com/@kaivertigoh?si=MDRinuur-FaUJX7C',
    instagramUrl: undefined,
    subscribers: '260K',
  },
  {
    id: 'rimzler',
    name: 'Rimzler',
    description: 'Fast-paced FPV content and pilot perspective sessions for improving control.',
    tags: ['Freestyle', 'Pilot POV'],
    youtubeUrl: 'https://youtube.com/@rimzler?si=JGYhxU8nfnhiMSjf',
    instagramUrl: undefined,
    subscribers: '37.6K',
  },
  {
    id: 'hurricanefpv',
    name: 'Hurricanefpv',
    description: 'FPV progression, flights, and practical drone-focused content.',
    tags: ['Progression', 'FPV'],
    youtubeUrl: 'https://youtube.com/@fpvdrone-l7b?si=HHgrjlOSJiZ5Pfx4',
    instagramUrl: undefined,
    subscribers: '1.97K',
  },
  {
    id: 'brodie-reed',
    name: 'Brodie Reed',
    description: 'Freestyle sessions and drone storytelling from locations around the world.',
    tags: ['Freestyle', 'Storytelling'],
    youtubeUrl: 'https://youtube.com/@brodiereed?si=chHjaxMhLjCMLuQD',
    instagramUrl: undefined,
    subscribers: '40.2K',
  },
  {
    id: 'le-drib',
    name: 'Le Drib',
    description: 'Adventure-oriented, cinematic freestyle FPV with stunning locations and smooth flying.',
    tags: ['Cinematic', 'Exploration'],
    youtubeUrl: 'https://www.youtube.com/@LeDribFPV',
    instagramUrl: undefined,
    subscribers: '103K',
  },
  {
    id: 'bubby-fpv',
    name: 'Bubby FPV',
    description: 'Cinematic FPV sessions, creative edits, and storytelling-driven flight content.',
    tags: ['Cinematic', 'Freestyle'],
    youtubeUrl: 'https://www.youtube.com/@BubbyFPV',
    instagramUrl: undefined,
    subscribers: '20.5K',
  },
  {
    id: 'captain-vanover',
    name: 'Captain Vanover',
    description: 'Freestyle flying, tutorials, and practical FPV tips from years of field experience.',
    tags: ['Freestyle', 'Tutorials'],
    youtubeUrl: 'https://www.youtube.com/@captainvanover',
    instagramUrl: undefined,
    subscribers: '52.8K',
  },
];

// ----------------------------------------------------------------------------
// Icon Components
// ----------------------------------------------------------------------------

function PathIcon({ type }: { type: PathCard['icon'] }) {
  switch (type) {
    case 'fpv':
      return (
        <svg className="w-8 h-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
          <circle cx="12" cy="12" r="3"/>
          <circle cx="5" cy="5" r="2"/>
          <circle cx="19" cy="5" r="2"/>
          <circle cx="5" cy="19" r="2"/>
          <circle cx="19" cy="19" r="2"/>
          <path d="M9 9L6 6M15 9l3-3M9 15l-3 3M15 15l3 3"/>
        </svg>
      );
    case 'camera':
      return (
        <svg className="w-8 h-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
          <path d="M23 19a2 2 0 01-2 2H3a2 2 0 01-2-2V8a2 2 0 012-2h4l2-3h6l2 3h4a2 2 0 012 2z"/>
          <circle cx="12" cy="13" r="4"/>
        </svg>
      );
    case 'fixedwing':
      return (
        <svg className="w-8 h-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
          <path d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
        </svg>
      );
  }
}

function BasicIcon({ type }: { type: BasicItem['icon'] }) {
  switch (type) {
    case 'radio':
      return (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.14 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
        </svg>
      );
    case 'sim':
      return (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
      );
    case 'battery':
      return (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h14a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4a2 2 0 012-2zm16 3h2m-2 0v2" />
        </svg>
      );
    case 'safety':
      return (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
      );
  }
}

// ----------------------------------------------------------------------------
// Section Components
// ----------------------------------------------------------------------------

function PathCardComponent({ path }: { path: PathCard }) {
  return (
    <div className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-6 hover:border-slate-600 transition-colors">
      <div className="w-14 h-14 bg-primary-600/20 rounded-xl flex items-center justify-center mb-4 text-primary-400">
        <PathIcon type={path.icon} />
      </div>
      <h3 className="text-xl font-semibold text-white mb-2">{path.title}</h3>
      <p className="text-slate-400 text-sm mb-4">{path.description}</p>
      
      <div className="mb-4">
        <div className="text-xs font-medium text-slate-500 uppercase mb-1">Who it's for</div>
        <p className="text-slate-300 text-sm">{path.whoItsFor}</p>
      </div>
      
      <div className="mb-4">
        <div className="text-xs font-medium text-slate-500 uppercase mb-2">What you'll learn</div>
        <ul className="space-y-1">
          {path.whatYouLearn.map((item, i) => (
            <li key={`${path.id}-learn-${i}`} className="text-slate-300 text-sm flex items-start gap-2">
              <svg className="w-4 h-4 text-primary-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4" />
              </svg>
              {item}
            </li>
          ))}
        </ul>
      </div>
      
      <div>
        <div className="text-xs font-medium text-slate-500 uppercase mb-2">Gear checklist</div>
        <ul className="space-y-1">
          {path.checklist.map((item, i) => (
            <li key={`${path.id}-gear-${i}`} className="text-slate-400 text-sm flex items-start gap-2">
              <span className="w-1.5 h-1.5 rounded-full bg-slate-500 mt-2 flex-shrink-0" />
              {item}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function SimulatorCard({ sim }: { sim: Simulator }) {
  return (
    <div className="bg-slate-800 border border-slate-700 rounded-xl p-4 hover:border-slate-600 transition-colors flex flex-col">
      <div className="flex items-start justify-between gap-3 mb-2">
        <h4 className="text-base font-semibold text-white">{sim.name}</h4>
        <span className="px-2 py-0.5 bg-primary-600/20 text-primary-400 text-xs font-medium rounded-full whitespace-nowrap">
          {sim.bestFor}
        </span>
      </div>
      <p className="text-slate-400 text-sm mb-3 flex-1">{sim.description}</p>
      <a
        href={sim.link}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-primary-400 hover:text-primary-300 transition-colors group"
        aria-label={`Visit ${sim.name} on ${sim.linkType === 'steam' ? 'Steam' : 'official site'} (opens in new tab)`}
      >
        {sim.linkType === 'steam' ? (
          <>
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M11.979 0C5.678 0 .511 4.86.022 11.037l6.432 2.658c.545-.371 1.203-.59 1.912-.59.063 0 .125.004.188.006l2.861-4.142V8.91c0-2.495 2.028-4.524 4.524-4.524 2.494 0 4.524 2.031 4.524 4.527s-2.03 4.525-4.524 4.525h-.105l-4.076 2.911c0 .052.004.105.004.159 0 1.875-1.515 3.396-3.39 3.396-1.635 0-3.016-1.173-3.331-2.727L.436 15.27C1.862 20.307 6.486 24 11.979 24c6.627 0 11.999-5.373 11.999-12S18.605 0 11.979 0zM7.54 18.21l-1.473-.61c.262.543.714.999 1.314 1.25 1.297.539 2.793-.076 3.332-1.375.263-.63.264-1.319.005-1.949s-.75-1.121-1.377-1.383c-.624-.26-1.29-.249-1.878-.03l1.523.63c.956.4 1.409 1.5 1.009 2.455-.397.957-1.497 1.41-2.454 1.012zm8.039-6.917h.014c.965 0 1.75.785 1.75 1.75s-.787 1.75-1.75 1.75c-.964 0-1.75-.785-1.75-1.75 0-.965.785-1.75 1.736-1.75zm.003-3.5c2.082 0 3.77 1.69 3.77 3.772s-1.69 3.77-3.772 3.77c-2.082 0-3.771-1.69-3.771-3.772s1.689-3.77 3.773-3.77z"/>
            </svg>
            View on Steam
          </>
        ) : (
          <>
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            Visit Official Site
          </>
        )}
        <svg className="w-3 h-3 opacity-0 -translate-x-1 group-hover:opacity-100 group-hover:translate-x-0 transition-all" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </a>
    </div>
  );
}

function SocialIcon({ platform }: { platform: 'youtube' | 'instagram' }) {
  if (platform === 'youtube') {
    return (
      <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>
      </svg>
    );
  }

  return (
    <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M7.75 2h8.5A5.75 5.75 0 0 1 22 7.75v8.5A5.75 5.75 0 0 1 16.25 22h-8.5A5.75 5.75 0 0 1 2 16.25v-8.5A5.75 5.75 0 0 1 7.75 2Zm0 1.5A4.25 4.25 0 0 0 3.5 7.75v8.5a4.25 4.25 0 0 0 4.25 4.25h8.5a4.25 4.25 0 0 0 4.25-4.25v-8.5a4.25 4.25 0 0 0-4.25-4.25h-8.5ZM17 6.25a1 1 0 1 1 0 2 1 1 0 0 1 0-2ZM12 7a5 5 0 1 1 0 10 5 5 0 0 1 0-10Zm0 1.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7Z"/>
    </svg>
  );
}

function getCreatorSocialLinks(creator: CreatorSpotlight): Array<{ platform: 'youtube' | 'instagram'; url: string }> {
  const socialLinks: Array<{ platform: 'youtube' | 'instagram'; url: string }> = [];
  if (creator.youtubeUrl) {
    socialLinks.push({ platform: 'youtube', url: creator.youtubeUrl });
  }
  if (creator.instagramUrl) {
    socialLinks.push({ platform: 'instagram', url: creator.instagramUrl });
  }
  return socialLinks;
}

function CreatorCard({ creator, onSelect }: { creator: CreatorSpotlight; onSelect: (creator: CreatorSpotlight) => void }) {
  return (
    <button
      type="button"
      onClick={() => onSelect(creator)}
      className={`block w-full text-left bg-slate-800 border rounded-xl p-5 hover:border-slate-500 transition-colors group focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/80 ${
        creator.featured
          ? 'border-primary-500/50 ring-1 ring-primary-500/20'
          : 'border-slate-700'
      }`}
      aria-label={`Open creator details for ${creator.name}`}
    >
      <div className="flex items-start gap-4">
        {/* Creator icon */}
        <div className={`w-12 h-12 rounded-xl flex items-center justify-center flex-shrink-0 ${
          creator.featured ? 'bg-primary-600/20' : 'bg-slate-700'
        }`}>
          <svg className={`w-6 h-6 ${creator.featured ? 'text-primary-400' : 'text-slate-400'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M16 7a4 4 0 11-8 0 4 4 0 018 0ZM4 20a8 8 0 0116 0" />
          </svg>
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-3 mb-1">
            <div className="flex items-center gap-2 min-w-0">
              <h4 className="text-base font-semibold text-white group-hover:text-primary-400 transition-colors">
                {creator.name}
              </h4>
              {creator.featured && (
                <span className="px-1.5 py-0.5 bg-primary-600/20 text-primary-400 text-xs font-medium rounded">
                  Essential
                </span>
              )}
            </div>
          </div>
          
          <p className="text-slate-400 text-sm mb-3 line-clamp-2">
            {creator.description}
          </p>
          
          <div className="flex items-center justify-between gap-2">
            <div className="flex flex-wrap gap-1.5">
              {creator.tags.map(tag => (
                <span
                  key={tag}
                  className="px-2 py-0.5 bg-slate-700 text-slate-300 text-xs rounded-full"
                >
                  {tag}
                </span>
              ))}
            </div>
            {creator.subscribers && (
              <span className="text-xs text-slate-500 whitespace-nowrap">
                {creator.subscribers} subscribers
              </span>
            )}
          </div>
        </div>
      </div>
    </button>
  );
}

function CreatorLinksModal({ creator, onClose }: { creator: CreatorSpotlight | null; onClose: () => void }) {
  useEffect(() => {
    if (!creator) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', handleEscape);
    return () => {
      window.removeEventListener('keydown', handleEscape);
    };
  }, [creator, onClose]);

  if (!creator) return null;

  const socialLinks = getCreatorSocialLinks(creator);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={`creator-links-title-${creator.id}`}
        className="relative w-full max-w-md rounded-2xl border border-slate-700 bg-slate-800 p-6 shadow-2xl"
      >
        <button
          onClick={onClose}
          className="absolute right-4 top-4 rounded-lg p-2 text-slate-400 hover:bg-slate-700 hover:text-white transition-colors"
          aria-label="Close creator details"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <h3 id={`creator-links-title-${creator.id}`} className="text-xl font-semibold text-white pr-10">
          {creator.name}
        </h3>
        <p className="mt-2 text-sm text-slate-400">{creator.description}</p>
        {creator.subscribers && (
          <p className="mt-3 text-sm text-slate-300">
            {creator.subscribers} subscribers
          </p>
        )}

        <div className="mt-5 space-y-3">
          {socialLinks.map(link => (
            <a
              key={link.platform}
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              className={`w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                link.platform === 'youtube'
                  ? 'bg-primary-600 hover:bg-primary-500 text-white'
                  : 'bg-slate-700 hover:bg-slate-600 text-white'
              }`}
            >
              <SocialIcon platform={link.platform} />
              {link.platform === 'youtube' ? 'Open YouTube channel' : 'Open Instagram profile'}
            </a>
          ))}
          {socialLinks.length === 0 && (
            <p className="text-sm text-slate-500">
              No external links are available yet.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ----------------------------------------------------------------------------
// Main Component
// ----------------------------------------------------------------------------

export function GettingStarted({ onSignIn }: GettingStartedProps) {
  const fpvSims = SIMULATORS.filter(s => s.category === 'fpv');
  const fixedWingSims = SIMULATORS.filter(s => s.category === 'fixedwing');
  const [selectedCreator, setSelectedCreator] = useState<CreatorSpotlight | null>(null);

  return (
    <div className="flex-1 overflow-y-auto bg-slate-900">
      {/* Hero Section */}
      <section className="relative px-6 py-16 md:py-24 overflow-hidden border-b border-slate-800">
        <div className="absolute inset-0 bg-gradient-to-br from-primary-900/10 via-slate-900 to-slate-900" />
        <div className="relative max-w-4xl mx-auto text-center">
          <h1 className="text-4xl md:text-5xl font-bold text-white mb-4">
            Taking Off
          </h1>
          <p className="text-xl text-slate-300 max-w-2xl mx-auto">
            Everything you need to begin flying drones or fixed-wing—without wasting money.
          </p>
        </div>
      </section>

      {/* Choose Your Path */}
      <section className="px-6 py-16 border-b border-slate-800">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl md:text-3xl font-bold text-white mb-3">Choose your path</h2>
            <p className="text-slate-400 max-w-xl mx-auto">
              Different types of flying suit different people. Pick what excites you most.
            </p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {PATHS.map(path => (
              <PathCardComponent key={path.id} path={path} />
            ))}
          </div>
        </div>
      </section>

      {/* The Basics You'll Need */}
      <section className="px-6 py-16 bg-slate-800/30 border-b border-slate-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl md:text-3xl font-bold text-white mb-3">The basics you'll need</h2>
            <p className="text-slate-400 max-w-xl mx-auto">
              Before your first flight, understand these fundamentals.
            </p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {BASICS.map(basic => (
              <div key={basic.id} className="bg-slate-800 border border-slate-700 rounded-xl p-5">
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 bg-primary-600/20 rounded-lg flex items-center justify-center flex-shrink-0 text-primary-400">
                    <BasicIcon type={basic.icon} />
                  </div>
                  <div>
                    <h3 className="text-base font-semibold text-white mb-1">{basic.title}</h3>
                    <p className="text-slate-400 text-sm leading-relaxed">{basic.description}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Best Simulators */}
      <section className="px-6 py-16 border-b border-slate-800">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl md:text-3xl font-bold text-white mb-3">Best simulators to practice</h2>
            <p className="text-slate-400 max-w-xl mx-auto">
              Plug in your radio and build muscle memory before risking real hardware.
            </p>
          </div>

          {/* FPV Simulators */}
          <div className="mb-10">
            <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <svg className="w-5 h-5 text-primary-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                <circle cx="12" cy="12" r="3"/>
                <circle cx="5" cy="5" r="2"/>
                <circle cx="19" cy="5" r="2"/>
                <circle cx="5" cy="19" r="2"/>
                <circle cx="19" cy="19" r="2"/>
                <path d="M9 9L6 6M15 9l3-3M9 15l-3 3M15 15l3 3"/>
              </svg>
              FPV Simulators
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {fpvSims.map(sim => (
                <SimulatorCard key={sim.id} sim={sim} />
              ))}
            </div>
          </div>

          {/* Fixed-Wing Simulators */}
          <div>
            <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <svg className="w-5 h-5 text-primary-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                <path d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
              </svg>
              Fixed-Wing Simulators
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {fixedWingSims.map(sim => (
                <SimulatorCard key={sim.id} sim={sim} />
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* YouTube Creators */}
      <section className="px-6 py-16 border-b border-slate-800">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl md:text-3xl font-bold text-white mb-3">Learn from the pros</h2>
            <p className="text-slate-400 max-w-xl mx-auto">
              Some of the best FPV and drone education lives on YouTube and Instagram. These creators are trusted across the community.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {TAKING_OFF_CREATORS.map(creator => (
              <CreatorCard key={creator.id} creator={creator} onSelect={setSelectedCreator} />
            ))}
          </div>

          <div className="text-center mt-8">
            <p className="text-slate-500 text-sm">
              Links open in a new tab. These are community recommendations—we're not affiliated with any creators.
            </p>
          </div>
        </div>
      </section>

      {/* Training Progression */}
      <section className="px-6 py-16 bg-slate-800/30 border-b border-slate-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl md:text-3xl font-bold text-white mb-3">Training progression</h2>
            <p className="text-slate-400 max-w-xl mx-auto">
              Follow this path from zero to confident pilot.
            </p>
          </div>
          <div className="relative">
            {/* Connection line */}
            <div className="absolute left-6 top-8 bottom-8 w-0.5 bg-slate-700 hidden md:block" />
            
            <div className="space-y-6">
              {PROGRESSION.map((step) => (
                <div key={step.step} className="relative flex gap-6">
                  {/* Step number */}
                  <div className="w-12 h-12 bg-primary-600 rounded-full flex items-center justify-center flex-shrink-0 relative z-10 shadow-lg shadow-primary-600/25">
                    <span className="text-lg font-bold text-white">{step.step}</span>
                  </div>
                  {/* Content */}
                  <div className="flex-1 bg-slate-800 border border-slate-700 rounded-xl p-5 mt-0.5">
                    <h3 className="text-base font-semibold text-white mb-1">{step.title}</h3>
                    <p className="text-slate-400 text-sm">{step.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Regulations and Safety */}
      <section className="px-6 py-16 border-b border-slate-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl md:text-3xl font-bold text-white mb-3">Regulations and Safety</h2>
            <p className="text-slate-400 max-w-xl mx-auto">
              Flying is fun—but do it responsibly. These basics apply almost everywhere.
            </p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {SAFETY_RULES.map(rule => (
              <div key={rule.id} className="bg-slate-800 border border-slate-700 rounded-xl p-5">
                <div className="w-10 h-10 bg-yellow-500/20 rounded-lg flex items-center justify-center mb-4">
                  <svg className="w-5 h-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <h3 className="text-base font-semibold text-white mb-2">{rule.rule}</h3>
                <p className="text-slate-400 text-sm leading-relaxed">{rule.detail}</p>
              </div>
            ))}
          </div>
          <div className="text-center mt-8">
            <p className="text-slate-500 text-sm mb-4">
              This is general guidance—always check your local regulations before flying.
            </p>
            <a
              href="https://www.faa.gov/uas"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-lg text-sm text-slate-300 hover:text-white transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
              </svg>
              FAA Drone Regulations (US)
            </a>
          </div>
        </div>
      </section>

      {/* Call to Action */}
      <section className="px-6 py-20">
        <div className="max-w-2xl mx-auto text-center">
          <h2 className="text-2xl md:text-3xl font-bold text-white mb-4">
            Ready to start tracking your gear?
          </h2>
          <p className="text-slate-400 mb-8">
            Create your FlyingForge account and set up your first aircraft, radio, and batteries—all in one place.
          </p>
          <button
            onClick={onSignIn}
            className="px-8 py-4 bg-primary-600 hover:bg-primary-700 text-white font-semibold rounded-xl transition-colors shadow-lg shadow-primary-600/25 inline-flex items-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create your first setup in FlyingForge
          </button>
        </div>
      </section>

      <CreatorLinksModal
        creator={selectedCreator}
        onClose={() => setSelectedCreator(null)}
      />

      {/* Footer */}
      <footer className="px-6 py-8 border-t border-slate-800 bg-slate-900">
        <div className="max-w-6xl mx-auto text-center text-sm text-slate-500">
          <p>FlyingForge — Build it. Fly it. Share it.</p>
        </div>
      </footer>
    </div>
  );
}
