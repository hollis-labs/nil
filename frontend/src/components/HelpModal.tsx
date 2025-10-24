import * as React from "react";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export default function HelpModal({ open, onOpenChange }: Props) {
  const [activeTab, setActiveTab] = React.useState<'app' | 'todotxt'>('app');

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
        className="custom-scrollbar"
        style={{
          background: 'var(--term-bg)',
          border: '1px solid var(--term-border)',
          borderRadius: '12px',
          padding: '32px',
          maxWidth: '700px',
          maxHeight: '80vh',
          overflow: 'auto',
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '24px' }}>
          <h2 style={{ margin: 0, color: 'var(--term-accent)', fontSize: '24px', fontWeight: 700 }}>
            Help & Guide
          </h2>
          <button
            onClick={() => onOpenChange(false)}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--term-fg)',
              cursor: 'pointer',
              fontSize: '24px',
              padding: '4px 8px',
            }}
          >
            ×
          </button>
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', gap: '8px', marginBottom: '24px', borderBottom: '1px solid var(--term-border)', paddingBottom: '8px' }}>
          <button
            className={activeTab === 'app' ? 'badge success' : 'badge'}
            onClick={() => setActiveTab('app')}
            style={{ padding: '8px 16px' }}
          >
            App Help
          </button>
          <button
            className={activeTab === 'todotxt' ? 'badge success' : 'badge'}
            onClick={() => setActiveTab('todotxt')}
            style={{ padding: '8px 16px' }}
          >
            Todo.txt Guide
          </button>
        </div>

        {/* Content */}
        <div style={{ color: 'var(--term-fg)', lineHeight: '1.8' }}>
          {activeTab === 'app' && <AppHelp />}
          {activeTab === 'todotxt' && <TodoTxtHelp />}
        </div>

        <div style={{ marginTop: '32px', textAlign: 'right' }}>
          <button
            className="badge info"
            onClick={() => onOpenChange(false)}
            style={{ padding: '10px 24px' }}
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

function AppHelp() {
  return (
    <div>
      {/* Content will be filled by sub-agent */}
      <h3 style={{ color: 'var(--term-accent)', marginTop: 0 }}>Getting Started with PLANCK</h3>
      <p>PLANCK is your simple, powerful task manager. Here's how to get started:</p>
      
      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Adding Your First Task</h4>
      <p>Click the <strong>+</strong> button or press <strong>⌘N</strong> (Cmd+N on Mac) to create a new task. Just type what you need to do!</p>
      
      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Organizing Tasks</h4>
      <ul style={{ marginLeft: '20px' }}>
        <li><strong>Priority:</strong> Add (A), (B), or (C) at the start - like "(A) Important task"</li>
        <li><strong>Projects:</strong> Add +project to group related tasks - like "+home" or "+work"</li>
        <li><strong>Contexts:</strong> Add @context for where you'll do it - like "@computer" or "@phone"</li>
        <li><strong>Tags:</strong> Add #tag for categories - like "#urgent" or "#ideas"</li>
      </ul>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Searching</h4>
      <p>Use the search bar to filter your tasks:</p>
      <ul style={{ marginLeft: '20px' }}>
        <li>Type any word to search</li>
        <li>Type <code>+project</code> to see only that project</li>
        <li>Type <code>@context</code> to filter by context</li>
        <li>Type <code>#tag</code> to find tagged items</li>
        <li>Combine them! Try: <code>+work @computer #urgent</code></li>
      </ul>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Organizing with Sections</h4>
      <p>PLANCK has three sections to help you focus:</p>
      <ul style={{ marginLeft: '20px' }}>
        <li><strong>Now:</strong> What you're working on right now (3-5 tasks)</li>
        <li><strong>Soon:</strong> What's coming up next</li>
        <li><strong>Anytime:</strong> Everything else - ideas, someday tasks</li>
      </ul>
      <p>Right-click any task and choose "Move to..." to organize it.</p>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Adding Notes</h4>
      <p>Click any task to add detailed notes. You can use basic formatting like bold, lists, and links.</p>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Quick Tips</h4>
      <ul style={{ marginLeft: '20px' }}>
        <li>Check the box to mark a task done</li>
        <li>Right-click for more actions (edit, delete, archive)</li>
        <li>Click the 🎯 icon to set a temporary focus filter</li>
        <li>Click ⚙️ to change themes and settings</li>
      </ul>
    </div>
  );
}

function TodoTxtHelp() {
  return (
    <div>
      {/* Content will be filled by sub-agent */}
      <h3 style={{ color: 'var(--term-accent)', marginTop: 0 }}>What is Todo.txt?</h3>
      <p>Todo.txt is a simple, plain-text format for managing tasks. Your tasks are stored as regular text, so you own your data forever!</p>
      
      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Basic Format</h4>
      <p>A basic task looks like this:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        Call dentist for appointment
      </code>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Adding Priority</h4>
      <p>Put (A), (B), or (C) at the very beginning:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        (A) Call dentist for appointment
      </code>
      <p>(A) = most important, (B) = important, (C) = nice to have</p>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Projects (with +)</h4>
      <p>Group related tasks with +project:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        (A) Call dentist for appointment +health
      </code>
      <p>Examples: +home, +work, +errands, +vacation</p>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Contexts (with @)</h4>
      <p>Tag where or how you'll do the task with @context:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        (A) Call dentist for appointment +health @phone
      </code>
      <p>Examples: @phone, @computer, @home, @office, @errands</p>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Tags (with #)</h4>
      <p>Add flexible labels with #tag:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        (A) Call dentist for appointment +health @phone #urgent
      </code>
      <p>Examples: #urgent, #waiting, #someday, #idea</p>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Due Dates</h4>
      <p>Add a deadline with due:YYYY-MM-DD:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        (A) Call dentist @phone due:2025-11-01
      </code>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Completed Tasks</h4>
      <p>When done, tasks get an 'x' at the beginning:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        x 2025-10-23 Called dentist for appointment
      </code>

      <h4 style={{ color: 'var(--term-accent)', marginTop: '20px' }}>Complete Example</h4>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0' }}>
        (A) Buy groceries for dinner party +home @errands #tonight due:2025-10-24
      </code>
      <p>This task is:</p>
      <ul style={{ marginLeft: '20px' }}>
        <li>High priority (A)</li>
        <li>Part of +home project</li>
        <li>Done while running @errands</li>
        <li>Tagged #tonight</li>
        <li>Due October 24th, 2025</li>
      </ul>

      <p style={{ marginTop: '24px', fontSize: '13px', opacity: 0.8 }}>
        <strong>Why todo.txt?</strong> Your tasks are stored as plain text files. No proprietary formats, no vendor lock-in. You can edit them in any text editor, sync them anywhere, and keep them forever!
      </p>
    </div>
  );
}
