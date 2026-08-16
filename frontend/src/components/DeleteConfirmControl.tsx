type Props = {
  // isEditMode && !!onDelete — whether delete is available at all in this
  // modal instance.
  show: boolean;
  confirming: boolean;
  onRequestConfirm: () => void;
  onConfirmDelete: () => void;
};

// The footer's inline delete-confirm bar: a "Delete" button that flips to a
// "Confirm delete?" prompt + "Yes, Delete" button rather than opening a
// separate dialog. Self-contained — the two-step state lives in the parent
// (EditItemModal's showDeleteConfirm) since it needs to reset when the modal
// re-opens.
export default function DeleteConfirmControl({ show, confirming, onRequestConfirm, onConfirmDelete }: Props) {
  if (!show) return null;

  if (!confirming) {
    return (
      <button
        type="button"
        className="badge warn"
        onClick={onRequestConfirm}
        style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
      >
        Delete
      </button>
    );
  }

  return (
    <>
      <span style={{ fontSize: '12px', color: 'var(--term-dim)', marginRight: '4px' }}>Confirm delete?</span>
      <button
        type="button"
        className="badge warn"
        onClick={onConfirmDelete}
        style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
      >
        Yes, Delete
      </button>
    </>
  );
}
