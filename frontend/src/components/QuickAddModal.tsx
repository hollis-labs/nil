import * as React from "react";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSubmit: (line: string, extras: any) => void;
};

export default function QuickAddModal({ open, onOpenChange, onSubmit }: Props) {
  const [line, setLine] = React.useState("");
  const [priority, setPriority] = React.useState("");
  const [due, setDue] = React.useState("");

  React.useEffect(() => {
    if (open) {
      setLine("");
      setPriority("");
      setDue(""); // No default due date
    }
  }, [open]);

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onOpenChange(false);
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [open, onOpenChange]);

  if (!open) return null;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const extras: any = {};
    if (priority) extras.priority = priority;
    if (due) extras.due = due;
    onSubmit(line, extras);
  }

  const modalStyle = {
    position: 'fixed' as const,
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50
  };

  const inputStyle = {
    width: '100%',
    padding: '8px 12px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '6px',
    color: 'var(--term-fg)',
    fontSize: '13px'
  };

  return (
    <div style={modalStyle}>
      <div className="terminal-card" style={{ width: '640px', maxWidth: '95vw', padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div style={{ fontWeight: 600 }}>Quick Add Todo</div>
          <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Close</button>
        </div>
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Task</label>
            <input
              autoFocus
              style={inputStyle}
              placeholder="e.g. Review pull request +project @context #tag"
              value={line}
              onChange={(e) => setLine(e.target.value)}
            />
          </div>
          <div style={{ display: 'flex', gap: '12px' }}>
            <div style={{ flex: 1 }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Priority</label>
              <select 
                style={{
                  ...inputStyle,
                  appearance: 'none',
                  backgroundImage: `url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e")`,
                  backgroundRepeat: 'no-repeat',
                  backgroundPosition: 'right 8px center',
                  backgroundSize: '16px',
                  paddingRight: '32px'
                }}
                value={priority} 
                onChange={(e) => setPriority(e.target.value)}
              >
                <option value="" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>None</option>
                <option value="A" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>A (High)</option>
                <option value="B" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>B (Medium)</option>
                <option value="C" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>C (Low)</option>
              </select>
            </div>
            <div style={{ flex: 1 }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Due Date</label>
              <input type="date" style={inputStyle} value={due} onChange={(e) => setDue(e.target.value)} placeholder="Optional" />
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '8px' }}>
            <button type="button" className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Cancel</button>
            <button type="submit" className="badge success" style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Create</button>
          </div>
        </form>
      </div>
    </div>
  );
}
