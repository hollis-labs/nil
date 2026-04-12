import * as React from "react";

type Props = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  autoFocus?: boolean;
  recentContexts?: string[];
  recentProjects?: string[];
  recentTags?: string[];
};

type SuggestionItem = {
  value: string;
  type: '@' | '+' | '#';
};

export default function ItemTitleInput({
  value,
  onChange,
  placeholder = "e.g. Review pull request +project @context",
  autoFocus = false,
  recentContexts = [],
  recentProjects = [],
  recentTags = [],
}: Props) {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [showSuggestions, setShowSuggestions] = React.useState(false);
  const [suggestions, setSuggestions] = React.useState<SuggestionItem[]>([]);
  const [selectedIndex, setSelectedIndex] = React.useState(0);
  const [triggerChar, setTriggerChar] = React.useState<'@' | '+' | '#' | null>(null);
  const [triggerPos, setTriggerPos] = React.useState(0);
  const [, setSearchQuery] = React.useState("");

  React.useEffect(() => {
    if (autoFocus && inputRef.current) {
      inputRef.current.focus();
    }
  }, [autoFocus]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    onChange(newValue);

    const cursorPos = e.target.selectionStart || 0;
    
    // Find the last trigger character before cursor
    let lastTrigger: '@' | '+' | '#' | null = null;
    let lastTriggerPos = -1;
    
    for (let i = cursorPos - 1; i >= 0; i--) {
      const char = newValue[i];
      if (char === ' ' || char === '\n') {
        break; // Stop at space or newline
      }
      if (char === '@' || char === '+' || char === '#') {
        lastTrigger = char;
        lastTriggerPos = i;
        break;
      }
    }

    if (lastTrigger && lastTriggerPos >= 0) {
      const query = newValue.slice(lastTriggerPos + 1, cursorPos).toLowerCase();
      setTriggerChar(lastTrigger);
      setTriggerPos(lastTriggerPos);
      setSearchQuery(query);

      let items: string[] = [];
      if (lastTrigger === '@') items = recentContexts;
      else if (lastTrigger === '+') items = recentProjects;
      else if (lastTrigger === '#') items = recentTags;

      const filtered = items
        .filter(item => item.toLowerCase().startsWith(query))
        .slice(0, 10)
        .map(item => ({ value: item, type: lastTrigger as '@' | '+' | '#' }));

      setSuggestions(filtered);
      setShowSuggestions(filtered.length > 0);
      setSelectedIndex(0);
    } else {
      setShowSuggestions(false);
      setSuggestions([]);
    }
  };

  const insertSuggestion = (suggestion: SuggestionItem) => {
    if (!triggerChar) return;

    const beforeTrigger = value.slice(0, triggerPos);
    const afterCursor = value.slice(inputRef.current?.selectionStart || value.length);
    const newValue = `${beforeTrigger}${triggerChar}${suggestion.value} ${afterCursor}`;
    
    onChange(newValue);
    setShowSuggestions(false);
    
    // Set cursor position after the inserted text
    setTimeout(() => {
      if (inputRef.current) {
        const newPos = beforeTrigger.length + triggerChar.length + suggestion.value.length + 1;
        inputRef.current.setSelectionRange(newPos, newPos);
        inputRef.current.focus();
      }
    }, 0);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!showSuggestions || suggestions.length === 0) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((prev) => (prev + 1) % suggestions.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((prev) => (prev - 1 + suggestions.length) % suggestions.length);
    } else if (e.key === 'Tab') {
      e.preventDefault();
      insertSuggestion(suggestions[selectedIndex]!);
    } else if (e.key === 'Enter' && showSuggestions) {
      e.preventDefault();
      insertSuggestion(suggestions[selectedIndex]!);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      setShowSuggestions(false);
    }
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 12px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '6px',
    color: 'var(--term-fg)',
    fontSize: '13px',
    outline: 'none',
    minHeight: '40px',
  };

  const dropdownStyle: React.CSSProperties = {
    position: 'absolute',
    top: '100%',
    left: 0,
    right: 0,
    marginTop: '4px',
    background: 'rgba(0, 0, 0, 0.6)',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    borderRadius: '6px',
    padding: '4px',
    boxShadow: '0 4px 12px rgba(0,0,0,0.5)',
    maxHeight: '200px',
    overflowY: 'auto',
    zIndex: 1000,
    backdropFilter: 'blur(8px)',
  };

  const itemStyle = (isSelected: boolean): React.CSSProperties => ({
    width: '100%',
    textAlign: 'left',
    padding: '8px 12px',
    background: isSelected ? 'rgba(255, 255, 255, 0.15)' : 'transparent',
    color: isSelected ? 'var(--term-fg)' : 'rgba(255, 255, 255, 0.6)',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '13px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.15s ease',
  });

  return (
    <div style={{ position: 'relative' }}>
      <input
        ref={inputRef}
        type="text"
        style={inputStyle}
        placeholder={placeholder}
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
      />
      {showSuggestions && suggestions.length > 0 && (
        <div style={dropdownStyle}>
          {suggestions.map((item, index) => (
            <button
              key={`${item.type}${item.value}`}
              onClick={() => insertSuggestion(item)}
              style={itemStyle(index === selectedIndex)}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              <span style={{ opacity: 0.7, fontSize: '12px', minWidth: '12px' }}>
                {item.type}
              </span>
              {item.value}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
