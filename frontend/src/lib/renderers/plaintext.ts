// Plain-text renderer: walks the PM JSON tree and concatenates text content
// with paragraph breaks between block-level nodes. Mirrors the Go-side
// `ingest.DocToPlainText` so client and server agree on what counts as
// indexable / AI-context-worthy text.

type Node = {
  type: string;
  text?: string;
  content?: Node[];
  attrs?: Record<string, unknown>;
};

const BLOCK_TYPES = new Set([
  "paragraph", "heading", "blockquote", "codeBlock",
  "bulletList", "orderedList", "listItem", "horizontalRule",
]);

export function renderPlainText(doc: unknown): string {
  const root = (doc as Node) ?? { type: "doc" };
  const buf: string[] = [];
  walk(root, buf, true);
  let out = buf.join("").trim();
  while (out.includes("\n\n\n")) {
    out = out.replace(/\n\n\n/g, "\n\n");
  }
  return out;
}

function walk(n: Node, buf: string[], isRoot: boolean): void {
  switch (n.type) {
    case "text":
      buf.push(n.text ?? "");
      return;
    case "hardBreak":
      buf.push("\n");
      return;
    case "wikilink": {
      const label = (n.attrs?.["label"] as string) ?? "";
      if (label) buf.push(label);
      return;
    }
    case "horizontalRule":
      buf.push("\n");
      return;
    default:
      for (const c of n.content ?? []) {
        walk(c, buf, false);
      }
      if (!isRoot && BLOCK_TYPES.has(n.type)) {
        buf.push("\n\n");
      }
  }
}
