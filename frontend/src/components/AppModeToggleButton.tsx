import { CheckSquare, FileText, Layers } from "lucide-react";
import * as React from "react";

// Cycles todos -> notes -> all on click; long-press opens Settings on the
// "tabs" tab. Extracted verbatim from App.tsx's Inner() JSX — the long-press
// timer refs (lpTimer/lpFired) were only ever read/written by this button,
// so they move here as genuinely local UI state rather than being
// prop-drilled from the parent.
interface AppModeToggleButtonProps {
  appMode: 'todos' | 'notes' | 'all' | 'inbox';
  onCycle: (next: 'todos' | 'notes' | 'all') => void;
  onLongPress: () => void;
}

export default function AppModeToggleButton({ appMode, onCycle, onLongPress }: AppModeToggleButtonProps) {
  const lpTimer = React.useRef<NodeJS.Timeout | null>(null);
  const lpFired = React.useRef(false);

  return (
    <button
      className="badge info"
      draggable
      onDragStart={(e) => e.preventDefault()}
      onContextMenu={(e) => e.preventDefault()}
      onMouseDown={() => {
        lpFired.current = false;
        lpTimer.current = setTimeout(() => {
          lpFired.current = true;
          onLongPress();
        }, 1000);
      }}
      onMouseUp={() => {
        if (lpTimer.current) {
          clearTimeout(lpTimer.current);
          lpTimer.current = null;
        }
      }}
      onMouseLeave={() => {
        if (lpTimer.current) {
          clearTimeout(lpTimer.current);
          lpTimer.current = null;
        }
      }}
      onClick={() => {
        if (!lpFired.current) {
          // 3-cycle: todos → notes → all → todos
          const next: 'todos' | 'notes' | 'all' = appMode === 'todos' ? 'notes'
                             : appMode === 'notes' ? 'all'
                             : 'todos';
          onCycle(next);
        }
        lpFired.current = false;
      }}
      style={{
        padding: '8px 12px',
        fontSize: '13px',
        borderRadius: '6px',
        border: 'none',
        display: 'flex',
        alignItems: 'center',
        gap: '4px',
        cursor: 'pointer',
        userSelect: 'none'
      }}
      title={
        appMode === 'todos' ? 'Switch to Notes (long-press for tab settings)'
        : appMode === 'notes' ? 'Switch to All (long-press for tab settings)'
        : 'Switch to Items (long-press for tab settings)'
      }
    >
      {appMode === 'todos' ? (
        <>
          <CheckSquare size={12} />
          Items
        </>
      ) : appMode === 'notes' ? (
        <>
          <FileText size={12} />
          Notes
        </>
      ) : (
        <>
          <Layers size={12} />
          All
        </>
      )}
    </button>
  );
}
