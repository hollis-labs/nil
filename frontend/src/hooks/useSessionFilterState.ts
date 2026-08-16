import * as React from "react";
import { clearActiveSession, getActiveSession, setActiveSession } from "@/lib/sessionContext";

// Owns the session-as-filter badge state (sessionFilterCount / sessionAsFilter)
// plus the mount-time session validation/cleanup. Extracted verbatim from
// App.tsx's Inner(). Business logic that actually *applies* the session
// filter (runSearch, handleInputSubmit) re-reads getActiveSession() directly
// rather than depending on this hook's derived state, so this slice is
// display-only and genuinely disjoint from the search/appMode core.
export function useSessionFilterState() {
  const [sessionFilterCount, setSessionFilterCount] = React.useState(0);
  const [sessionAsFilter, setSessionAsFilter] = React.useState(false);

  const updateSessionFilterCount = React.useCallback(() => {
    const session = getActiveSession();
    if (!session) {
      setSessionFilterCount(0);
      setSessionAsFilter(false);
      return;
    }
    const count =
      (session.contexts?.length || 0) +
      (session.projects?.length || 0) +
      (session.tags?.length || 0) +
      (session.priority ? 1 : 0);

    // If session has useAsFilterTab enabled but no actual filters, clear it
    if (session.useAsFilterTab && count === 0) {
      setActiveSession({ ...session, useAsFilterTab: false });
      setSessionFilterCount(0);
      setSessionAsFilter(false);
      return;
    }

    setSessionFilterCount(count);
    setSessionAsFilter(session.useAsFilterTab || false);
  }, []);

  React.useEffect(() => {
    // Validate and clean session on mount
    const session = getActiveSession();
    console.log('[App Mount] Session on startup:', session);
    if (session) {
      const count =
        (session.contexts?.length || 0) +
        (session.projects?.length || 0) +
        (session.tags?.length || 0) +
        (session.priority ? 1 : 0);

      console.log('[App Mount] Session filter count:', count, 'useAsFilterTab:', session.useAsFilterTab);

      // If session has useAsFilterTab enabled but no actual filters, disable it immediately
      if (session.useAsFilterTab && count === 0) {
        console.log('[App Mount] Disabling empty useAsFilterTab');
        setActiveSession({ ...session, useAsFilterTab: false });
      }

      // If session exists but has empty arrays, clear it entirely
      if (count === 0 && !session.useAsFilterTab) {
        console.log('[App Mount] Clearing empty session');
        clearActiveSession();
      }
    }
    updateSessionFilterCount();
  }, [updateSessionFilterCount]);

  return { sessionFilterCount, sessionAsFilter, updateSessionFilterCount };
}
