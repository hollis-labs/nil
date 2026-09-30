# Security policy

## Supported versions

NIL is in public beta. Security fixes are made on `main` and in the newest
release when one is published. Older releases may not receive backports.

## Report a vulnerability

Do not include an exploit, API key, vault file, or other sensitive material in
a public issue.

Use GitHub's private vulnerability-reporting flow when the repository's
Security tab offers it. If it is unavailable, contact a repository maintainer
privately through a contact channel published on the Hollis Labs organization
or maintainer profile. Include:

- the affected version or commit and operating system
- the surface involved (GUI, local HTTP API, CLI, `nil-mcp`, chat bridge)
- reproduction steps and the security impact
- whether vault contents or keys may have been exposed
- a safe way to contact you about coordination

Maintainers will acknowledge a private report, investigate it, and coordinate
disclosure; response times are best effort.

## Deployment boundary

NIL is a single-user desktop application, not a server. Its optional local HTTP
API (also run headless with `nil serve-api`) binds `127.0.0.1` only and is
guarded by one shared secret sent in an `X-API-Key` header. `nil-mcp` reads that
key from the local configuration and talks to the API over stdio.

- Do not forward or proxy the API to another machine. It has no TLS and no
  per-caller authorization: anything holding the key can read, write and delete
  every vault.
- Anyone who can read your configuration file, or see the key in Settings, has
  full access to your vaults. Protect your user account and do not share
  screenshots of Settings.
- The API key comparison is not constant-time and CORS is permissive
  (`Access-Control-Allow-Origin: *`); both are acceptable only because of the
  loopback-only binding and would need fixing before any wider exposure. See
  `docs/architecture/ARCHITECTURE.md` §7 for the full trust-boundary analysis.
- `nil-mcp` fully trusts the local API endpoint in its configuration. Run it
  only for clients you trust with your vaults.

## Data at rest

Each vault is a plain SQLite file with no encryption, and the chat history is
a separate SQLite database. Sync is whatever synced folder you place a vault in
(iCloud Drive, Dropbox, and so on); NIL runs no sync service, and the folder
provider sees the file.

## External data processors

The only host NIL contacts is `api.anthropic.com`, from the optional chat
bridge, using an Anthropic API key you provide. Chat prompts and any vault
content the assistant is asked to read or act on are sent to Anthropic. The
chat bridge has a per-vault capability tier (read / write / delete / direct
create) that gates what the assistant may do.

## Current security limitations

- no at-rest encryption for vaults or chat history
- a single shared API key with no per-caller authorization
- no TLS on the local API (loopback only)
- pre-1.0 beta; interfaces can change
