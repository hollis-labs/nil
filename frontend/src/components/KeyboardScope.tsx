import * as React from "react";

type Props = {
  onQuickAdd: () => void;
  onEscape?: () => void;
};

export default function KeyboardScope({ onQuickAdd, onEscape }: Props) {
  React.useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
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
  }, [onQuickAdd, onEscape]);

  return null;
}
