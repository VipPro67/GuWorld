import { useCallback, useEffect, useLayoutEffect, useRef, type KeyboardEvent, type RefObject } from "react";

let activeDialogCount = 0;
let bodyOverflowBeforeDialog = "";
const focusableSelector = [
  "button:not(:disabled)",
  "a[href]",
  "input:not(:disabled):not([type=hidden])",
  "select:not(:disabled)",
  "textarea:not(:disabled)",
  "[contenteditable=true]",
  "summary",
  '[tabindex]:not([tabindex="-1"])',
].join(", ");

function getFocusableElements(dialog: HTMLElement): HTMLElement[] {
  return Array.from(dialog.querySelectorAll<HTMLElement>(focusableSelector)).filter((element) => {
    const style = window.getComputedStyle(element);
    return element.getClientRects().length > 0
      && style.visibility !== "hidden"
      && !element.closest('[hidden], [inert], [aria-hidden="true"]');
  });
}

/** Adds focus entry/return, Tab containment, Escape handling and body scroll lock to a mounted dialog. */
export function useAccessibleDialog<T extends HTMLElement>(onClose: () => void, enabled = true, activationKey?: unknown): {
  dialogRef: RefObject<T | null>;
  onKeyDown: (event: KeyboardEvent<T>) => void;
} {
  const dialogRef = useRef<T>(null);
  const onCloseRef = useRef(onClose);
  useLayoutEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    if (!enabled) return;
    const dialog = dialogRef.current;
    if (!dialog) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    if (activeDialogCount === 0) bodyOverflowBeforeDialog = document.body.style.overflow;
    activeDialogCount += 1;
    document.body.style.overflow = "hidden";
    const frame = requestAnimationFrame(() => {
      const firstControl = getFocusableElements(dialog)[0];
      (firstControl || dialog).focus();
    });
    return () => {
      cancelAnimationFrame(frame);
      activeDialogCount = Math.max(0, activeDialogCount - 1);
      if (activeDialogCount === 0) document.body.style.overflow = bodyOverflowBeforeDialog;
      if (previousFocus?.isConnected) previousFocus.focus();
    };
  }, [enabled, activationKey]);

  const onKeyDown = useCallback((event: KeyboardEvent<T>) => {
    if (event.key === "Escape") {
      event.stopPropagation();
      onCloseRef.current();
      return;
    }
    if (event.key !== "Tab") return;
    const controls = dialogRef.current ? getFocusableElements(dialogRef.current) : [];
    if (!controls.length) {
      event.preventDefault();
      dialogRef.current?.focus();
      return;
    }
    const first = controls[0];
    const last = controls.at(-1)!;
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }, []);

  return { dialogRef, onKeyDown };
}
