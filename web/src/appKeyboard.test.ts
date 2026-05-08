import { describe, expect, it } from 'vitest';
import {
  hasOpenModalDialog,
  isEditableKeyboardTarget,
  shouldHandleGlobalSlashShortcut,
} from './appKeyboard';

describe('appKeyboard', () => {
  it('treats the Call to Action link input as an editable keyboard target', () => {
    const input = document.createElement('input');
    input.setAttribute('aria-label', 'Call to Action Link');

    expect(isEditableKeyboardTarget(input)).toBe(true);
    expect(shouldHandleGlobalSlashShortcut(input, false)).toBe(false);
  });

  it('does not allow the global slash shortcut while a modal dialog is open', () => {
    const dialog = document.createElement('div');
    dialog.setAttribute('role', 'dialog');
    dialog.setAttribute('aria-modal', 'true');
    document.body.appendChild(dialog);

    const target = document.createElement('div');

    expect(hasOpenModalDialog(document)).toBe(true);
    expect(shouldHandleGlobalSlashShortcut(target, hasOpenModalDialog(document))).toBe(false);

    dialog.remove();
  });

  it('allows the global slash shortcut when no modal is open and the target is not editable', () => {
    const target = document.createElement('div');

    expect(hasOpenModalDialog(document)).toBe(false);
    expect(shouldHandleGlobalSlashShortcut(target, false)).toBe(true);
  });
});
