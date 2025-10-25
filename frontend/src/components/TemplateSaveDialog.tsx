import * as React from "react";
import { saveTemplate } from "@/lib/templates";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  values: {
    contexts: string[];
    projects: string[];
    tags: string[];
    priority?: 'A' | 'B' | 'C';
  };
  onSaved?: () => void;
};

export default function TemplateSaveDialog({ open, onOpenChange, values, onSaved }: Props) {
  const [templateName, setTemplateName] = React.useState("");

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (!templateName.trim()) return;

    saveTemplate({
      name: templateName.trim(),
      contexts: values.contexts,
      projects: values.projects,
      tags: values.tags,
      priority: values.priority,
    });

    setTemplateName("");
    onOpenChange(false);
    onSaved?.();
  };

  if (!open) return null;

  const modalStyle: React.CSSProperties = {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 100,
  };

  const dialogStyle: React.CSSProperties = {
    background: 'var(--term-panel)',
    border: '1px solid var(--term-border)',
    borderRadius: '8px',
    padding: '20px',
    maxWidth: '400px',
    width: '90%',
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 12px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '6px',
    color: 'var(--term-fg)',
    fontSize: '13px',
    marginBottom: '12px',
  };

  return (
    <div style={modalStyle} onClick={() => onOpenChange(false)}>
      <div style={dialogStyle} onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginBottom: '16px', fontSize: '16px', fontWeight: 600 }}>Save as Template</h3>

        <form onSubmit={handleSave}>
          <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">
            Template Name
          </label>
          <input
            type="text"
            style={inputStyle}
            placeholder="e.g., Work Tasks"
            value={templateName}
            onChange={(e) => setTemplateName(e.target.value)}
            autoFocus
          />

          <div style={{ marginBottom: '16px', padding: '12px', background: 'var(--term-bg)', borderRadius: '6px', border: '1px solid var(--term-border)' }}>
            <div style={{ fontSize: '12px', marginBottom: '8px', fontWeight: 500 }}>Template Preview:</div>
            <div style={{ fontSize: '11px', color: 'var(--term-dim)' }}>
              {values.projects.length > 0 && <div>Projects: +{values.projects.join(', +')}</div>}
              {values.contexts.length > 0 && <div>Contexts: @{values.contexts.join(', @')}</div>}
              {values.tags.length > 0 && <div>Tags: #{values.tags.join(', #')}</div>}
              {values.priority && <div>Priority: {values.priority}</div>}
              {values.projects.length === 0 && values.contexts.length === 0 && values.tags.length === 0 && !values.priority && (
                <div style={{ fontStyle: 'italic' }}>No values to save</div>
              )}
            </div>
          </div>

          <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
            <button
              type="button"
              className="badge"
              onClick={() => onOpenChange(false)}
              style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="badge success"
              disabled={!templateName.trim()}
              style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
            >
              Save Template
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
