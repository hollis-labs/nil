// YAML frontmatter parser/serializer for markdown import/export. Treats
// frontmatter as a translation layer between flat markdown documents and
// NIL's structured columns (kind, tags, projects, contexts, priority,
// due_at, etc.) — not as a storage concern. The body is what gets fed to
// the editor; metadata maps to columns on the item.
//
// Intentionally light: supports flat string / string[] / number / boolean
// values which is what the NIL columns need. Nested objects are not
// supported and will be skipped with a console warning. If we ever ship
// import/export for arbitrary YAML, swap in a proper parser (js-yaml).

const FRONTMATTER_RE = /^---\s*\n([\s\S]*?)\n---\s*\n?/;

export type Frontmatter = Record<string, string | number | boolean | string[]>;

export function parseFrontmatter(markdown: string): { metadata: Frontmatter; body: string } {
  const m = markdown.match(FRONTMATTER_RE);
  if (!m) return { metadata: {}, body: markdown };
  const yaml = m[1] ?? "";
  const body = markdown.slice(m[0].length);
  return { metadata: parseFlatYAML(yaml), body };
}

export function serializeFrontmatter(metadata: Frontmatter, body: string): string {
  const keys = Object.keys(metadata).filter((k) => metadata[k] !== undefined);
  if (keys.length === 0) return body;
  const lines: string[] = ["---"];
  for (const k of keys) {
    const v = metadata[k];
    if (Array.isArray(v)) {
      lines.push(`${k}: [${v.map(quoteIfNeeded).join(", ")}]`);
    } else if (typeof v === "boolean" || typeof v === "number") {
      lines.push(`${k}: ${v}`);
    } else {
      lines.push(`${k}: ${quoteIfNeeded(String(v))}`);
    }
  }
  lines.push("---");
  return lines.join("\n") + "\n" + body;
}

// Minimal flat YAML: `key: value` per line, supports strings, numbers,
// booleans, and inline arrays `[a, b, c]`. Comments (#) and indentation
// are not supported.
function parseFlatYAML(yaml: string): Frontmatter {
  const out: Frontmatter = {};
  for (const rawLine of yaml.split("\n")) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const colon = line.indexOf(":");
    if (colon < 0) continue;
    const key = line.slice(0, colon).trim();
    const valStr = line.slice(colon + 1).trim();
    out[key] = parseScalar(valStr);
  }
  return out;
}

function parseScalar(s: string): string | number | boolean | string[] {
  if (s.startsWith("[") && s.endsWith("]")) {
    const inner = s.slice(1, -1);
    if (!inner.trim()) return [];
    return inner.split(",").map((p) => unquote(p.trim()));
  }
  if (s === "true") return true;
  if (s === "false") return false;
  if (s !== "" && !isNaN(Number(s)) && /^-?\d+(\.\d+)?$/.test(s)) return Number(s);
  return unquote(s);
}

function unquote(s: string): string {
  if ((s.startsWith('"') && s.endsWith('"')) || (s.startsWith("'") && s.endsWith("'"))) {
    return s.slice(1, -1);
  }
  return s;
}

function quoteIfNeeded(s: string): string {
  if (/[,:\[\]"'#]|^\s|\s$/.test(s)) {
    return JSON.stringify(s);
  }
  return s;
}
