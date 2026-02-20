import * as React from "react";

type RefItem = {
  id: number;
  title: string;
  type: string;
};

type Props = {
  items: RefItem[];
  command: (attrs: { id: number; label: string; refType: string }) => void;
};

const WikilinkSuggestion = React.forwardRef<
  { onKeyDown: (props: { event: KeyboardEvent }) => boolean },
  Props
>(({ items, command }, ref) => {
  const [selectedIndex, setSelectedIndex] = React.useState(0);

  React.useEffect(() => {
    setSelectedIndex(0);
  }, [items]);

  const selectItem = (index: number) => {
    const item = items[index];
    if (item) {
      command({ id: item.id, label: item.title, refType: item.type });
    }
  };

  React.useImperativeHandle(ref, () => ({
    onKeyDown: ({ event }: { event: KeyboardEvent }) => {
      if (event.key === "ArrowUp") {
        setSelectedIndex((i) => (i - 1 + Math.max(items.length, 1)) % Math.max(items.length, 1));
        return true;
      }
      if (event.key === "ArrowDown") {
        setSelectedIndex((i) => (i + 1) % Math.max(items.length, 1));
        return true;
      }
      if (event.key === "Enter") {
        selectItem(selectedIndex);
        return true;
      }
      return false;
    },
  }));

  if (items.length === 0) return null;

  return (
    <div
      style={{
        background: "var(--term-panel)",
        border: "1px solid var(--term-border)",
        borderRadius: "6px",
        boxShadow: "0 4px 16px rgba(0,0,0,0.4)",
        overflow: "hidden",
        minWidth: "260px",
        maxWidth: "360px",
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
        fontSize: "13px",
      }}
    >
      {items.map((item, index) => (
        <button
          key={item.id}
          onClick={() => selectItem(index)}
          onMouseEnter={() => setSelectedIndex(index)}
          style={{
            display: "flex",
            alignItems: "center",
            gap: "8px",
            width: "100%",
            padding: "8px 12px",
            background: index === selectedIndex ? "var(--term-accent)" : "transparent",
            color: index === selectedIndex ? "#000" : "var(--term-fg)",
            border: "none",
            borderBottom: "1px solid var(--term-border)",
            cursor: "pointer",
            textAlign: "left",
            transition: "background 0.1s ease",
          }}
        >
          <span
            style={{
              fontSize: "10px",
              padding: "2px 5px",
              borderRadius: "3px",
              fontWeight: 600,
              letterSpacing: "0.04em",
              flexShrink: 0,
              background: item.type === "note"
                ? "rgba(96,165,250,0.25)"
                : "rgba(74,222,128,0.25)",
              color: item.type === "note"
                ? (index === selectedIndex ? "#000" : "var(--term-info)")
                : (index === selectedIndex ? "#000" : "var(--term-success)"),
              border: `1px solid ${item.type === "note" ? "var(--term-info)" : "var(--term-success)"}`,
            }}
          >
            {item.type}
          </span>
          <span
            style={{
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {item.title}
          </span>
        </button>
      ))}
    </div>
  );
});

WikilinkSuggestion.displayName = "WikilinkSuggestion";
export default WikilinkSuggestion;
