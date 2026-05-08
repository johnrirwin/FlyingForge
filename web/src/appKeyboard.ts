function resolveTargetElement(target: EventTarget | null): Element | null {
  if (target instanceof Element) {
    return target;
  }
  if (target instanceof Node) {
    return target.parentElement;
  }
  return null;
}

export function isEditableKeyboardTarget(target: EventTarget | null): boolean {
  const element = resolveTargetElement(target);
  if (!element) {
    return false;
  }

  if (element.closest('input, textarea, select')) {
    return true;
  }

  let current: Element | null = element;
  while (current) {
    if (current instanceof HTMLElement && current.isContentEditable) {
      return true;
    }
    current = current.parentElement;
  }

  return false;
}

export function hasOpenModalDialog(doc: Document = document): boolean {
  return doc.querySelector('[role="dialog"][aria-modal="true"]') !== null;
}

export function shouldHandleGlobalSlashShortcut(target: EventTarget | null, modalDialogOpen: boolean): boolean {
  if (modalDialogOpen) {
    return false;
  }

  return !isEditableKeyboardTarget(target);
}
