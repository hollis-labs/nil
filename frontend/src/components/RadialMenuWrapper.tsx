import * as React from 'react';
import { TodoRow } from './TerminalList';
import { 
  ArrowUp, 
  ArrowRight, 
  ArrowDown, 
  Trash2, 
  Archive, 
  FileEdit, 
  CheckCircle2, 
  MoreHorizontal,
  Copy,
  Tag,
  Pin
} from 'lucide-react';

interface RadialMenuWrapperProps {
  todo: TodoRow;
  position: { x: number; y: number };
  onMoveSection: (id: number, section: string) => void;
  onDelete: (id: number) => void;
  onArchive: (id: number, archived: boolean) => void;
  onToggle: (id: number, completed: boolean) => void;
  onEdit: (todo: TodoRow) => void;
  onClone: (todo: TodoRow) => void;
  onMeta: (todo: TodoRow) => void;
  onPin: (id: number, pinned: boolean) => void;
  onClose: () => void;
}

export default function RadialMenuWrapper({
  todo,
  position,
  onMoveSection,
  onDelete,
  onArchive,
  onToggle,
  onEdit,
  onClone,
  onMeta,
  onPin,
  onClose
}: RadialMenuWrapperProps) {
  const menuRef = React.useRef<HTMLDivElement>(null);
  const [activeLayer, setActiveLayer] = React.useState<'main' | 'more'>('main');

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (activeLayer === 'more') {
          setActiveLayer('main');
        } else {
          onClose();
        }
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [activeLayer, onClose]);

  const canMoveToNow = todo.section !== 'now';
  const canMoveToSoon = todo.section !== 'soon';
  const canMoveToAnytime = todo.section !== 'anytime';

  const handleAction = (action: string) => {
    switch (action) {
      case 'now':
        if (canMoveToNow) {
          onMoveSection(todo.id, 'now');
          onClose();
        }
        break;
      case 'soon':
        if (canMoveToSoon) {
          onMoveSection(todo.id, 'soon');
          onClose();
        }
        break;
      case 'anytime':
        if (canMoveToAnytime) {
          onMoveSection(todo.id, 'anytime');
          onClose();
        }
        break;
      case 'delete':
        onDelete(todo.id);
        onClose();
        break;
      case 'edit':
        onEdit(todo);
        onClose();
        break;
      case 'complete':
        onToggle(todo.id, !todo.completed);
        onClose();
        break;
      case 'more':
        // Force immediate state update
        setActiveLayer('more');
        break;
      case 'archive':
        onArchive(todo.id, true);
        onClose();
        break;
      case 'clone':
        onClone(todo);
        onClose();
        break;
      case 'meta':
        onMeta(todo);
        onClose();
        break;
      case 'pin':
        onPin(todo.id, !(todo as any).pinned);
        onClose();
        break;
      case 'back':
        setActiveLayer('main');
        break;
    }
  };

  const mainMenuItems = [
    { id: 'now', label: 'NOW', icon: ArrowUp, disabled: !canMoveToNow, angle: 0 },
    { id: 'soon', label: 'SOON', icon: ArrowRight, disabled: !canMoveToSoon, angle: 45 },
    { id: 'anytime', label: 'ANY', icon: ArrowDown, disabled: !canMoveToAnytime, angle: 90 },
    { id: 'delete', label: 'DEL', icon: Trash2, disabled: false, angle: 135 },
    { id: 'edit', label: 'EDIT', icon: FileEdit, disabled: false, angle: 180 },
    { id: 'complete', label: 'DONE', icon: CheckCircle2, disabled: false, angle: 225 },
    { id: 'more', label: 'MORE', icon: MoreHorizontal, disabled: false, angle: 270 },
    { id: 'pin', label: 'PIN', icon: Pin, disabled: false, angle: 315 },
  ];

  const moreMenuItems = [
    { id: 'archive', label: 'ARCH', icon: Archive, disabled: false, angle: 135 },
    { id: 'clone', label: 'COPY', icon: Copy, disabled: false, angle: 180 },
    { id: 'meta', label: 'META', icon: Tag, disabled: false, angle: 225 },
  ];

  const menuItems = activeLayer === 'main' ? mainMenuItems : moreMenuItems;
  const radius = 60;
  const centerRadius = 35;

  return (
    <>
      <style>{`
        @keyframes radialFadeIn {
          from {
            opacity: 0;
            transform: translate(-50%, -50%) scale(0.8);
          }
          to {
            opacity: 1;
            transform: translate(-50%, -50%) scale(1);
          }
        }

        .radial-menu-overlay {
          position: fixed;
          inset: 0;
          background: rgba(0, 0, 0, 0.15);
          backdrop-filter: blur(1px);
          z-index: 9999;
        }

        .radial-menu-container {
          position: fixed;
          animation: radialFadeIn 0.2s ease-out;
        }

        .radial-menu-item {
          position: absolute;
          cursor: pointer;
          transition: none;
          pointer-events: auto;
          will-change: transform, opacity;
        }
        
        .radial-menu-item.hidden {
          display: none !important;
        }
        
        .layer-wrapper {
          position: absolute;
          inset: 0;
          will-change: contents;
        }

        .radial-menu-item.disabled {
          opacity: 0.3;
          cursor: not-allowed;
        }

        .item-button {
          width: 35px;
          height: 35px;
          border-radius: 50%;
          background: var(--term-bgAlt);
          border: 1px solid var(--term-border);
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 4px;
          color: var(--term-fg);
          font-size: 8px;
          font-weight: 600;
          letter-spacing: 0.5px;
          transition: all 0.15s ease;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
        }

        .radial-menu-item:not(.disabled):hover .item-button {
          background: var(--term-info);
          border-color: var(--term-info);
          color: var(--term-bg);
          transform: scale(1.1);
          box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
        }

        .center-button {
          width: 35px;
          height: 35px;
          border-radius: 50%;
          background: var(--term-bgAlt);
          border: 1px solid var(--term-border);
          display: flex;
          align-items: center;
          justify-content: center;
          color: var(--term-fg);
          font-size: 14px;
          font-weight: 700;
          cursor: pointer;
          transition: all 0.15s ease;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          z-index: 100;
        }

        .center-button:hover {
          background: var(--term-info);
          border-color: var(--term-info);
          color: var(--term-bg);
          transform: translate(-50%, -50%) scale(1.1);
          box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
        }
      `}</style>

      <div className="radial-menu-overlay" onClick={onClose}>
        <div 
          className="radial-menu-container" 
          ref={menuRef}
          style={{
            left: `${position.x}px`,
            top: `${position.y}px`,
            transform: 'translate(-50%, -50%)'
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {activeLayer === 'main' && (
            <div className="layer-wrapper">
              {mainMenuItems.map((item, index) => {
                const angle = (item.angle * Math.PI) / 180;
                const x = Math.cos(angle) * radius;
                const y = Math.sin(angle) * radius;
                const Icon = item.icon;

                return (
                  <div
                    key={item.id}
                    className={`radial-menu-item ${item.disabled ? 'disabled' : ''}`}
                    style={{
                      left: `calc(50% + ${x}px)`,
                      top: `calc(50% + ${y}px)`,
                      transform: 'translate(-50%, -50%)',
                    }}
                    onClick={() => !item.disabled && handleAction(item.id)}
                    onMouseDown={(e) => e.stopPropagation()}
                  >
                    <div className="item-button">
                      <Icon size={12} />
                      <span>{item.label}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {activeLayer === 'more' && (
            <div className="layer-wrapper">
              {moreMenuItems.map((item, index) => {
                const angle = (item.angle * Math.PI) / 180;
                const x = Math.cos(angle) * radius;
                const y = Math.sin(angle) * radius;
                const Icon = item.icon;

                return (
                  <div
                    key={item.id}
                    className={`radial-menu-item ${item.disabled ? 'disabled' : ''}`}
                    style={{
                      left: `calc(50% + ${x}px)`,
                      top: `calc(50% + ${y}px)`,
                      transform: 'translate(-50%, -50%)',
                    }}
                    onClick={() => !item.disabled && handleAction(item.id)}
                    onMouseDown={(e) => e.stopPropagation()}
                  >
                    <div className="item-button">
                      <Icon size={12} />
                      <span>{item.label}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          <div 
            className="center-button"
            onClick={() => activeLayer === 'more' ? handleAction('back') : onClose()}
          >
            {activeLayer === 'more' ? '←' : '×'}
          </div>
        </div>
      </div>
    </>
  );
}
