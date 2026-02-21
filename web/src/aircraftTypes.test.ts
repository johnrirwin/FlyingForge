import { describe, expect, it } from 'vitest';
import { COMPONENT_CATEGORIES } from './aircraftTypes';

describe('aircraft component categories', () => {
  it('includes FC/ESC stack as an assignable component', () => {
    expect(COMPONENT_CATEGORIES).toContainEqual({
      value: 'stack',
      label: 'FC/ESC Stack',
      equipmentCategory: 'stacks',
    });
  });
});
