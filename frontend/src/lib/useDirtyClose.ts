import * as React from "react";

export type CloseBehaviorSetting = 'ask' | 'always' | 'never';

// Snapshot of the fields EditItemModal treats as "content" for dirty-checking
// purposes. notes_doc is the stringified TipTap/ProseMirror JSON — comparison
// is by JSON equality, which is stable because TipTap normalizes its own doc
// shape on setContent()/getJSON() round-trips.
export type EditItemFieldsSnapshot = {
  line: string;
  priority: string;
  due: string;
  tags: string[];
  contexts: string[];
  projects: string[];
  notes_doc: string;
};

// Dirty-detection + close-behavior logic for EditItemModal, lifted out as-is.
// Escape / the header "Close" button / the footer "Cancel" button all route
// through requestClose(), which checks isDirty() against the baseline
// captured via captureInitial() and then branches on the closeBehavior
// setting ('ask' shows the close prompt, 'always' saves and closes, 'never'
// closes without saving).
export function useDirtyClose(params: {
  // Recomputed fresh on every call so isDirty() always compares against the
  // latest field values, not a value captured at some earlier render.
  getCurrentSnapshot: () => EditItemFieldsSnapshot;
  closeBehavior: CloseBehaviorSetting;
  onOpenChange: (open: boolean) => void;
  onSaveAndClose: () => void;
  onRememberBehavior: (behavior: CloseBehaviorSetting) => void;
}) {
  const { getCurrentSnapshot, closeBehavior, onOpenChange, onSaveAndClose, onRememberBehavior } = params;

  const initialValuesRef = React.useRef<EditItemFieldsSnapshot | null>(null);
  const [showClosePrompt, setShowClosePrompt] = React.useState(false);
  const [pendingBehavior, setPendingBehavior] = React.useState<CloseBehaviorSetting>('ask');

  // Stable identity (like a useState setter) since it only touches the ref —
  // lets callers list it in their own effect deps without it forcing a
  // re-run on every render, and matches how the plain setState calls it
  // replaced behaved before this extraction.
  const captureInitial = React.useCallback((snapshot: EditItemFieldsSnapshot) => {
    initialValuesRef.current = snapshot;
  }, []);

  function isDirty(): boolean {
    const initial = initialValuesRef.current;
    if (!initial) return false;
    const current = getCurrentSnapshot();
    const arrSame = (a: string[], b: string[]) =>
      JSON.stringify([...a].sort()) === JSON.stringify([...b].sort());
    return (
      current.line !== initial.line ||
      current.priority !== initial.priority ||
      current.due !== initial.due ||
      !arrSame(current.tags, initial.tags) ||
      !arrSame(current.contexts, initial.contexts) ||
      !arrSame(current.projects, initial.projects) ||
      current.notes_doc !== initial.notes_doc
    );
  }

  function requestClose() {
    if (!isDirty()) { onOpenChange(false); return; }
    if (closeBehavior === 'always') { onSaveAndClose(); return; }
    if (closeBehavior === 'never') { onOpenChange(false); return; }
    setPendingBehavior(closeBehavior);
    setShowClosePrompt(true);
  }

  function confirmDiscard() {
    if (pendingBehavior !== closeBehavior) onRememberBehavior(pendingBehavior);
    setShowClosePrompt(false);
    onOpenChange(false);
  }

  function confirmSaveAndClose() {
    if (pendingBehavior !== closeBehavior) onRememberBehavior(pendingBehavior);
    setShowClosePrompt(false);
    onSaveAndClose();
  }

  // Stable identity for the same reason as captureInitial above.
  const cancelClosePrompt = React.useCallback(() => {
    setShowClosePrompt(false);
  }, []);

  return {
    captureInitial,
    isDirty,
    requestClose,
    showClosePrompt,
    pendingBehavior,
    setPendingBehavior,
    confirmDiscard,
    confirmSaveAndClose,
    cancelClosePrompt,
  };
}
