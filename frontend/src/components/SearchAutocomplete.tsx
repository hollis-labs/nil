import * as React from "react";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  value: string;
  onChange: (value: string) => void;
  onSearch: () => void;
  inputRef?: React.RefObject<HTMLInputElement>;
};

type Suggestion = {
  type: 'project' | 'context' | 'tag';
  value: string;
  display: string;
};

export default function SearchAutocomplete({ value, onChange, onSearch, inputRef }: Props) {
  const [suggestions, setSuggestions] = React.useState<Suggestion[]>([]);
  const [allProjects, setAllProjects] = React.useState<string[]>([]);
  const [allContexts, setAllContexts] = React.useState<string[]>([]);
  const [allTags, setAllTags] = React.useState<string[]>([]);
  const [showDropdown, setShowDropdown] = React.useState(false);
  const [selectedIndex, setSelectedIndex] = React.useState(0);
  const dropdownRef = React.useRef<HTMLDivElement>(null);

  // Load all available filters
  React.useEffect(() => {
    Backend.GetFilters().then((result: any) => {
      if (result && typeof result === 'object') {
        setAllProjects(result.projects || []);
        setAllContexts(result.contexts || []);
        setAllTags(result.tags || []);
      }
    }).catch(err => {
      console.error("Failed to load filters:", err);
    });
  }, []);

  // Generate suggestions based on current input
  React.useEffect(() => {
    const words = value.split(/\s+/);
    const lastWord = words[words.length - 1]?.toLowerCase() || '';
    
    if (lastWord.length < 2) {
      setSuggestions([]);
      setShowDropdown(false);
      return;
    }

    const newSuggestions: Suggestion[] = [];

    // Check if last word starts with a prefix
    const hasPrefix = lastWord.startsWith('+') || lastWord.startsWith('@') || lastWord.startsWith('#');
    const searchTerm = hasPrefix ? lastWord.slice(1) : lastWord;

    if (!hasPrefix || lastWord.startsWith('+')) {
      // Suggest projects
      allProjects.forEach(p => {
        if (p.toLowerCase().includes(searchTerm)) {
          newSuggestions.push({
            type: 'project',
            value: p,
            display: `+${p}`
          });
        }
      });
    }

    if (!hasPrefix || lastWord.startsWith('@')) {
      // Suggest contexts
      allContexts.forEach(c => {
        if (c.toLowerCase().includes(searchTerm)) {
          newSuggestions.push({
            type: 'context',
            value: c,
            display: `@${c}`
          });
        }
      });
    }

    if (!hasPrefix || lastWord.startsWith('#')) {
      // Suggest tags
      allTags.forEach(t => {
        if (t.toLowerCase().includes(searchTerm)) {
          newSuggestions.push({
            type: 'tag',
            value: t,
            display: `#${t}`
          });
        }
      });
    }

    setSuggestions(newSuggestions.slice(0, 10)); // Limit to 10 suggestions
    setShowDropdown(newSuggestions.length > 0);
    setSelectedIndex(0);
  }, [value, allProjects, allContexts, allTags]);

  const handleSelect = (suggestion: Suggestion) => {
    const words = value.split(/\s+/);
    words[words.length - 1] = suggestion.display;
    onChange(words.join(' ') + ' ');
    setShowDropdown(false);
    inputRef?.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!showDropdown) {
      if (e.key === 'Enter') {
        onSearch();
      }
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex(prev => (prev + 1) % suggestions.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex(prev => (prev - 1 + suggestions.length) % suggestions.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (suggestions[selectedIndex]) {
        handleSelect(suggestions[selectedIndex]);
      } else {
        onSearch();
      }
    } else if (e.key === 'Escape') {
      setShowDropdown(false);
    }
  };

  React.useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'project': return 'var(--term-success)';
      case 'context': return 'var(--term-info)';
      case 'tag': return 'var(--term-warn)';
      default: return 'var(--term-fg)';
    }
  };

  return (
    <div style={{ position: 'relative', flex: 1 }}>
      <input
        ref={inputRef}
        style={{
          width: '100%',
          padding: '8px 12px',
          background: 'var(--term-panel)',
          border: '1px solid var(--term-border)',
          borderRadius: '6px',
          color: 'var(--term-fg)',
          fontSize: '13px',
          outline: 'none'
        }}
        placeholder="Search… e.g. review #work +todoapp @home pri:A"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        onFocus={(e) => e.currentTarget.style.borderColor = 'var(--term-accent)'}
        onBlur={(e) => e.currentTarget.style.borderColor = 'var(--term-border)'}
      />
      
      {showDropdown && suggestions.length > 0 && (
        <div
          ref={dropdownRef}
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            marginTop: '4px',
            background: 'var(--term-panel)',
            border: '1px solid var(--term-border)',
            borderRadius: '6px',
            maxHeight: '200px',
            overflowY: 'auto',
            zIndex: 1000,
            boxShadow: '0 4px 6px rgba(0, 0, 0, 0.3)'
          }}
        >
          {suggestions.map((suggestion, index) => (
            <button
              key={`${suggestion.type}-${suggestion.value}`}
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                handleSelect(suggestion);
              }}
              style={{
                width: '100%',
                padding: '8px 12px',
                background: index === selectedIndex ? 'var(--term-hover)' : 'transparent',
                border: 'none',
                textAlign: 'left',
                cursor: 'pointer',
                fontSize: '13px',
                color: 'var(--term-fg)',
                display: 'flex',
                alignItems: 'center',
                gap: '8px'
              }}
            >
              <span style={{ 
                color: getTypeColor(suggestion.type),
                fontWeight: 600,
                fontFamily: 'monospace'
              }}>
                {suggestion.display}
              </span>
              <span style={{ 
                fontSize: '11px',
                color: 'var(--term-dim)',
                textTransform: 'capitalize'
              }}>
                {suggestion.type}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
