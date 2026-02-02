import { useState, useEffect, useCallback } from 'react';
import { getPilotProfile } from '../pilotApi';
import type { PilotProfile as PilotProfileType, AircraftPublic } from '../socialTypes';

interface PilotProfileProps {
  pilotId: string;
  onBack: () => void;
}

export function PilotProfile({ pilotId, onBack }: PilotProfileProps) {
  const [profile, setProfile] = useState<PilotProfileType | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadProfile = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getPilotProfile(pilotId);
      setProfile(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load pilot profile');
    } finally {
      setIsLoading(false);
    }
  }, [pilotId]);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const getDisplayName = () => {
    if (!profile) return 'Unknown Pilot';
    if (profile.callSign) return profile.callSign;
    if (profile.displayName) return profile.displayName;
    if (profile.googleName) return profile.googleName;
    return 'Unknown Pilot';
  };

  const getSecondaryName = () => {
    if (!profile) return null;
    if (profile.callSign && profile.displayName) return profile.displayName;
    if (profile.callSign && profile.googleName) return profile.googleName;
    return null;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Search
        </button>
        <div className="p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400 max-w-2xl">
          {error}
        </div>
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="p-6">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Search
        </button>
        <div className="text-slate-500">Pilot not found</div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Back Button */}
      <button
        onClick={onBack}
        className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Back to Search
      </button>

      {/* Profile Header */}
      <div className="bg-slate-800 rounded-lg p-6 mb-6 max-w-2xl">
        <div className="flex items-center gap-6">
          {/* Avatar */}
          {profile.effectiveAvatarUrl ? (
            <img
              src={profile.effectiveAvatarUrl}
              alt=""
              className="w-24 h-24 rounded-full object-cover border-2 border-slate-600"
            />
          ) : (
            <div className="w-24 h-24 rounded-full bg-slate-700 flex items-center justify-center border-2 border-slate-600">
              <svg className="w-12 h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
            </div>
          )}

          {/* Info */}
          <div>
            <h1 className="text-2xl font-bold text-white">{getDisplayName()}</h1>
            {getSecondaryName() && (
              <p className="text-slate-400">{getSecondaryName()}</p>
            )}
            <p className="text-sm text-slate-500 mt-2">
              Member since {new Date(profile.createdAt).toLocaleDateString()}
            </p>
          </div>
        </div>
      </div>

      {/* Aircraft Section */}
      <div className="bg-slate-800 rounded-lg p-6 max-w-2xl">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Aircraft</h2>
          <span className="text-sm text-slate-400">
            {profile.aircraft.length} aircraft
          </span>
        </div>

        {profile.aircraft.length === 0 ? (
          <div className="text-center py-8 text-slate-500">
            <svg className="w-12 h-12 mx-auto mb-3 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
            <p>No aircraft to display</p>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {profile.aircraft.map((aircraft) => (
              <AircraftCard key={aircraft.id} aircraft={aircraft} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function AircraftCard({ aircraft }: { aircraft: AircraftPublic }) {
  const formatType = (type?: string) => {
    if (!type) return 'Unknown Type';
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  return (
    <div className="bg-slate-900 rounded-lg overflow-hidden border border-slate-700">
      {/* Image */}
      <div className="aspect-video bg-slate-800 flex items-center justify-center">
        <svg className="w-12 h-12 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
        </svg>
      </div>

      {/* Info */}
      <div className="p-4">
        <h3 className="font-medium text-white truncate">{aircraft.name}</h3>
        {aircraft.nickname && (
          <p className="text-sm text-slate-400 truncate mt-0.5">"{aircraft.nickname}"</p>
        )}
        {aircraft.type && (
          <div className="mt-2">
            <span className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300">
              {formatType(aircraft.type)}
            </span>
          </div>
        )}
        {aircraft.description && (
          <p className="text-sm text-slate-500 mt-2 line-clamp-2">{aircraft.description}</p>
        )}
      </div>
    </div>
  );
}
