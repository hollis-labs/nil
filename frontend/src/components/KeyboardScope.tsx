import * as React from "react";

type Props = {
  onQuickAdd: () => void;
  onEscape?: () => void;
  onQuickAddNote?: () => void;
  onToggleAppMode?: () => void;
  onOpenInbox?: () => void;
};

export default function KeyboardScope({ onQuickAdd, onEscape, onQuickAddNote, onToggleAppMode, onOpenInbox }: Props) {
  React.useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
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
  }, [onQuickAdd, onEscape, onQuickAddNote, onToggleAppMode, onOpenInbox]);

  return null;
}
