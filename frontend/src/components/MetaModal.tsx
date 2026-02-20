import * as React from "react";
import { ItemRow } from "./TerminalList";
import ContextsAutocomplete from "./ContextsAutocomplete";
import ProjectsAutocomplete from "./ProjectsAutocomplete";
import TagsAutocomplete from "./TagsAutocomplete";
import CustomScrollbar from "./CustomScrollbar";

type Props = {
  open: boolean;
  todo: ItemRow | null;
  onOpenChange: (open: boolean) => void;
  onSave: (id: number, updates: Partial<ItemRow>) => void;
};

export default function MetaModal({ open, todo, onOpenChange, onSave }: Props) {
  const [contexts, setContexts] = React.useState<string[]>([]);
  const [projects, setProjects] = React.useState<string[]>([]);
  const [tags, setTags] = React.useState<string[]>([]);
  const [priority, setPriority] = React.useState<'A' | 'B' | 'C' | ''>('');
  const [dueDate, setDueDate] = React.useState<string>('');

  React.useEffect(() => {
    if (open && todo) {
      setContexts(todo.contexts || []);
      setProjects(todo.projects || []);
      setTags(todo.tags || []);
      setPriority((todo.priority || '') as 'A' | 'B' | 'C' | '');
      setDueDate(todo.due_at ? todo.due_at.slice(0, 10) : '');
    }
  }, [open, todo]);

  const handleSave = () => {
    if (!todo) return;

    onSave(todo.id, {
      contexts,
      projects,
      tags,
      priority: priority || undefined,
      due_at: dueDate || undefined,
    });

    onOpenChange(false);
  };

  if (!open || !todo) return null;

  const modalStyle: React.CSSProperties = {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50,
  };

  return (
    <div style={modalStyle} onClick={() => onOpenChange(false)}>
      <div 
        className="terminal-card" 
        style={{
          width: '480px',
          maxWidth: '95vw',
          height: '550px',
          maxHeight: '85vh',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden'
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ padding: '20px 20px 0 20px' }}>
          <div style={{ marginBottom: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'end', justifyContent: 'flex-end', marginBottom: '1px' }}>
              <button
                className="badge info"
                onClick={() => onOpenChange(false)}
                style={{
                  padding: '8px 12px',
                  fontSize: '13px',
                  borderRadius: '6px'
                }}
              >
                Close
              </button>
            </div>
            <label style={{ fontSize: '12px', marginLeft: '3px', textTransform: 'uppercase', letterSpacing: '0.5px' }} className="text-dim">Quick Meta Edit:</label>
            <div style={{
              padding: '4px 4px',
              borderBottom: '1px solid var(--term-border)',
              color: 'var(--term-fg)',
              fontSize: '18px',
              fontWeight: 500,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap'
            }}>
              {todo.title}
            </div>
          </div>
        </div>

        <CustomScrollbar style={{ flex: 1, minHeight: 0 }}>
          <div style={{ padding: '0 20px 40px 20px' }}>
            <div style={{
              display: 'flex',
              gap: '0px',
              marginBottom: '12px',
              flexWrap: 'wrap'
            }}>
              <button
                type="button"
                className={`badge ${priority === 'C' ? 'success' : ''}`}
                onClick={() => setPriority(priority === 'C' ? '' : 'C')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '4px 0 0 4px',
                  fontSize: '11px',
                  padding: '4px 8px',
                  opacity: priority === 'C' ? 1 : 0.8
                }}
              >
                LOW
              </button>
              <button
                type="button"
                className={`badge ${priority === 'B' ? 'info' : ''}`}
                onClick={() => setPriority(priority === 'B' ? '' : 'B')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '0',
                  borderLeft: '1px solid var(--term-border)',
                  fontSize: '11px',
                  padding: '4px 8px',
                  opacity: priority === 'B' ? 1 : 0.8
                }}
              >
                MED
              </button>
              <button
                type="button"
                className={`badge ${priority === 'A' ? 'warn' : ''}`}
                onClick={() => setPriority(priority === 'A' ? '' : 'A')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '0',
                  borderLeft: '1px solid var(--term-border)',
                  fontSize: '11px',
                  padding: '4px 8px',
                  opacity: priority === 'A' ? 1 : 0.8
                }}
              >
                HIGH
              </button>
              <button
                type="button"
                className={`badge ${priority === '' ? 'success' : ''}`}
                onClick={() => setPriority('')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '0 4px 4px 0',
                  borderLeft: '1px solid var(--term-border)',
                  fontSize: '11px',
                  padding: '4px 8px'
                }}
              >
                NA
              </button>
            </div>

            <div style={{ marginBottom: '12px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Due Date</label>
              <input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  background: 'var(--term-bg)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  color: 'var(--term-fg)',
                  fontSize: '13px',
                }}
              />
            </div>

            <div style={{
              display: 'grid',
              gridTemplateColumns: '1fr',
              gap: '12px',
              marginBottom: '12px'
            }}>
              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Projects</label>
                <ProjectsAutocomplete values={projects} onValuesChange={setProjects} placeholder="Add project..." />
              </div>

              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Contexts</label>
                <ContextsAutocomplete values={contexts} onValuesChange={setContexts} placeholder="Add context..." />
              </div>

              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Tags</label>
                <TagsAutocomplete values={tags} onValuesChange={setTags} placeholder="Add tag..." />
              </div>
            </div>
          </div>
        </CustomScrollbar>

        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '16px 20px 20px 20px',
          borderTop: '1px solid var(--term-border)',
        }}>
          <button 
            type="button" 
            className="badge" 
            onClick={() => onOpenChange(false)} 
            style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', background: 'var(--term-bg)', border: '1px solid var(--term-border)' }}
          >
            Cancel
          </button>
          <button
            type="button"
            className="badge success"
            onClick={handleSave}
            style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
          >
            Save Changes
          </button>
        </div>
      </div>
    </div>
  );
}
