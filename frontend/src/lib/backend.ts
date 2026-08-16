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

// An update must carry id; everything else is optional. Optional fields are
// allowed to be explicitly `undefined` so callers can spread Wails-generated
// objects (which carry `undefined` for absent optionals) without fighting
// exactOptionalPropertyTypes. Excess properties are still rejected, which is
// the typo-catching guarantee these helpers exist for.
export type ItemUpdate = { id: number } & {
  [K in keyof Omit<store.Item, "id">]?: store.Item[K] | undefined;
};

export function updateItem(update: ItemUpdate): Promise<void> {
  return Backend.UpdateItem(update as store.Item);
}
