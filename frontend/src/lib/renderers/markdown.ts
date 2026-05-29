// Markdown renderer: placeholder. The plan is to back this with either a
// JSON-tree walker (mirror of ingest.MarkdownToDoc in reverse) or an
// off-screen TipTap+tiptap-markdown instance that uses
// `editor.storage.markdown.getMarkdown()`. Wiring it requires settling on
// markdown serialization choices (e.g. `**bold**` vs `__bold__`, dash vs
// asterisk bullets); deferring until an export/copy-as-markdown surface
// actually needs it.

export function renderMarkdown(_doc: unknown): string {
  throw new Error("markdown renderer not implemented yet (deferred until export flow lands)");
}
