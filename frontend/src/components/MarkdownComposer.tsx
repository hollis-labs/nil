import { useEffect, useRef, useState } from "react";
import type { capture } from "../../wailsjs/go/models";
import { displaySource, reconcileInput, sourceOffset } from "../lib/markdownSource";

type Props = {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  searchReferences: (query: string) => Promise<capture.Suggestion[]>;
  onSubmit: () => void;
};

function escapeLabel(label: string) {
  return label
    .replace(/\\/g, "\\\\")
    .replace(/[[\]]/g, "\\$&")
    .replace(/[\r\n]/g, " ");
}

export default function MarkdownComposer({
  value,
  onChange,
  disabled,
  searchReferences,
  onSubmit,
}: Props) {
  const input = useRef<HTMLTextAreaElement>(null);
  const [cursor, setCursor] = useState(0);
  const [composing, setComposing] = useState(false);
  const [dismissed, setDismissed] = useState("");
  const [suggestions, setSuggestions] = useState<capture.Suggestion[]>([]);
  const [active, setActive] = useState(0);
  const [searchError, setSearchError] = useState("");
  const [searching, setSearching] = useState(false);
  const editGeneration = useRef(0);
  const [link, setLink] = useState<{
    start: number;
    end: number;
    label: string;
    url: string;
    source: string;
  } | null>(null);
  const [linkError, setLinkError] = useState("");
  const match = /(?:^|\s)@([^\s@]*)$/.exec(value.slice(0, cursor));
  const query = !composing && match ? (match[1] ?? "") : null;
  const start = query === null ? 0 : cursor - query.length - 1;
  const menuKey = query === null ? "" : `${start}:${query}`;
  const menuOpen = !!menuKey && menuKey !== dismissed && !link;

  useEffect(() => {
    setSuggestions([]);
    setActive(0);
    setSearchError("");
    if (!menuOpen || query === null) {
      setSearching(false);
      return;
    }
    setSearching(true);
    let cancelled = false;
    const timer = setTimeout(() => {
      searchReferences(query)
        .then((items) => {
          if (!cancelled) {
            setSuggestions(items.slice(0, 8));
            setSearching(false);
          }
        })
        .catch(() => {
          if (!cancelled) {
            setSearchError("Suggestions unavailable; plain text is kept.");
            setSearching(false);
          }
        });
    }, 120);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [menuOpen, query, searchReferences]);

  useEffect(() => {
    if (!disabled) input.current?.focus();
  }, [disabled]);

  function replace(
    from: number,
    to: number,
    text: string,
    selectionStart = from + text.length,
    selectionEnd = selectionStart,
  ) {
    const generation = ++editGeneration.current;
    const nextSource = value.slice(0, from) + text + value.slice(to);
    onChange(nextSource);
    setCursor(selectionEnd);
    requestAnimationFrame(() => {
      if (generation !== editGeneration.current) return;
      input.current?.focus();
      input.current?.setSelectionRange(
        displaySource(nextSource.slice(0, selectionStart)).length,
        displaySource(nextSource.slice(0, selectionEnd)).length,
      );
    });
  }

  function format(marker: string) {
    const el = input.current;
    if (!el) return;
    const from = sourceOffset(value, el.selectionStart),
      to = sourceOffset(value, el.selectionEnd);
    const selected = value.slice(from, to);
    replace(from, to, marker + selected + marker, from + marker.length, to + marker.length);
  }

  function openLink() {
    const el = input.current;
    if (!el) return;
    const from = sourceOffset(value, el.selectionStart),
      to = sourceOffset(value, el.selectionEnd);
    // Edit a supported inline link in place. Other Markdown is never parsed and
    // reserialized, so unsupported/nested syntax stays intact in the source.
    const links = value.matchAll(/\[([^\]\n]*)\]\(([^\s)]*)\)/g);
    for (const candidate of links) {
      const offset = candidate.index ?? 0;
      if (from >= offset && to <= offset + candidate[0].length) {
        setLink({
          start: offset,
          end: offset + candidate[0].length,
          label: candidate[1] ?? "",
          url: candidate[2] ?? "",
          source: value,
        });
        setLinkError("");
        return;
      }
    }
    setLink({ start: from, end: to, label: value.slice(from, to), url: "https://", source: value });
    setLinkError("");
  }

  function closeLink() {
    setLink(null);
    requestAnimationFrame(() => input.current?.focus());
  }

  function applyLink() {
    if (!link) return;
    if (value !== link.source) {
      setLinkError("Draft changed while editing this link. Cancel and select it again.");
      return;
    }
    try {
      const url = new URL(link.url);
      if (
        !["https:", "http:", "mailto:", "nil:"].includes(url.protocol) ||
        Array.from(link.url).some((c) => c.charCodeAt(0) <= 32 || c.charCodeAt(0) === 127)
      )
        throw new Error();
      replace(
        link.start,
        link.end,
        `[${escapeLabel(link.label || link.url)}](${link.url.replace(/[()]/g, (s) => (s === "(" ? "%28" : "%29"))})`,
      );
      setLink(null);
    } catch {
      setLinkError("Use an http, https, mailto or local Nil link.");
    }
  }

  function mention(item: capture.Suggestion) {
    replace(
      start,
      cursor,
      `[${escapeLabel(item.reference.label || item.reference.id)}](${item.url}) `,
    );
    setDismissed(menuKey);
    setSuggestions([]);
  }

  return (
    <div style={{ display: "flex", flex: 1, minHeight: 0, flexDirection: "column", gap: 8 }}>
      <div role="toolbar" aria-label="Markdown formatting" style={{ display: "flex", gap: 8 }}>
        <button type="button" disabled={disabled || !!link} onClick={() => format("**")}>
          Bold
        </button>
        <button type="button" disabled={disabled || !!link} onClick={() => format("*")}>
          Italic
        </button>
        <button type="button" disabled={disabled || !!link} onClick={() => format("`")}>
          Code
        </button>
        <button type="button" disabled={disabled || !!link} onClick={openLink}>
          Edit link
        </button>
      </div>
      <label htmlFor="capture-source">Markdown</label>
      <textarea
        id="capture-source"
        ref={input}
        value={displaySource(value)}
        disabled={disabled || !!link}
        aria-describedby="capture-editing-help"
        aria-controls={menuOpen ? "capture-mentions" : undefined}
        aria-activedescendant={
          menuOpen && suggestions[active] ? `capture-mention-${active}` : undefined
        }
        onSelect={(e) => setCursor(sourceOffset(value, e.currentTarget.selectionEnd))}
        onChange={(e) => {
          if (disabled || link) return;
          editGeneration.current++;
          const next = reconcileInput(value, e.target.value);
          setCursor(sourceOffset(next, e.currentTarget.selectionEnd));
          onChange(next);
        }}
        onCompositionStart={() => setComposing(true)}
        onCompositionEnd={() => setComposing(false)}
        onPaste={(e) => {
          e.preventDefault();
          if (disabled || link) return;
          const el = e.currentTarget;
          replace(
            sourceOffset(value, el.selectionStart),
            sourceOffset(value, el.selectionEnd),
            e.clipboardData.getData("text/plain"),
          );
        }}
        onKeyDown={(e) => {
          if (composing || e.nativeEvent.isComposing || e.keyCode === 229) return;
          if (menuOpen) {
            if (e.key === "Escape") {
              e.preventDefault();
              setDismissed(menuKey);
              return;
            }
            if (suggestions.length && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
              e.preventDefault();
              setActive(
                (i) =>
                  (i + (e.key === "ArrowDown" ? 1 : suggestions.length - 1)) % suggestions.length,
              );
              return;
            }
            const item = suggestions[active];
            if (item && (e.key === "Enter" || e.key === "Tab")) {
              e.preventDefault();
              mention(item);
              return;
            }
          }
          if (e.metaKey || e.ctrlKey) {
            if (e.key === "Enter") {
              e.preventDefault();
              onSubmit();
            } else if (e.key.toLowerCase() === "b") {
              e.preventDefault();
              format("**");
            } else if (e.key.toLowerCase() === "i") {
              e.preventDefault();
              format("*");
            } else if (e.key.toLowerCase() === "k") {
              e.preventDefault();
              openLink();
            }
          }
        }}
        style={{
          flex: 1,
          minHeight: 120,
          resize: "none",
          background: "var(--term-panel)",
          color: "var(--term-fg)",
          border: "1px solid var(--term-border)",
          padding: 12,
          fontFamily: "monospace",
        }}
      />
      {menuOpen && (
        <div>
          <div id="capture-mentions" role="listbox" aria-label="Local Nil references">
            {suggestions.map((item, i) => (
              <button
                type="button"
                role="option"
                aria-selected={i === active}
                id={`capture-mention-${i}`}
                key={item.url}
                onClick={() => mention(item)}
              >
                {item.reference.label || item.reference.id} · {item.reference.instance}
              </button>
            ))}
          </div>
          <p>
            {searchError ||
              (suggestions.length
                ? "Enter/Tab selects a resource; Escape keeps literal text."
                : searching
                  ? "Searching local vault…"
                  : "No local matches; plain text is kept.")}
          </p>
        </div>
      )}
      {link && (
        <fieldset
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              e.preventDefault();
              closeLink();
            }
          }}
        >
          <legend>Edit Markdown link</legend>
          <label htmlFor="capture-link-label">Link label</label>
          <input
            id="capture-link-label"
            value={link.label}
            onChange={(e) => setLink({ ...link, label: e.target.value })}
          />
          <label htmlFor="capture-link-url">Link URL</label>
          <input
            id="capture-link-url"
            value={link.url}
            onChange={(e) => setLink({ ...link, url: e.target.value })}
          />
          <button type="button" onClick={applyLink}>
            Apply link
          </button>
          <button
            type="button"
            onClick={() => {
              closeLink();
            }}
          >
            Cancel link
          </button>
          {linkError && <p role="alert">{linkError}</p>}
        </fieldset>
      )}
      <p id="capture-editing-help">
        Enter inserts a newline. Ctrl/Cmd+Enter keeps the draft locally; delivery is unavailable
        until configured. Plain @context is parsed only in Quick Add; selected suggestions insert
        typed local links.
      </p>
    </div>
  );
}
