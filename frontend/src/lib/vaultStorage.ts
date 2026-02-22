/**
 * Vault-scoped localStorage helpers.
 *
 * Keys are namespaced as `nanite.vault.{vaultId}.{key}` so each vault
 * can carry its own tabs, sessions, and templates.
 */

function vaultKey(vaultId: string, key: string): string {
  return `nanite.vault.${vaultId}.${key}`;
}

export function getVaultSetting<T>(vaultId: string, key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(vaultKey(vaultId, key));
    if (raw === null) return fallback;
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

export function setVaultSetting<T>(vaultId: string, key: string, value: T): void {
  try {
    localStorage.setItem(vaultKey(vaultId, key), JSON.stringify(value));
  } catch {
    // localStorage full or unavailable — fail silently
  }
}

export function clearVaultSettings(vaultId: string): void {
  const prefix = vaultKey(vaultId, "");
  const toRemove: string[] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const k = localStorage.key(i);
    if (k && k.startsWith(prefix)) toRemove.push(k);
  }
  toRemove.forEach(k => localStorage.removeItem(k));
}
