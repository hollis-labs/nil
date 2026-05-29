// HTML renderer: the canonical HTML for a doc is what TipTap produces, which
// is exactly what gets written to the notes_html cache at save time. For
// cached rendering, prefer reading `item.notes_html` directly. This function
// exists for the renderer-registry interface — when called with just a doc
// it falls back to a synchronous, simplified walk that's good enough for
// previews but is NOT a substitute for the TipTap-produced cache.
//
// If we ever need byte-identical TipTap HTML at render time (rather than
// from the cache), spin up an off-screen editor in the caller — that's the
// only way to guarantee parity. Keeping that out of here so this renderer
// stays cheap and synchronous.

type Node = {
  type: string;
  text?: string;
  content?: Node[];
  attrs?: Record<string, unknown>;
  marks?: Array<{ type: string; attrs?: Record<string, unknown> }>;
};

export function renderHTML(doc: unknown): string {
  const root = (doc as Node) ?? { type: "doc" };
  const parts: string[] = [];
  for (const c of root.content ?? []) {
    writeBlock(c, parts);
  }
  return parts.join("");
}

function writeBlock(n: Node, out: string[]): void {
  switch (n.type) {
    case "paragraph":
      out.push("<p>");
      writeInline(n.content ?? [], out);
      out.push("</p>");
      return;
    case "heading": {
      const level = clampHeadingLevel(n.attrs?.["level"]);
      out.push(`<h${level}>`);
      writeInline(n.content ?? [], out);
      out.push(`</h${level}>`);
      return;
    }
    case "bulletList":
      out.push("<ul>");
      for (const c of n.content ?? []) writeBlock(c, out);
      out.push("</ul>");
      return;
    case "orderedList": {
      const start = n.attrs?.["start"];
      out.push(start != null ? `<ol start="${escapeAttr(String(start))}">` : "<ol>");
      for (const c of n.content ?? []) writeBlock(c, out);
      out.push("</ol>");
      return;
    }
    case "listItem":
      out.push("<li>");
      for (const c of n.content ?? []) writeBlock(c, out);
      out.push("</li>");
      return;
    case "blockquote":
      out.push("<blockquote>");
      for (const c of n.content ?? []) writeBlock(c, out);
      out.push("</blockquote>");
      return;
    case "codeBlock": {
      const lang = n.attrs?.["language"] as string | undefined;
      out.push(lang ? `<pre><code class="language-${escapeAttr(lang)}">` : "<pre><code>");
      for (const c of n.content ?? []) {
        if (c.type === "text") out.push(escapeText(c.text ?? ""));
      }
      out.push("</code></pre>");
      return;
    }
    case "horizontalRule":
      out.push("<hr>");
      return;
    default:
      for (const c of n.content ?? []) writeBlock(c, out);
  }
}

function writeInline(nodes: Node[], out: string[]): void {
  for (const n of nodes) writeInlineNode(n, out);
}

function writeInlineNode(n: Node, out: string[]): void {
  if (n.type === "text") {
    const { open, close } = renderMarks(n.marks ?? []);
    out.push(open);
    out.push(escapeText(n.text ?? ""));
    out.push(close);
    return;
  }
  if (n.type === "hardBreak") {
    out.push("<br>");
    return;
  }
  if (n.type === "wikilink") {
    const id = String(n.attrs?.["id"] ?? "");
    const refType = String(n.attrs?.["refType"] ?? "");
    const label = String(n.attrs?.["label"] ?? "");
    out.push(
      `<span data-type="wikilink" data-id="${escapeAttr(id)}" ` +
      `data-ref-type="${escapeAttr(refType)}" data-label="${escapeAttr(label)}">` +
      `${escapeText(label)}</span>`
    );
    return;
  }
  for (const c of n.content ?? []) writeInlineNode(c, out);
}

function renderMarks(marks: Array<{ type: string; attrs?: Record<string, unknown> }>): { open: string; close: string } {
  if (marks.length === 0) return { open: "", close: "" };
  const open: string[] = [];
  const close: string[] = [];
  for (const m of marks) {
    switch (m.type) {
      case "bold":   open.push("<strong>"); close.unshift("</strong>"); break;
      case "italic": open.push("<em>");     close.unshift("</em>");     break;
      case "code":   open.push("<code>");   close.unshift("</code>");   break;
      case "strike": open.push("<s>");      close.unshift("</s>");      break;
      case "link": {
        const href = String(m.attrs?.["href"] ?? "");
        open.push(`<a href="${escapeAttr(href)}">`);
        close.unshift("</a>");
        break;
      }
    }
  }
  return { open: open.join(""), close: close.join("") };
}

function clampHeadingLevel(v: unknown): number {
  const n = typeof v === "number" ? v : 1;
  if (n < 1 || n > 6) return 1;
  return Math.trunc(n);
}

function escapeText(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;");
}
