import * as React from "react";
import CustomScrollbar from "./CustomScrollbar";

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
            <div style={{ fontWeight: 600, fontSize: '16px' }}>Help & Guide</div>
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
              onClick={() => setActiveTab('todotxt')}
              className={`badge ${activeTab === 'todotxt' ? 'success' : ''}`}
              style={{
                padding: '4px 8px',
                fontSize: '11px',
                borderRadius: '0 4px 4px 0',
                border: 'none',
                borderLeft: '1px solid var(--term-border)',
                opacity: activeTab === 'todotxt' ? 1 : 0.8
              }}
            >
              Todo.txt Guide
            </button>
          </div>
        </div>

        {/* Scrollable Content */}
        <CustomScrollbar style={{ flex: 1, minHeight: 0 }}>
          <div style={{ padding: '0 20px' }}>
            {activeTab === 'app' && <AppHelp />}
            {activeTab === 'todotxt' && <TodoTxtHelp />}
          </div>
        </CustomScrollbar>
      </div>
    </div>
  );
}

function AppHelp() {
  return (
    <div style={{ paddingBottom: '20px' }}>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginTop: 0, marginBottom: '12px' }}>Getting Started with PLANCK</h3>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '12px' }}>PLANCK is your simple, powerful task manager. Here's how to get started:</p>
      
      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Adding Your First Task</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '12px' }}>Click the <strong>+</strong> button or press <strong>⌘N</strong> (Cmd+N on Mac) to create a new task. Just type what you need to do!</p>
      
      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Organizing Tasks</h4>
      <ul style={{ marginLeft: '20px', fontSize: '13px', lineHeight: '1.6' }}>
        <li style={{ marginBottom: '6px' }}><strong>Priority:</strong> Add (A), (B), or (C) at the start - like "(A) Important task"</li>
        <li style={{ marginBottom: '6px' }}><strong>Projects:</strong> Add +project to group related tasks - like "+home" or "+work"</li>
        <li style={{ marginBottom: '6px' }}><strong>Contexts:</strong> Add @context for where you'll do it - like "@computer" or "@phone"</li>
        <li style={{ marginBottom: '6px' }}><strong>Tags:</strong> Add #tag for categories - like "#urgent" or "#ideas"</li>
      </ul>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Searching</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Use the search bar to filter your tasks:</p>
      <ul style={{ marginLeft: '20px', fontSize: '13px', lineHeight: '1.6' }}>
        <li style={{ marginBottom: '6px' }}>Type any word to search</li>
        <li style={{ marginBottom: '6px' }}>Type <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>+project</code> to see only that project</li>
        <li style={{ marginBottom: '6px' }}>Type <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>@context</code> to filter by context</li>
        <li style={{ marginBottom: '6px' }}>Type <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>#tag</code> to find tagged items</li>
        <li style={{ marginBottom: '6px' }}>Combine them! Try: <code style={{ fontSize: '11px', background: 'var(--term-panel)', padding: '2px 4px', borderRadius: '3px' }}>+work @computer #urgent</code></li>
      </ul>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Organizing with Sections</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>PLANCK has three sections to help you focus:</p>
      <ul style={{ marginLeft: '20px', fontSize: '13px', lineHeight: '1.6' }}>
        <li style={{ marginBottom: '6px' }}><strong>Now:</strong> What you're working on right now (3-5 tasks)</li>
        <li style={{ marginBottom: '6px' }}><strong>Soon:</strong> What's coming up next</li>
        <li style={{ marginBottom: '6px' }}><strong>Anytime:</strong> Everything else - ideas, someday tasks</li>
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
    <div style={{ paddingBottom: '20px' }}>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginTop: 0, marginBottom: '12px' }}>Todo.txt Format</h3>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '12px' }}>PLANCK uses the popular todo.txt format. It's a simple, human-readable way to organize tasks in plain text.</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Basic Format</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>A basic task looks like this:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        Call dentist for appointment
      </code>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Adding Priority</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Put (A), (B), or (C) at the very beginning:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Call dentist for appointment
      </code>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>(A) = most important, (B) = important, (C) = nice to have</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Projects (with +)</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Group related tasks with +project:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Call dentist for appointment +health
      </code>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Examples: +home, +work, +errands, +vacation</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Contexts (with @)</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Tag where or how you'll do the task with @context:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Call dentist for appointment +health @phone
      </code>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Examples: @phone, @computer, @home, @office, @errands</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Tags (with #)</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Add flexible labels with #tag:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Call dentist for appointment +health @phone #urgent
      </code>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Examples: #urgent, #waiting, #someday, #idea</p>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Due Dates</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>Add a deadline with due:YYYY-MM-DD:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Call dentist @phone due:2025-11-01
      </code>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Completed Tasks</h4>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>When done, tasks get an 'x' at the beginning:</p>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        x 2025-10-23 Called dentist for appointment
      </code>

      <h4 style={{ fontSize: '13px', fontWeight: 600, marginTop: '20px', marginBottom: '8px' }}>Complete Example</h4>
      <code style={{ display: 'block', background: 'var(--term-panel)', padding: '12px', borderRadius: '6px', margin: '12px 0', fontSize: '12px', fontFamily: 'monospace' }}>
        (A) Buy groceries for dinner party +home @errands #tonight due:2025-10-24
      </code>
      <p style={{ fontSize: '13px', lineHeight: '1.6', marginBottom: '8px' }}>This task is:</p>
      <ul style={{ marginLeft: '20px', fontSize: '13px', lineHeight: '1.6' }}>
        <li style={{ marginBottom: '6px' }}>High priority (A)</li>
        <li style={{ marginBottom: '6px' }}>Part of +home project</li>
        <li style={{ marginBottom: '6px' }}>Done while running @errands</li>
        <li style={{ marginBottom: '6px' }}>Tagged #tonight</li>
        <li style={{ marginBottom: '6px' }}>Due October 24th, 2025</li>
      </ul>

      <p style={{ marginTop: '24px', fontSize: '11px', lineHeight: '1.6', opacity: 0.8 }}>
        <strong>Why todo.txt?</strong> Your tasks are stored as plain text files. No proprietary formats, no vendor lock-in. You can edit them in any text editor, sync them anywhere, and keep them forever!
      </p>
    </div>
  );
}
