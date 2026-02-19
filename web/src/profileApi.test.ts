import { describe, expect, it, vi } from 'vitest';

vi.mock('bad-words-next', () => ({
  default: class {
    check(value: string): boolean {
      return value.toLowerCase().includes('dronecurse');
    }
  },
}));

vi.mock('bad-words-next/lib/en', () => ({
  default: { id: 'en', words: [], lookalike: {} },
}));

import { validateCallSign } from './profileApi';

describe('validateCallSign', () => {
  it('allows a valid callsign', () => {
    expect(validateCallSign('Pilot_One-7')).toBeNull();
  });

  it('blocks terms detected by the profanity library', () => {
    expect(validateCallSign('fpv_dronecurse')).toBe('Callsign contains inappropriate language');
  });

  it('allows non-matching callsigns', () => {
    expect(validateCallSign('AcePilot')).toBeNull();
  });

  it('keeps existing format validation behavior', () => {
    expect(validateCallSign('Pilot One')).toBe('Callsign can only contain letters, numbers, underscores, and hyphens');
  });
});
