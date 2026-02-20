import * as React from "react";
import CustomScrollbar from "./CustomScrollbar";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export default function HelpModal({ open, onOpenChange }: Props) {
  const [activeTab, setActiveTab] = React.useState<'app' | 'syntax'>('app');

  if (!open) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.85)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 10000,
      }}
      onClick={() => onOpenChange(false)}
    >
      <div
        style={{
          background: 'var(--term-bg)',
          border: '1px solid var(--term-border)',
          borderRadius: '12px',
          width: '700px',
          maxWidth: '90vw',
          height: '650px',
          maxHeight: '90vh',
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden'
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Fixed Header */}
        <div style={{ padding: '20px 20px 0 20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
            <div style={{ fontWeight: 600, fontSize: '16px' }}>Help &amp; Guide</div>
            <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Close</button>
          </div>

          {/* Tab Navigation */}
          <div style={{ display: 'flex', gap: '0px', marginBottom: '20px', paddingBottom: '12px', borderBottom: '1px solid var(--term-border)' }}>
            <button
              onClick={() => setActiveTab('app')}
              className={`badge ${activeTab === 'app' ? 'success' : ''}`}
              style={{
                padding: '4px 8px',
                fontSize: '11px',
                borderRadius: '4px 0 0 4px',
                border: 'none',
                opacity: activeTab === 'app' ? 1 : 0.8
              }}
            >
              App Help
            </button>
            <button
              onClick={() => setActiveTab('syntax')}
              className={`badge ${activeTab === 'syntax' ? 'success' : ''}`}
              style={{
                padding: '4px 8px',
                fontSize: '11px',
                borderRadius: '0 4px 4px 0',
                border: 'none',
                borderLeft: '1px solid var(--term-border)',
                opacity: activeTab === 'syntax' ? 1 : 0.8
              }}
            >
              Syntax Guide
            </button>
          </div>
        </div>

        {/* Scrollable Content */}
        <CustomScrollbar style={{ flex: 1, minHeight: 0 }}>
          <div style={{ padding: '0 20px' }}>
            {activeTab === 'app' && <AppHelp />}
            {activeTab === 'syntax' && <SyntaxHelp />}
          </div>
        </CustomScrollbar>
      </div>
    </div>
  );
}

function AppHelp() {
  const code = (s: string) => (
    <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>{s}</code>
  );

  return (
    <div style={{ paddingBottom: '20px' }}>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginTop: 0, marginBottom: '12px' }}>Getting Started with NANITE</h3>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '12px' }}>NANITE is a keyboard-driven task and note manager. Here's how to get started:</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Keyboard Shortcuts</h4>
      <table style={{ width: '100%', fontSize: '13px', borderCollapse: 'collapse', marginBottom: '16px' }}>
        <tbody>
          {[
            ['Cmd/Ctrl+N', 'New Item'],
            ['Cmd/Ctrl+Shift+N', 'New Note'],
            ['Cmd/Ctrl+S', 'Quick Search'],
            ['Shift+Shift', 'Quick Search (alias)'],
            ['Cmd/Ctrl+I', 'Open Inbox'],
            ['Cmd/Ctrl+Shift+M', 'Toggle Items / Notes mode'],
            ['Escape', 'Close modal / back'],
          ].map(([key, desc]) => (
            <tr key={key}>
              <td style={{ padding: '4px 12px 4px 0', fontFamily: 'monospace', fontSize: '12px', whiteSpace: 'nowrap', color: 'var(--term-accent)' }}>{key}</td>
              <td style={{ padding: '4px 0', opacity: 0.9 }}>{desc}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Adding Items</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Press <strong>Cmd+N</strong> or click the <strong>+</strong> button. Type what you need to do and press Enter.</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Organizing with Sections</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>NANITE has three sections to help you focus:</p>
      <ul style={{ marginLeft: '20px', fontSize: '13px', lineHeight: '1.6' }}>
        <li style={{ marginBottom: '6px' }}><strong>Now</strong> — What you're working on right now (keep this to 3–5 items)</li>
        <li style={{ marginBottom: '6px' }}><strong>Soon</strong> — What's coming up next</li>
        <li style={{ marginBottom: '6px' }}><strong>Anytime</strong> — Everything else: ideas, someday tasks</li>
      </ul>
      <p style={{ fontSize: '13px', lineHeight: '1.6' }}>Right-click any item and choose "Move to…" to organize it.</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Inbox — Fast Capture</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>
        Press <strong>Cmd+I</strong> to open the Inbox. Create an item with no title to send it straight to the Inbox — add context later.
        Use the "→ Inbox" button in the create modal to deliberately route any item to the Inbox regardless of content.
      </p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Searching</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Use the search bar or Quick Search (Cmd+S) to filter items:</p>
      <ul style={{ marginLeft: '20px', fontSize: '13px', lineHeight: '1.6' }}>
        <li style={{ marginBottom: '6px' }}>Type any word to search by keyword</li>
        <li style={{ marginBottom: '6px' }}>Type {code('+project')} to filter by project</li>
        <li style={{ marginBottom: '6px' }}>Type {code('@context')} to filter by context</li>
        <li style={{ marginBottom: '6px' }}>Type {code('#tag')} to filter by tag</li>
        <li style={{ marginBottom: '6px' }}>Combine freely: {code('+work @computer #urgent')}</li>
      </ul>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Sessions — Focused Work</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>
        Click the 🎯 icon to set a temporary session context (e.g., {code('+work @computer')}).
        While a session is active, new items automatically inherit the session's tags and contexts for faster capture.
        Enable "Use as Filter Tab" to filter the main view to only session items.
      </p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Notes</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6' }}>
        Click any item to add rich-text notes with Markdown support: headings, bold, italic, lists, code, and {code('@reference')} wikilinks that link to other items.
      </p>
    </div>
  );
}

function SyntaxHelp() {
  return (
    <div style={{ paddingBottom: '20px' }}>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginTop: 0, marginBottom: '12px' }}>Quick Add Syntax Guide</h3>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '12px' }}>
        NANITE uses a todo.txt-inspired input syntax with extensions for due dates, recurrence, and tagging.
        It is <em>not</em> a strict todo.txt implementation — notably, {' '}
        <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>#tag</code> is a NANITE extension
        (standard todo.txt uses key:value pairs).
      </p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Full Example</h4>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Buy groceries +home @errands #tonight due:2026-02-25
      </code>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Syntax Reference</h4>
      <table style={{ width: '100%', fontSize: '13px', borderCollapse: 'collapse', marginBottom: '16px' }}>
        <thead>
          <tr>
            <th style={{ textAlign: 'left', padding: '4px 12px 8px 0', opacity: 0.6, fontSize: '11px', fontWeight: 500 }}>Token</th>
            <th style={{ textAlign: 'left', padding: '4px 0 8px 0', opacity: 0.6, fontSize: '11px', fontWeight: 500 }}>Meaning</th>
            <th style={{ textAlign: 'left', padding: '4px 0 8px 0', opacity: 0.6, fontSize: '11px', fontWeight: 500 }}>Example</th>
          </tr>
        </thead>
        <tbody>
          {[
            ['(A)', 'Priority — A, B, or C', '(A), (B), (C)'],
            ['+project', 'Project tag', '+shopping, +work'],
            ['@context', 'Context tag', '@errands, @computer'],
            ['#tag', 'Flexible label (NANITE extension)', '#urgent, #ideas'],
            ['due:YYYY-MM-DD', 'Due date', 'due:2026-02-25'],
            ['t:YYYY-MM-DD', 'Threshold — hide until this date', 't:2026-02-01'],
            ['rec:N d/w/m', 'Recurrence', 'rec:1w, rec:30d, rec:3m'],
          ].map(([token, meaning, example]) => (
            <tr key={token} style={{ borderTop: '1px solid var(--term-border)' }}>
              <td style={{ padding: '6px 12px 6px 0', fontFamily: 'monospace', fontSize: '12px', color: 'var(--term-accent)', whiteSpace: 'nowrap' }}>{token}</td>
              <td style={{ padding: '6px 12px 6px 0', opacity: 0.9 }}>{meaning}</td>
              <td style={{ padding: '6px 0', fontFamily: 'monospace', fontSize: '11px', opacity: 0.7 }}>{example}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Priority</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>
        Put <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>(A)</code> at the very start of the line.
        (A) = must do, (B) = should do, (C) = nice to have.
      </p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Completed Items</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>When marked done, an <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>x</code> prefix is recorded:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        x 2026-02-25 Called dentist for appointment
      </code>

      <p style={{ marginTop: '24px', fontSize: '11px', lineHeight: '1.6', opacity: 0.7 }}>
        Data is stored locally in SQLite. Nothing is sent to any server.
      </p>
    </div>
  );
}
