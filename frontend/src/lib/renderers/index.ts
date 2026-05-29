// Output renderer registry: small, plugin-ready surface for converting the
// canonical PM JSON doc into other formats. Built-ins shipped: plain-text
// (used for search snippets and AI context) and HTML (defers to the
// pre-rendered notes_html cache when available). Markdown renderer is a
// placeholder until the export flow needs it.
//
// Future Hollis Labs plugin SDK plugs into the same `register()` call.

export type Renderer = (doc: unknown) => string;

const registry: Record<string, Renderer> = {};

export function register(name: string, fn: Renderer): void {
  registry[name] = fn;
}

export function render(name: string, doc: unknown): string {
  const fn = registry[name];
  if (!fn) throw new Error(`renderer "${name}" is not registered`);
  return fn(doc);
}

export function list(): string[] {
  return Object.keys(registry).sort();
}

// Bootstrap built-ins on import.
import { renderPlainText } from "./plaintext";
import { renderHTML } from "./html";
import { renderMarkdown } from "./markdown";

register("plaintext", renderPlainText);
register("html", renderHTML);
register("markdown", renderMarkdown);
