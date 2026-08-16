import * as Backend from "../../wailsjs/go/main/App";
import { store } from "../../wailsjs/go/models";

// Typed wrappers around the Wails-generated bindings.
//
// The generated bindings require full store.SearchRequest / store.Item
// shapes. Call sites historically used `as any` casts to pass partials, which
// silently allowed field-name typos like `type:` instead of `kind:` (the v1.3.2
// silent-search-bug class). These helpers accept Partial<> inputs so the
// TypeScript compiler catches typos and unknown fields at the boundary.

const searchDefaults: store.SearchRequest = {
  query: "",
  projects: [],
  contexts: [],
  tags: [],
  statuses: [],
  priorities: [],
  page: 0,
  page_size: 50,
  sort_by: "created_at",
  sort_dir: "desc",
  kind: "all",
  include_inbox: false,
  updated_since: "",
};

export function search(partial: Partial<store.SearchRequest>): Promise<store.Item[]> {
  return Backend.Search({ ...searchDefaults, ...partial });
}

export function getInboxItems(
  partial: Partial<store.SearchRequest> = {},
): Promise<store.Item[]> {
  return Backend.GetInboxItems({
    ...searchDefaults,
    page_size: 200,
    include_inbox: true,
    ...partial,
  });
}

// Backend.UpdateItem is a full-replace update on the Go side, not a patch —
// an object missing a field (e.g. `tags`) blanks that field server-side
// rather than leaving it untouched. Requiring every field at the type level
// turns out to be impractical here: ItemRow (the shape most call sites
// spread from) legitimately under-declares several fields as optional for
// UI-rendering purposes, so "every key required" rejects call sites that are
// actually safe at runtime (they spread a full store.Item) and there's no
// clean way to distinguish those from a genuinely partial caller at the type
// level alone.
//
// Instead, make partial calls safe by construction: fetch the current item
// and merge the caller's fields onto it before writing, so an omitted field
// is preserved rather than blanked, regardless of what the caller passes.
// This mirrors service/items.Service.UpdatePatch's read-modify-write pattern
// on the Go side. Costs one extra round-trip per save — acceptable for a
// user-driven edit action, not a hot path.
export type ItemUpdate = { id: number } & {
  [K in keyof Omit<store.Item, "id">]?: store.Item[K] | undefined;
};

export async function updateItem(update: ItemUpdate): Promise<void> {
  const existing = await Backend.GetItem(update.id);
  await Backend.UpdateItem({ ...existing, ...update } as store.Item);
}
