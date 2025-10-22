import * as React from "react";

type TagsInputProps = {
  tags: string[];
  onTagsChange: (tags: string[]) => void;
  placeholder?: string;
};

export default function TagsInput({ tags, onTagsChange, placeholder = "Add tag..." }: TagsInputProps) {
  const [inputValue, setInputValue] = React.useState("");
  const inputRef = React.useRef<HTMLInputElement>(null);

  const addTag = () => {
    const trimmed = inputValue.trim();
    if (trimmed && !tags.includes(trimmed)) {
      onTagsChange([...tags, trimmed]);
      setInputValue("");
    }
  };

  const removeTag = (tagToRemove: string) => {
    onTagsChange(tags.filter(t => t !== tagToRemove));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag();
    } else if (e.key === 'Backspace' && inputValue === '' && tags.length > 0) {
      e.preventDefault();
      onTagsChange(tags.slice(0, -1));
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
    minHeight: '40px',
    cursor: 'text',
  };

  const tagStyle: React.CSSProperties = {
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

  return (
    <div style={containerStyle} onClick={() => inputRef.current?.focus()}>
      {tags.map((tag) => (
        <span
          key={tag}
          style={tagStyle}
          onClick={(e) => {
            e.stopPropagation();
            removeTag(tag);
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
          #{tag}
          <span style={{ fontSize: '14px', marginLeft: '2px', opacity: 0.7 }}>×</span>
        </span>
      ))}
      <input
        ref={inputRef}
        type="text"
        style={inputStyle}
        placeholder={tags.length === 0 ? placeholder : ''}
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={() => {
          if (inputValue.trim()) {
            addTag();
          }
        }}
      />
    </div>
  );
}
