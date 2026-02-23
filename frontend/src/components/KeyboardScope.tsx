import * as React from "react";

type Props = {
  onQuickAdd: () => void;
  onEscape?: () => void;
  onQuickAddNote?: () => void;
  onToggleAppMode?: () => void;
  onOpenInbox?: () => void;
  onOpenQuickSearch?: () => void;
};

export default function KeyboardScope({ onQuickAdd, onEscape, onQuickAddNote, onToggleAppMode, onOpenInbox, onOpenQuickSearch }: Props) {
  const lastShiftRef = React.useRef<number>(0);

  React.useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Any non-Shift key resets the double-tap window so normal typing never triggers it.
      if (e.key !== 'Shift') {
        lastShiftRef.current = 0;
      }

      // Shift+Shift (double-tap within 300ms) → Quick Search
      if (e.key === 'Shift' && !e.metaKey && !e.ctrlKey && !e.altKey) {
        const now = Date.now();
        if (now - lastShiftRef.current < 300) {
          e.preventDefault();
          onOpenQuickSearch?.();
          lastShiftRef.current = 0;
        } else {
          lastShiftRef.current = now;
        }
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === "n") {
        e.preventDefault();
        onQuickAddNote?.();
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === "m") {
        e.preventDefault();
        onToggleAppMode?.();
        return;
      }
      if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === "s") {
        e.preventDefault();
        onOpenQuickSearch?.();
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key === "i") {
        e.preventDefault();
        onOpenInbox?.();
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key === "n") {
        e.preventDefault();
        onQuickAdd();
      }
      if (e.key === "Escape" && onEscape) {
        e.preventDefault();
        onEscape();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onQuickAdd, onEscape, onQuickAddNote, onToggleAppMode, onOpenInbox, onOpenQuickSearch]);

  return null;
}
