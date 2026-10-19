// Package capture implements Nil v2 draft custody (CW-20261018-0077).
//
// Host decision: retain Wails v2, Go and React/TypeScript. These are already
// packaged together in Nil, provide generated typed IPC and backend-owned
// persistence, and allow the existing --term-* theme to remain compatible with
// the portfolio UI direction without adding another desktop runtime. Mac
// invocation, signing and native focus remain verification gates in 0081.
//
// The source editor uses plain Markdown rather than normalizing through TipTap
// JSON: unsupported syntax must survive byte-for-byte. Nanite's composer is
// interaction prior art, especially keyboard suggestion arbitration and focus;
// its command/shell execution is not part of capture. Rich interactions belong
// in 0078 and operate on source. The browser UI calls a small storage adapter;
// this package has no Wails dependency and can be tested without the shell.
//
// capture.db is independent of every legacy vault. No existing notes_doc,
// notes_md, notes_html, item, session profile or vault schema is rewritten or
// imported. Rollback to the prior application leaves those vaults usable and
// retains capture.db for later recovery. A future explicit import must preserve
// originals. The new database has its own schema version on each source row;
// unknown versions are rejected rather than silently normalized. Future table
// changes need an explicit capture migration, separate from vault migrations.
//
// HTML is an escaped source preview, rebuilt on read with content hash and
// renderer version, never a source of Markdown. There are no active links,
// remote images or HTML execution in this renderer. No network request is made
// by this package. Drafts remain local until 0080 adds immutable submission and
// FE confirmation; creating or saving a draft does not imply broker acceptance.
package capture
