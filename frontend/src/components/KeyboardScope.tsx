import * as React from "react";

type Props = {
  onQuickAdd: () => void;
};

export default function KeyboardScope({ onQuickAdd }: Props) {
  React.useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "n") {
        e.preventDefault();
        onQuickAdd();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onQuickAdd]);

  return null;
}
