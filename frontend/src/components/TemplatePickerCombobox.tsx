import * as React from "react";
import { TaskTemplate, getTemplates } from "@/lib/templates";
import { ChevronDown, FileText } from "lucide-react";

type Props = {
  onSelect: (template: TaskTemplate) => void;
  currentValues?: {
    contexts: string[];
    projects: string[];
    tags: string[];
    priority?: string;
  };
};

export default function TemplatePickerCombobox({ onSelect, currentValues }: Props) {
  const [templates, setTemplates] = React.useState<TaskTemplate[]>([]);
  const [isOpen, setIsOpen] = React.useState(false);
  const [searchQuery, setSearchQuery] = React.useState("");
  const dropdownRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    setTemplates(getTemplates());
  }, [isOpen]);

  React.useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const filteredTemplates = React.useMemo(() => {
    if (!searchQuery.trim()) return templates;
    const query = searchQuery.toLowerCase();
    return templates.filter(t => t.name.toLowerCase().includes(query));
  }, [templates, searchQuery]);

  const handleSelect = (template: TaskTemplate) => {
    onSelect(template);
    setIsOpen(false);
    setSearchQuery("");
  };

  const buttonStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    padding: '6px 10px',
    background: 'var(--term-panel)',
    border: '1px solid var(--term-border)',
    borderRadius: '4px',
    color: 'var(--term-fg)',
    fontSize: '12px',
    cursor: 'pointer',
    transition: 'all 0.15s ease',
  };

  const dropdownStyle: React.CSSProperties = {
    position: 'absolute',
    top: '100%',
    left: 0,
    marginTop: '4px',
    minWidth: '250px',
    background: 'rgba(0, 0, 0, 0.9)',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    borderRadius: '6px',
    padding: '8px',
    boxShadow: '0 4px 12px rgba(0,0,0,0.5)',
    zIndex: 1000,
    backdropFilter: 'blur(8px)',
  };

  const itemStyle: React.CSSProperties = {
    padding: '8px 12px',
    background: 'transparent',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    width: '100%',
    textAlign: 'left',
    transition: 'all 0.15s ease',
    marginBottom: '4px',
  };

  return (
    <div ref={dropdownRef} style={{ position: 'relative', display: 'inline-block' }}>
      <button
        type="button"
        style={buttonStyle}
        onClick={() => setIsOpen(!isOpen)}
        onMouseEnter={(e) => {
          e.currentTarget.style.borderColor = 'var(--term-accent)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.borderColor = 'var(--term-border)';
        }}
      >
        <FileText size={14} />
        Templates
        <ChevronDown size={14} />
      </button>

      {isOpen && (
        <div style={dropdownStyle}>
          <input
            type="text"
            placeholder="Search templates..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{
              width: '100%',
              padding: '6px 10px',
              background: 'var(--term-bg)',
              border: '1px solid var(--term-border)',
              borderRadius: '4px',
              color: 'var(--term-fg)',
              fontSize: '12px',
              marginBottom: '8px',
            }}
            autoFocus
          />

          <div style={{ maxHeight: '250px', overflowY: 'auto' }}>
            {filteredTemplates.length === 0 ? (
              <div style={{ padding: '16px', textAlign: 'center', color: 'var(--term-dim)', fontSize: '12px' }}>
                No templates found
              </div>
            ) : (
              filteredTemplates.map((template) => (
                <button
                  key={template.id}
                  onClick={() => handleSelect(template)}
                  style={itemStyle}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'rgba(255, 255, 255, 0.1)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'transparent';
                  }}
                >
                  <div style={{ fontWeight: 500, marginBottom: '4px', color: 'var(--term-fg)', fontSize: '13px' }}>
                    {template.name}
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--term-dim)' }}>
                    {template.projects.length > 0 && <span>+{template.projects.join(', +')}</span>}
                    {template.contexts.length > 0 && <span> @{template.contexts.join(', @')}</span>}
                    {template.tags.length > 0 && <span> #{template.tags.join(', #')}</span>}
                    {template.priority && <span> pri:{template.priority}</span>}
                  </div>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
