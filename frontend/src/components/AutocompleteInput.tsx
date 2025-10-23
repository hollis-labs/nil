import * as React from "react";

type Props = {
  values: string[];
  onValuesChange: (values: string[]) => void;
  suggestions: string[];
  placeholder: string;
  prefix?: string;
  label?: string;
};

export default function AutocompleteInput({
  values,
  onValuesChange,
  suggestions,
  placeholder,
  prefix = "",
  label,
}: Props) {
  const [inputValue, setInputValue] = React.useState("");
  const [showDropdown, setShowDropdown] = React.useState(false);
  const [selectedIndex, setSelectedIndex] = React.useState(0);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const dropdownRef = React.useRef<HTMLDivElement>(null);

  // Filter suggestions based on input
  const filteredSuggestions = React.useMemo(() => {
    if (!inputValue.trim()) return suggestions;
    const query = inputValue.toLowerCase();
    return suggestions
      .filter(s => s.toLowerCase().includes(query) && !values.includes(s))
      .slice(0, 10);
  }, [inputValue, suggestions, values]);

  // Show dropdown when input has focus and there are suggestions
  React.useEffect(() => {
    if (inputValue && filteredSuggestions.length > 0) {
      setShowDropdown(true);
      setSelectedIndex(0);
    } else {
      setShowDropdown(false);
    }
  }, [inputValue, filteredSuggestions]);

  // Click outside to close
  React.useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        !inputRef.current?.contains(e.target as Node)
      ) {
        setShowDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const addValue = (value: string) => {
    if (value && !values.includes(value)) {
      onValuesChange([...values, value]);
      setInputValue("");
      setShowDropdown(false);
      inputRef.current?.focus();
    }
  };

  const removeValue = (valueToRemove: string) => {
    onValuesChange(values.filter(v => v !== valueToRemove));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (showDropdown && filteredSuggestions.length > 0) {
        addValue(filteredSuggestions[selectedIndex]);
      } else if (inputValue.trim()) {
        addValue(inputValue.trim());
      }
    } else if (e.key === 'Tab') {
      // If there's input, add it. If empty, allow tab to next field
      if (inputValue.trim()) {
        e.preventDefault();
        if (showDropdown && filteredSuggestions.length > 0) {
          addValue(filteredSuggestions[selectedIndex]);
        } else {
          addValue(inputValue.trim());
        }
      }
      // else: let Tab naturally move to next field
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (showDropdown) {
        setSelectedIndex((prev) => (prev + 1) % filteredSuggestions.length);
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (showDropdown) {
        setSelectedIndex((prev) => (prev - 1 + filteredSuggestions.length) % filteredSuggestions.length);
      }
    } else if (e.key === 'Backspace' && inputValue === '' && values.length > 0) {
      e.preventDefault();
      removeValue(values[values.length - 1]);
    } else if (e.key === 'Escape') {
      setShowDropdown(false);
    }
  };

  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '6px',
    padding: '8px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '6px',
    minHeight: '44px',
    cursor: 'text',
    position: 'relative',
  };

  const chipStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '4px',
    padding: '4px 8px',
    background: 'var(--term-panel)',
    border: '1px solid var(--term-border)',
    borderRadius: '4px',
    fontSize: '12px',
    color: 'var(--term-fg)',
    cursor: 'pointer',
    transition: 'all 0.15s ease',
  };

  const inputStyle: React.CSSProperties = {
    flex: 1,
    minWidth: '120px',
    background: 'transparent',
    border: 'none',
    outline: 'none',
    color: 'var(--term-fg)',
    fontSize: '13px',
    padding: '4px',
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
      <div style={containerStyle} onClick={() => inputRef.current?.focus()}>
        {values.map((value) => (
          <span
            key={value}
            style={chipStyle}
            onClick={(e) => {
              e.stopPropagation();
              removeValue(value);
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-accent)';
              e.currentTarget.style.background = 'rgba(122, 162, 247, 0.1)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-border)';
              e.currentTarget.style.background = 'var(--term-panel)';
            }}
          >
            {prefix}{value}
            <span style={{ fontSize: '14px', marginLeft: '2px', opacity: 0.7 }}>×</span>
          </span>
        ))}
        <input
          ref={inputRef}
          type="text"
          style={inputStyle}
          placeholder={values.length === 0 ? placeholder : ''}
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => {
            if (inputValue && filteredSuggestions.length > 0) {
              setShowDropdown(true);
            }
          }}
        />
      </div>
      {showDropdown && filteredSuggestions.length > 0 && (
        <div ref={dropdownRef} style={dropdownStyle}>
          {filteredSuggestions.map((item, index) => (
            <button
              key={item}
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                addValue(item);
              }}
              style={itemStyle(index === selectedIndex)}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              {prefix && <span style={{ opacity: 0.7, fontSize: '12px', minWidth: '12px' }}>{prefix}</span>}
              {item}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
