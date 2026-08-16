import * as React from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { config } from "../../wailsjs/go/models";

// Owns vault list/active-vault/switcher-open state plus the switch flow.
// Extracted verbatim from App.tsx's Inner() — the only external dependency
// is `refreshInboxCount` (passed in), and `runSearch` is threaded back the
// same way the original code did it: a ref the caller assigns on every
// render (`runSearchRef.current = runSearch`) to dodge the forward-reference
// TDZ error, since `runSearch` is declared after vault state in Inner.
export function useVaultState(refreshInboxCount: () => void) {
  const [vaults, setVaults] = React.useState<config.Vault[]>([]);
  const [activeVault, setActiveVault] = React.useState<config.Vault | null>(null);
  const [showVaultSwitcher, setShowVaultSwitcher] = React.useState(false);

  const loadVaults = React.useCallback(async () => {
    try {
      const [allVaults, active] = await Promise.all([
        Backend.GetVaults(),
        Backend.GetActiveVault(),
      ]);
      setVaults(allVaults ?? []);
      setActiveVault(active ?? null);
    } catch (err) {
      console.error('Failed to load vaults:', err);
    }
  }, []);

  // Ref so handleVaultSwitch can call the latest runSearch without a forward-reference TDZ error
  const runSearchRef = React.useRef<() => void>(() => {});

  const handleVaultSwitch = React.useCallback(async (vault: config.Vault) => {
    setShowVaultSwitcher(false);
    try {
      await Backend.SwitchVault(vault.id);
      setActiveVault(vault);
      runSearchRef.current();
      refreshInboxCount();
    } catch (err) {
      console.error('Failed to switch vault:', err);
    }
  }, [refreshInboxCount]);

  return {
    vaults,
    activeVault,
    showVaultSwitcher,
    setShowVaultSwitcher,
    loadVaults,
    handleVaultSwitch,
    runSearchRef,
  };
}
