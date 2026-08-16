import * as React from "react";
import * as Backend from "../../wailsjs/go/main/App";

// Owns the inbox badge count and its refresh logic. Extracted verbatim from
// App.tsx's Inner() — genuinely disjoint from search/appMode state, so no
// prop-drilling of unrelated state was needed to pull this out.
export function useInboxCount() {
  const [inboxCount, setInboxCount] = React.useState(0);

  const refreshInboxCount = React.useCallback(async () => {
    try {
      const count = await Backend.GetInboxCount();
      setInboxCount(count);
    } catch (err) {
      console.error('Failed to get inbox count:', err);
    }
  }, []);

  return { inboxCount, setInboxCount, refreshInboxCount };
}
