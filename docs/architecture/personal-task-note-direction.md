# NIL Personal Task and Note Management Direction

**Status:** Directional architecture draft

**Date:** 2026-08-22
**Scope:** Desired product boundaries and architecture; not an implementation plan

## Purpose

This document describes the intended place of NIL in the Hollis Labs
portfolio. It evaluates the unified task/note model, vault custody, inbox and
triage, rich content, taxonomy, references, search, recurrence, external
access, AI assistance, addons, sync, credentials, privacy, persistence,
observability, and portfolio composition against the shared engineering
boundaries.

It deliberately does not define phases, estimates, task breakdowns, migration
steps, or compatibility sequencing. Existing code, accepted ADRs, and
`docs/architecture/ARCHITECTURE.md` remain the current implementation truth
until explicitly superseded. This document defines a desired state for a later
architecture and planning session.

NIL differs from most applications in this portfolio. It is not primarily an
engine operating on another application's records. It is a user-facing domain
product deliberately entrusted with personal tasks, notes, and organizational
state. The portfolio axiom still applies, but custody is central rather than
incidental.

## Portfolio axiom

> Hollis tools own execution and operational state, but not the business
> definitions or business data they operate on.

For NIL, the correct interpretation is:

- The user owns the meaning, sensitivity, priority, truth, and desired
  disposition of their tasks, notes, relationships, schedules, and settings.
- A vault owner controls the vault's membership, policy, retention, sharing,
  external access, and encryption posture.
- Addon publishers own kind definitions, view definitions, automation rules,
  templates, and integration behavior they publish.
- External applications remain authoritative for externally bound records.
- Credential authorities own model-provider keys, sync credentials, and other
  third-party secrets.
- NIL owns the durable custodial and operational record explicitly placed in a
  vault: item identity, immutable revisions, current heads, task state,
  taxonomy assignments, links, triage state, local schedules, action proposals,
  user decisions, external bindings, sync facts, derived indexes, and audit.

NIL can be the operational system of record for a user's task or note while
the user remains its semantic authority. The user may authorize NIL to mutate,
sync, render, remind, or expose that record. Those permissions are explicit and
revocable; storage alone does not grant every future feature access.

The useful shorthand is:

> The user owns the life represented in a vault; NIL owns faithful custody and
> execution of the user's declared intent.

## Product definition

NIL is a local-first, keyboard-driven personal task and note workspace. It:

1. Captures tasks, notes, and scratch material through one low-friction item
   model.
2. Lets an item acquire task, note, scheduling, linking, and addon-specific
   affordances without splitting the user's information across unrelated
   products.
3. Preserves user-authored content, item revisions, task state, taxonomy,
   references, and provenance inside explicitly selected vaults.
4. Provides inbox-based capture and deliberate triage without requiring
   taxonomy at entry time.
5. Supports fast deterministic retrieval, linked navigation, saved views, and
   optional deeper search over vault-authorized content.
6. Provides local task timing, recurrence, reminders, and personal planning
   semantics without becoming a general workflow engine.
7. Exposes the same domain through the desktop GUI, CLI, loopback API, MCP, and
   narrow integration contracts.
8. Allows AI assistance to search, synthesize, and propose changes inside
   explicit vault capabilities, with user authority over adoption.
9. Keeps vaults portable, inspectable, backupable, and independently useful
   without portfolio services or a network connection.

NIL is not:

- a team project-management or work-dispatch system;
- the canonical portfolio memory or knowledge service;
- a general fragment-ingestion and routing engine;
- a content-authoring and publication pipeline;
- a durable general-purpose workflow engine;
- an agent runtime, LLM gateway, or conversation platform;
- a general messaging or broadcast service;
- a calendar provider of record;
- a contact CRM merely because an addon can display contacts;
- a secret manager or credential authority for third-party providers;
- an infrastructure or local-daemon control plane;
- a cloud storage provider.

The product remains useful as a desktop application with a single local vault,
deterministic search, no AI provider, no API server, and no other Hollis Labs
application installed.

## Core product boundary: one item, several affordances

The product thesis remains foundational:

> Tasks and notes are the same object with different affordances.

This does not require every kind to share every physical column or lifecycle.
It means the user gets one identity, capture model, body model, taxonomy,
linking system, provenance model, and retrieval surface. Task behavior,
recurrence, note presentation, journal metadata, contact fields, or addon state
can be typed facets attached to that identity.

The logical model should therefore be:

```text
Item identity
  + immutable content revision
  + current adopted head
  + zero or more typed facets
  + operational state and relationships
  + provenance, policy, and audit
```

`todo`, `note`, and `scratch` remain excellent default affordance presets.
They are not separate authorities and do not need separate applications.
Changing a kind changes declared behavior and presentation; it does not create
a different item identity or silently discard content.

## Architecture sketch

```text
User / vault owner
  meaning · intent · policy · approvals · retention · sharing
                         |
                         v
              +-------------------------+
 Desktop UI ->|                         |<- CLI
 Wails IPC -->|           NIL           |<- loopback HTTP
              |                         |<- MCP adapter
              | item application svc    |
              | vault authority/policy  |
              | revisions + task state  |
              | taxonomy + references   |
              | search + projections    |
              | proposals + audit       |
              +------------+------------+
                           |
                    one explicit vault
                           |
              +------------v------------+
              | SQLite operational store|
              | content/artifact storage|
              | revision + sync journal |
              +------------+------------+
                           |
           checkpoint / replication / export contract
                           |
                backup or sync destination

External app ---- scoped request / source binding ----> NIL
NIL ------------ event / delivery receipt -----------> external app

Nanite/Tether/Hadron/Torque/Tesseract/Fragments/Loom are optional peers.
No peer obtains vault authority by being installed or reachable.
```

An item, item revision, task state, inbox placement, saved view, session
filter, chat session, model call, action proposal, approval, automation run,
sync operation, external binding, API principal, and vault are distinct
identities and lifecycles.

## Responsibility boundaries

| Concern | Authoritative owner | NIL responsibility |
|---|---|---|
| Item meaning and content | User or delegated vault principal | Preserve content, revisions, provenance, and authorized changes |
| Vault membership and policy | Vault owner | Enforce boundaries, capabilities, retention, and access |
| Item identity and current head | NIL for the entrusted vault record | Maintain stable identity, immutable revisions, and current projection |
| Task completion and personal planning state | User through NIL | Record state transitions, recurrence, reminders, and audit |
| External business object | External application | Store qualified binding and observed revision; do not silently overwrite |
| Kind and facet definition | Core or addon publisher | Validate, register, resolve, and apply exact revision |
| Taxonomy meaning | User or vault | Store labels and relationships without imposing a universal ontology |
| Item references | User for intentional links; processor for derived links | Preserve typed edges, provenance, and integrity |
| Rich-text source | User | Store canonical structured document and generate projections |
| Search result | NIL | Compute a derived view under vault authorization |
| Embedding or model-derived metadata | Producing model/processor | Attribute, version, expire, and never treat as source truth |
| AI suggestion | Model or agent as producer | Preserve candidate and evidence; require policy-authorized adoption |
| Human approval or denial | User | Capture, authenticate, apply, and audit |
| Personal reminder execution | NIL when enabled | Schedule, notify, retry, and record outcome |
| General workflow execution | Hadron | Invoke and correlate when composition is justified |
| Managed project work | Torque | Link or project without duplicating authority |
| Portfolio knowledge and memory | Tesseract namespace authority | Promote explicitly; maintain only vault-local recall |
| Fragment intake and routing | Fragments Engine | Accept deliberate deliveries or provide user-selected exports |
| Content publishing | Loom and destination authorities | Supply selected sources or accept explicit returned artifacts |
| Agent session | Nanite or another launcher | Request a bounded session and consume proposals when composed |
| Gateway and general messaging | Tether | Use optional transport without importing its domain |
| Third-party credentials | External credential authority | Store references and consume narrow runtime grants |
| NIL access credentials | NIL | Issue, scope, rotate, revoke, and verify without storing plaintext verifiers |
| Process supervision | Desktop OS or service manager | Start/stop cleanly and expose health in headless mode |

## Authority and custody model

### User-authored item

A user-authored item is deliberately created or adopted into a vault. The user
owns its semantics. NIL provides durable custody and becomes authoritative for
the item's recorded revision history and operational state in that vault.

```text
Item
  stable identity
  vault and owning principal
  current adopted revision
  kind and facet references
  lifecycle and retention policy
  external bindings
  creation, archival, and deletion facts
```

### External observation or binding

An item imported or synchronized from another application retains a qualified
source binding:

```text
ExternalBinding
  source authority and connector
  external object identity
  observed or accepted source revision
  locator and digest
  sync direction and conflict policy
  field authority map when required
  last successful exchange and receipt
```

An external reference is not a global idempotency string with unspecified
meaning. Authority, connector, object identity, and revision are separate.
Re-delivery creates an observed change or proposed revision under the binding's
policy; it does not silently replace user edits.

### Derived projections

HTML previews, plain text, FTS rows, embeddings, result caches, counts,
attention views, and rendered exports are projections. They record producer
and version when needed and are safe to rebuild from durable content.

Projection failure must not make the canonical item unreadable. Corrupt cached
HTML or an unavailable embedding provider cannot block access to the structured
document and revision history.

### AI and automation results

An AI answer, suggested edit, classification, extracted date, or automation
result is an attributed proposal or observation. It becomes item state only
through an explicit policy-authorized transition.

The model is authoritative for what it emitted. NIL is authoritative for the
request it made, the output it observed, and whether a user or policy adopted
it. Neither fact makes the model's interpretation semantically true.

### Replicas and exports

A synced copy, todo.txt export, Markdown rendering, or backup is not implicitly
a second writable authority. Every external representation declares one mode:

- managed projection from NIL;
- read-only backup/checkpoint;
- bidirectional replica under an explicit merge protocol;
- transferred export whose future authority belongs to the receiver;
- external authoritative source re-ingested through a binding.

File location alone does not determine ownership.

## Core domain model

### Vault

A vault is an explicit custody, privacy, and authorization boundary:

```text
Vault
  stable identity independent of path
  owner and authorized principals
  display name and local storage binding
  policy revision
  encryption and key reference
  sync and backup registrations
  schema and application compatibility
  creation, lock, archive, and deletion facts
```

A vault path is a deployment binding. Moving a directory does not create a new
vault. Copying the directory creates a replica or fork that needs its own
identity relationship; it must not silently appear as the same concurrent
writer.

`project`, `work`, `personal`, or other labels may describe use but do not
replace owner identity or policy.

### Item identity and revision

Stable item identity is separate from SQLite row number and vault-local
storage position:

```text
ItemRevision
  item identity
  immutable revision identity and sequence
  parent or merge-parent revisions
  title and canonical body reference
  kind and facet payloads
  taxonomy and explicit relation changes
  actor, reason, and provenance
  content digest and timestamp
```

The current adopted head is a small mutable projection referencing one
revision. Concurrent edits use expected revision, conflict detection, and an
explicit merge or choice rather than last-write-wins overwrite.

Stable IDs survive vault moves, export/import round trips, sync, API use, and
reference creation. Human-friendly local numbers can remain display aliases.

### Canonical body and representations

The TipTap/ProseMirror JSON document is a sound canonical rich-text
representation for current editing. Its contract should be explicit:

- schema and extension versions are recorded;
- unknown nodes and marks are preserved or rejected visibly, never silently
  discarded;
- document validation occurs at write boundaries;
- format conversion records loss or warnings;
- attachments and embedded content use stable artifact references;
- plain text, HTML, and Markdown are derived unless an import mode says
  otherwise;
- editor upgrades have deterministic migration and rollback semantics.

`notes_html` remains a versioned rebuildable cache. Plain text remains a
derived search and interchange view. A Markdown exporter should promise only
the fidelity its profile can preserve.

### Kinds and facets

Kinds choose affordances and default views. Facets carry typed behavior:

```text
KindDefinition
  publisher-qualified identity and immutable revision
  display and editor capabilities
  allowed or required facets
  validation and projection schemas
  migration compatibility
  declared effects and trust evidence

TaskFacet
  completion policy and state
  priority
  due, threshold, recurrence, and reminder references

NoteFacet
  note-specific presentation or structure hints
```

The unified item model remains logical even if facet data moves into separate
tables or extension payloads. This avoids stuffing every future addon field
into one wide row while preserving one identity and retrieval system.

Changing kind is a validated revision and facet transition. Addon removal does
not delete content; unknown facet data stays preservable and inspectable.

### Taxonomy and relationships

Projects, contexts, and tags are user-owned taxonomies. NIL stores stable
taxonomy identities plus names and aliases so renaming a label does not rewrite
item history or break references.

Item links are typed edges:

```text
ItemRelation
  stable edge identity
  source and target item identities
  relation definition
  intentional or derived classification
  actor/processor and provenance
  creation and removal facts
```

Wikilinks are one authoring syntax for an intentional relation. A display label
is not the target identity. Backlinks are a derived reverse view.

Cross-vault references require explicit authorization and a portable locator.
When the target vault is locked or absent, NIL preserves the unresolved link
without leaking target content.

### Inbox and triage

Inbox is an operational placement and attention condition, not an item kind or
separate semantic authority.

```text
Capture
  item identity and initial revision
  target vault or unresolved custody intent
  capture source and principal
  inbox placement and received time
  proposed taxonomy or routing observations
```

Triage can select a vault, kind, taxonomy, task state, or disposition. Moving an
item between stores is a durable transfer with source and target identities,
idempotency, recovery state, and a receipt. Copy-then-delete across SQLite files
is not atomic and should never be presented as one guaranteed transaction.

The shared inbox may be modeled as a special vault or explicit staging store,
but its authority and backup behavior must be as clear as ordinary vaults.

### Views and sessions

Now, Soon, Anytime, saved tabs, search filters, and session contexts are views
over item state, not alternative item stores.

```text
ViewDefinition
  owner and scope
  versioned query/filter definition
  ordering and grouping
  presentation hints

FocusSession
  identity and owning user/device
  active view and inherited taxonomy policy
  start/end and optional persistence policy
```

Device-only presentation preferences can remain local. Saved views, session
profiles, templates, and other user-created definitions that users expect to
travel with a vault belong in durable, exportable storage. Browser localStorage
must not silently become the only record of meaningful user configuration.

### Task state, recurrence, and reminders

Task semantics are distinct from content revisions:

```text
TaskState
  item identity
  open/completed/cancelled state
  current section and attention facts
  completion actor and timestamp
  state transition history

RecurrenceDefinition
  publisher/user-owned rule revision
  timezone and calendar semantics
  occurrence-generation policy
  completion/catch-up behavior

ReminderIntent
  item and occurrence reference
  desired trigger and channel
  delivery policy and status
```

Editing the title creates a content revision. Completing an occurrence creates a
task-state fact. Updating a recurrence rule creates a new rule revision. These
can be presented as one item in the UI without conflating their histories.

NIL may own personal reminder scheduling and local notifications. It does not
become a general durable workflow engine or external calendar authority.

### External binding and action proposal

External writes and AI writes share a safe conceptual boundary:

```text
MutationProposal
  stable identity and idempotency key
  proposing principal/adapter/model
  target vault, item, and expected revision
  typed operation and candidate changes
  reason, evidence, and source binding
  required capability and decision policy
  lifecycle and expiry

MutationDecision
  authenticated decision principal
  approve, deny, amend, or expire
  policy revision and evidence
  timestamp and applied result reference
```

Direct execution is a policy mode, not a special untracked path. Even an
auto-approved action produces the proposal, policy decision, resulting
revision, and audit evidence.

## Lifecycle separation

| Lifecycle | Example states | Owner |
|---|---|---|
| Item content | working, adopted, superseded, conflicted | NIL custody under user authority |
| Task | open, completed, reopened, cancelled | User through NIL |
| Inbox | captured, triaged, deferred, discarded | User through NIL |
| Vault | available, locked, read-only, unavailable, archived | Vault owner/NIL custody |
| Addon definition | discovered, trusted, active, disabled, incompatible | Publisher for definition; NIL for registration |
| AI/chat session | active, ended, retained, deleted | NIL for native bounded chat or Nanite when delegated |
| Mutation proposal | pending, approved, denied, expired, applied, failed | NIL under user policy |
| Automation run | queued, running, waiting, completed, failed | NIL if bounded; Hadron if delegated |
| Reminder | scheduled, due, delivered, acknowledged, missed | NIL or registered reminder adapter |
| Sync operation | discovered, transferring, merged, conflicted, complete | NIL sync boundary |
| External object | provider-specific | External authority |

One lifecycle may affect another but cannot impersonate it:

- editing content does not complete a task;
- completing a task does not delete or archive its note body;
- moving out of inbox does not transfer semantic ownership;
- a model call succeeding does not approve its mutation;
- an API request being authenticated does not authorize every vault or effect;
- an automation run completing does not prove user intent was satisfied;
- a sync upload does not prove every replica adopted the revision;
- an addon being installed does not grant access to all vaults.

## Vault isolation and multi-vault behavior

Per-vault storage is a valuable product boundary to preserve:

- users can separate work, personal, shared, or sensitive contexts;
- a vault can be moved, backed up, exported, locked, or deleted independently;
- external principals can receive grants to one vault without learning others;
- unavailable vaults do not prevent the application from opening another;
- cross-vault queries and links are explicit rather than accidental.

Isolation must extend beyond the main item table. Chat transcripts, proposals,
tool results, embeddings, attachments, search caches, audit records, and sync
metadata either belong inside the vault's custody domain or carry an explicit
vault binding and equivalent policy in an application-level store.

The active vault is a UI selection, not a safe fallback for a failed identity
lookup. A request naming an unknown or unauthorized vault must fail closed. It
must never operate on the active vault instead.

Moving an item between vaults requires authorization for both sides, stable
identity policy, link handling, conflict behavior, and crash recovery. A move
may preserve identity under one owner or create a new identity with lineage
when crossing ownership boundaries.

## Sync, cloud folders, backup, and recovery

Cloud folder storage is not itself a sync protocol. SQLite databases in WAL
mode can have multiple files and active writer state; consumer file-sync tools
can copy those files at different times, create conflicts, or present an older
snapshot as current.

The target distinguishes:

- **local operational store:** the database currently owned by a NIL writer;
- **checkpoint:** a transactionally consistent closed or SQLite-produced
  snapshot;
- **backup:** an immutable checkpoint plus manifest and integrity metadata;
- **replica:** another writable instance participating in a merge protocol;
- **export:** a transferred representation with declared fidelity;
- **folder transport:** a mechanism for moving checkpoints or replication
  records, not proof of consistency.

True multi-device sync needs stable record/revision identities, tombstones,
causal or ordered change metadata, conflict detection, deterministic merge
rules for safe fields, and explicit user resolution for semantic conflicts.
Last writer wins is not a sufficient universal policy for notes, completion,
recurrence, taxonomy, or references.

If a user chooses simple file sync, NIL should declare the supported operating
contract: checkpoint/close rules, lock detection, conflict-copy handling,
integrity verification, and recovery. It should not imply that two concurrently
open apps safely share one live SQLite/WAL set through a cloud folder.

Backup scope includes the vault database, non-reconstructible attachments,
revision content, audit and tombstones, vault-owned definitions, encryption
metadata, and any chat or proposal records covered by vault retention. Device
preferences, FTS, HTML caches, embeddings, and ephemeral tool caches are
rebuildable.

Restore is non-destructive by default: inspect, validate, and restore to a new
vault or explicit replacement operation. Recovery tools operate on closed,
resolved paths and preserve the original before material change.

## Search and recall

FTS5 is the deterministic default and should remain available offline. Search
operates within one explicitly authorized vault unless the user chooses a
multi-vault query and policy allows it.

The retrieval model distinguishes:

- exact and structured filtering;
- full-text relevance;
- fuzzy matching;
- graph traversal through explicit references;
- semantic retrieval over derived embeddings;
- model-produced synthesis over selected results.

Embeddings and model summaries are attributed projections. They record model,
version, input revision, sensitivity policy, and invalidation. They do not
replace canonical content or make NIL a portfolio memory authority.

Search results are deterministic where the same store and index state permit
it, with explicit stable tie-breaking. A speed/depth control selects a declared
retrieval plan rather than silently changing providers or data egress.

Queries, results, and excerpts may be highly sensitive. They do not enter logs,
traces, analytics, or shared caches by default. Cross-process result caches are
scoped by verified principal, vault, policy, and TTL.

## AI chat and automation boundary

NIL owns the user experience and domain policy for “help me with this vault.”
It does not need to own a general agent runtime.

Three execution shapes can coexist behind one proposal/result contract:

- deterministic local helper functions;
- a bounded, provider-neutral model/tool loop embedded for standalone use;
- a delegated Nanite agent or Hadron workflow when richer execution is
  composed.

The application selects the least expansive mechanism that satisfies the
declared operation. An embedded loop remains bounded by turns, tools, vault,
data egress, and effects. It uses provider-neutral contracts and prefers
official SDKs over a hand-maintained wire protocol.

The current Propose → Approve/Deny → Execute model is directionally strong and
should become the common mutation boundary for model, automation, and external
writer proposals.

Capabilities should express at least:

- vault and item scope;
- read metadata versus read body;
- search and relationship traversal;
- propose create/update/delete/move;
- direct create/update under bounded rules;
- export or external delivery;
- model egress and provider class;
- maximum result/body size and time;
- expiry and revocation.

Read-only is the default. Delete, cross-vault move, export, and external
delivery are distinct effects. A broad `Write` flag does not imply all of them.

Approval binds the exact candidate, target item revision, vault, and policy.
Changing the proposal after approval invalidates the decision. Execution uses
expected-revision checks and produces an immutable audit outcome.

Chat transcript and tool-call retention are explicit per vault/session. Tool
results can contain complete private notes and must not be retained in a global
application database or telemetry stream without declared custody.

## Interfaces and parity

NIL follows “one core, several doors”:

- **Wails GUI:** trusted same-process desktop interaction;
- **CLI:** explicit user scripting and one-off administration;
- **loopback HTTP:** inter-process API and headless service surface;
- **MCP:** agent-friendly adapter over the application API;
- **events:** change and attention signals, not a second command model;
- **sync/portfolio clients:** narrow typed integrations.

All surfaces route through one application service and authorization model.
Transport differences do not justify different item defaulting, note
conversion, vault resolution, proposal semantics, or audit.

The Wails bridge shares process authority with the desktop backend, but
frontend code is still untrusted input at validation boundaries. Generated
bindings provide shape, not authorization or semantic validation.

The HTTP API exposes stable domain resources, revision preconditions,
idempotency, typed errors, pagination, tombstones/change feeds, principal and
vault capability discovery, and explicit batch semantics.

MCP remains a thin semantic adapter rather than a database owner. It obtains a
verified principal and scoped token from the hosting transport or NIL. It does
not learn full authority merely by reading a shared config file.

CLI behavior has two explicit modes:

- proxy to the active NIL service that owns the vault;
- offline administration against a closed vault with exclusive ownership.

A short-lived CLI silently opening a vault already owned by another process is
not the default coordination model.

## Authentication and authorization

Loopback is a network boundary, not an identity system. Any local process can
connect to `127.0.0.1`, and browser pages can initiate requests subject to
browser controls. Trust must come from authenticated principals and scoped
capabilities.

NIL may own credentials whose sole purpose is authorizing NIL access. It should:

- issue a distinct credential or capability per client/integration;
- store only a verifier or OS-protected credential material where possible;
- show a secret once rather than return it through ordinary settings reads;
- support scope, vault restrictions, read/write effects, expiry, rotation, and
  revocation;
- compare verifiers safely;
- record principal and grant on every external mutation;
- reject unknown vault IDs rather than falling back to the active vault;
- use narrow CORS policy or no browser CORS unless a registered client needs it;
- rate-limit and bound requests that expose private content.

The Wails same-process channel can use the logged-in OS user and application
process as its local trust root, but destructive and AI-mediated operations may
still require user confirmation under product policy.

Vault policy is evaluated after authentication. Possession of a valid NIL API
credential does not imply access to every vault or every operation.

## Secrets, encryption, and privacy

NIL must not become the credential store for Anthropic, sync providers, Tether,
or other third parties. It stores credential references and consumes narrow
runtime grants from the OS keychain or another external authority.

```text
credential://nil/provider/anthropic
secret://os-keychain/nil/sync-provider
grant://session/vault/read-body
```

Provider API keys in cleartext `config.json` are a current implementation
compromise, not the target. API access secrets should likewise move out of
ordinary readable configuration; NIL may own their issuance and verifier
because they authorize NIL itself.

Vault encryption is a custody policy, not a place for NIL to invent key
escrow. The user owns the encryption authority. NIL may:

- create or consume a key through an OS credential service;
- unlock a vault for the current user/session;
- materialize the key only at the storage boundary;
- support explicit recovery material chosen by the user;
- record encryption algorithm and key-version metadata;
- rotate and re-encrypt under an explicit operation;
- lock, clear memory, and fail closed when authorization expires.

NIL should never promise encryption without a threat model. At minimum, the
model distinguishes data at rest, active process memory, OS compromise,
cloud-provider access, backups, telemetry, external model egress, and shared
device users.

Privacy defaults:

- no item bodies, search queries, chat prompts, model results, API tokens, or
  vault paths in telemetry;
- local search and deterministic features do not require network access;
- model egress is per vault and visible before first use;
- source excerpts are minimized to the operation;
- logs and support bundles are redacted;
- clipboard, notifications, and OS previews respect sensitivity policy;
- retention and deletion apply to chat, audit, caches, backups, and replicas as
  explicitly defined rather than only the main item row.

## Addon and plugin model

NIL's addon system should expand personal organization without turning the
desktop process into an ambient plugin runtime.

Useful extension classes include:

- item kinds and typed facets;
- editors and renderers;
- view and query definitions;
- import/export codecs;
- taxonomy and relation definitions;
- deterministic classifiers and triage helpers;
- reminder and notification adapters;
- external source/sync connectors;
- model providers and bounded automation actions;
- UI contributions through declared slots.

An addon manifest declares publisher identity, version, digest, contract
versions, kind/facet schemas, migrations, configuration schema, capabilities,
effects, credential references, UI contributions, compatibility, diagnostics,
and trust evidence.

Plugins declare; NIL validates and grants. Declared, granted, and used effects
are distinct. Installation does not imply activation in every vault.

Addon-owned definitions remain publisher-owned. NIL owns registration,
compatibility observations, enabled state, granted capabilities, migration
facts, and runtime audit. Removing an addon preserves user content and unknown
facet payloads.

The Hollis Labs plugin SDK should provide common manifest, lifecycle,
capability, configuration, trust, and diagnostics primitives where they fit.
NIL retains task/note-specific item, facet, view, and action contracts.

Trusted in-process Go or frontend extensions share desktop process authority.
Third-party or independently versioned code should run behind a process, WASM,
or declarative boundary with explicit filesystem, network, vault, UI, and
credential grants.

Potential addon boundaries:

- Journal is naturally an item kind/facet and view.
- Calendar is a NIL projection or adapter unless NIL becomes the external
  calendar authority by explicit product decision.
- Scheduler may own personal reminders but delegates general workflows.
- Broadcast uses a Tether or provider messaging adapter; NIL owns user intent,
  not general delivery infrastructure.
- Contacts can begin as typed items but should not smuggle a full CRM domain
  into the core model.

## Automations and flows

NIL may support bounded personal automations such as triage rules, recurrence,
saved transformations, reminders, and small content-to-item actions.

An automation definition is user/addon-owned, versioned input. NIL validates,
registers, and executes it under vault policy. Each run binds exact definition,
inputs, actor, grants, and outputs.

NIL does not need a second general workflow language. Multi-step durable flows,
external waits, generalized retries, complex branching, or portfolio-wide
orchestration can be delegated to Hadron. NIL remains independently useful
with a small deterministic local automation executor.

Hints and classifiers may suggest taxonomy or attention. Invariants such as
vault access, destructive confirmation, recurrence uniqueness, and external
delivery use deterministic enforcement.

## Persistence and recovery model

SQLite remains an appropriate local operational store. The conceptual durable
model includes:

- vault identity, policy, and storage binding;
- items, immutable revisions, and current heads;
- task-state and recurrence history;
- taxonomy and typed relations;
- inbox captures and cross-vault transfers;
- attachments and artifact references;
- saved views, vault-owned templates, and addon definitions/bindings;
- external bindings, sync journal, tombstones, and conflicts;
- mutation proposals, decisions, executions, and audit;
- addon registrations and migration state.

Persist facts that cannot be reconstructed. Rebuild FTS, HTML, plain text,
embeddings, counts, previews, and ordinary caches.

Deletion is a lifecycle:

```text
active -> trashed/tombstoned -> retention elapsed -> purged
```

The user may choose immediate purge, but external sync and audit must receive an
explicit tombstone rather than infer deletion forever from a missing integer
ID. Destructive purge explains which backups, replicas, chat records, and
attachments remain outside the operation.

Migrations are transactional where possible, idempotent, backed up, and
observable. Data conversion failures are preserved for recovery rather than
silently replaced with empty canonical content. A repair tool reports every
material change and never assumes an actively opened vault is safe to modify.

## Process and desktop/service model

The Wails desktop shell remains the primary deployment. One Go process owns the
active desktop application, service layer, vault connections, and embedded
frontend. The headless API is an optional deployment mode of the same product.

The target makes one process the writer authority for each open vault. GUI,
CLI, MCP, sync, and automations proxy that service where it exists. Offline
one-shot tools obtain exclusive ownership of a closed vault.

SQLite WAL and busy timeouts reduce contention; they do not establish process
ownership, coordinate migrations, or make cloud-synchronized concurrent files
safe.

Headless mode exposes liveness, readiness, locked-vault and degraded-capability
status, bounded graceful shutdown, and one-off administration. Optional model,
sync, or portfolio integrations may be degraded while local task/note access
remains ready.

NIL does not install or supervise a daemon. If a user chooses a persistent
headless service, the OS service manager or deployment runtime owns process
lifetime. Cerberus may validate/materialize the service registration and
observe its declared health without owning vault contents.

Desktop distribution remains an immutable release artifact. The application
does not compile itself, install arbitrary addon code, or mutate its release
bundle during normal execution.

## Configuration and settings ownership

Configuration separates:

- device-local application preferences;
- vault registry and storage bindings;
- vault-owned definitions and policies;
- addon registrations and grants;
- external resource and credential references;
- release/deployment settings;
- ephemeral invocation overrides.

User-authored saved views, templates, session profiles, automation definitions,
and vault policies should be versioned and exportable when users expect them to
travel. Window size, current tab, transient selection, and device-specific
shortcuts may remain device-local.

Config writes are atomic, permissions are restrictive, corruption is surfaced,
and the last known good version is recoverable. A load failure must not silently
replace a user's registry or security policy with permissive defaults.

Environment variables and flags are transports into typed runtime
configuration. They do not become the canonical model or contain raw secrets
when a credential reference can be resolved.

## Events, audit, and observability

Operational telemetry and durable user audit are separate.

Operational telemetry includes:

- structured logs to stderr/stdout or the desktop logging facility;
- traces for process, database, API, sync, and optional provider boundaries;
- metrics for latency, failure, queue depth, lock contention, and cache health;
- redacted support diagnostics.

Durable domain events include:

- item created, revised, merged, archived, trashed, restored, or purged;
- task completed, reopened, rescheduled, or recurred;
- capture triaged or transferred between vaults;
- relation or taxonomy changed;
- proposal created, approved, denied, expired, applied, or failed;
- reminder scheduled, delivered, acknowledged, or missed;
- sync discovered, exchanged, merged, conflicted, or completed;
- addon registered, granted, activated, disabled, or migrated;
- credential or vault grant issued, rotated, revoked, or denied.

Events carry stable identity, schema version, vault, subject and revision,
actor, causation, correlation, timestamp, and redacted payload or immutable
reference.

Audit answers:

- Who or what changed this item, from which revision, and why?
- Which external source or proposal produced the change?
- What did an AI model read, propose, and execute?
- Which capability and approval authorized the effect?
- When did a reminder, sync, export, or external delivery occur?
- Which replica or backup contains a revision or tombstone?
- Which addon and version interpreted this facet or automation?

Telemetry never includes content bodies by default. Domain audit stores the
minimum evidence required under vault retention policy rather than duplicating
entire private records into a global log.

## Portfolio composition

### Torque

NIL tasks are personal commitments, notes, and attention management. Torque
owns governed project work, assignment, dependencies, dispatch, attempts, and
acceptance.

A NIL item may reference or project a Torque task. The binding declares field
authority and sync direction. NIL does not mirror every Torque task into a
personal vault automatically, and Torque does not own the user's private note
body. Completing a personal reminder does not necessarily complete managed
work.

### Tesseract

NIL owns vault-local authored notes and retrieval. Tesseract owns authorized
portfolio memory and knowledge namespaces.

A user may explicitly promote a NIL note, excerpt, or pointer to Tesseract.
Tesseract records the adopted knowledge revision under namespace policy; NIL
retains its source item and delivery receipt. NIL does not mirror a private
vault wholesale, and a searchable note is not automatically canonical
knowledge.

### Fragments Engine

Fragments Engine owns broad content capture, normalization, provenance triage,
and routing. NIL owns intentional personal capture and task/note triage.

Fragments Engine can deliver a selected fragment into a NIL inbox through an
immutable source binding. NIL can send an explicitly selected note or capture
to Fragments Engine. Neither product scrapes or silently duplicates the other's
entire store.

### Loom

Loom owns content authoring, editorial operations, compilation, and publication
records. NIL may supply selected notes, ideas, sources, or personal reminders to
Loom and may receive an explicit content artifact or review reminder.

Loom does not own NIL's personal task lifecycle. NIL does not grow templates,
campaigns, review workflows, or publication delivery into its core simply
because a note can become published content.

### Hadron

Hadron owns reusable workflow definitions and durable general workflow
execution. NIL may trigger a pinned workflow or receive a result reference for
complex flows. NIL retains the user-facing item, proposal, and adoption record;
Hadron retains the run.

Simple local recurrence, reminders, and deterministic triage remain NIL-native
and do not require Hadron.

### Nanite

Nanite owns agent sessions and runtime. NIL may request a bounded vault helper,
researcher, organizer, or review session through exact context references and
scoped capabilities. NIL applies returned proposals through its own policy.

Native bounded chat can remain independently useful through shared provider and
tool-loop libraries. It must not evolve into a second general agent platform.

### Tether

Tether is optional for model/MCP gateway access, messaging, and federation.
NIL owns vault policy, user intent, and content-egress decisions. Tether owns
gateway and transport mechanics. A Broadcast addon uses a Tether or provider
adapter rather than making NIL a message broker.

### Tangent

NIL already owns its desktop interaction surface. Tangent may host a rich
external review or input interaction requested by an automation, but it is not
required for core editing or approval. Typed resolutions return through the
same NIL proposal boundary.

### Sigil

Sigil may compile publisher-owned UI definitions into artifacts, but NIL's
domain and desktop runtime remain NIL-owned. Shared generator or design-system
patterns do not transfer runtime authorization or user-data authority.

### Cerberus and Coder

Cerberus may materialize optional headless-service configuration and observe
health. NIL remains primarily a desktop application and does not require
Cerberus. Coder workspaces may host an agent that calls NIL through a scoped
API capability; the workspace does not own the vault or item.

## Twelve-factor and Go operating model

| Factor | NIL direction |
|---|---|
| One codebase | One revision-controlled NIL codebase produces desktop and optional headless deployments |
| Dependencies | Declare Go modules, frontend packages, Wails, toolchains, addons, and external CLIs explicitly |
| Config | Parse device, vault, release, and invocation configuration into typed models; secrets remain references |
| Backing services | Treat vault stores, artifact directories, providers, sync targets, and peer apps as attached resources |
| Build/release/run | Build immutable signed desktop/headless artifacts; release binds platform configuration; run does not compile source |
| Processes | Keep process memory disposable; restore from explicit vault and application stores |
| Port binding | Optional headless/HTTP mode binds an explicit loopback address; desktop IPC needs no port |
| Concurrency | Use goroutines for bounded work and one declared writer authority per vault; scale is product-driven, not assumed |
| Disposability | Start promptly, recover vault state, checkpoint safely, honor cancellation, and shut down within a bound |
| Dev/prod parity | Use the same schemas, service contracts, conversion logic, and generated-binding checks across builds |
| Logs | Emit structured redacted operational logs; keep user audit in vault-scoped durable storage |
| Admin processes | Run migrate, verify, backup, restore, import, export, and repair from the same release |

## Current strengths to preserve

- The user-facing thesis—tasks and notes are one object—is coherent and
  differentiating.
- Per-vault SQLite provides local-first operation and meaningful separation.
- Vault deletion unregisters rather than deleting user files.
- Wails produces a compact desktop app with same-process GUI access.
- The HTTP API is separated from Wails and can run headlessly.
- `nil-mcp` is a thin HTTP adapter rather than a competing database owner.
- GUI, HTTP, CLI, and most chat behavior converge on a shared item service.
- Quick Add offers low-friction capture with useful deterministic syntax.
- Inbox preserves capture without forcing immediate taxonomy.
- Projects, contexts, tags, scopes, saved views, and sessions offer orthogonal
  organization.
- Canonical rich text is structured ProseMirror JSON rather than opaque HTML.
- HTML and plain text are explicitly derived representations.
- FTS5 provides fast offline retrieval.
- Wikilinks and backlinks create a useful personal graph.
- The kinds registry anticipates extensibility without splitting core CRUD.
- External references and batch create show real integration pressure and
  idempotency awareness.
- The chat Propose → Approve/Deny → Execute flow is a strong safety primitive.
- Vault capabilities default chat to read-only.
- Chat audit records proposals, denials, failures, dry runs, and executions.
- The API binds to loopback rather than exposing vault data to the LAN by
  default.
- The project has current architecture documentation, accepted ADRs, recovery
  tooling, migration tests, binding-drift checks, and a broad verification
  command.

## Architectural tensions to resolve

### Mutable item rows lack durable revision history

The `todos` row currently combines identity, canonical body, task state,
current metadata, and deletion behavior. External sync, AI proposals,
conflicts, audit, rollback, and trustworthy references need immutable item
revisions plus a current-head projection.

### Integer IDs do not survive vault boundaries

Item IDs and wikilink targets are vault-local row numbers. Copying an item to
another vault resets its ID, risks broken references, and makes durable
external correlation harder. The target needs stable portable identities.

### Cross-vault move is copy then delete

The current move is non-atomic across SQLite databases. A crash can leave a
duplicate or missing source disposition, and source references may change.
The target needs a recoverable transfer record and idempotent completion.

### Invalid vault selection falls back to active vault

The HTTP store resolver currently uses the active store when a supplied vault
ID cannot be resolved. This is a fail-open authority error. Named unknown or
unauthorized vaults must fail closed.

### External reference is unqualified mutable upsert

`external_ref` is a vault-scoped string whose create path overwrites an
existing record's create fields. It does not identify source authority,
connector, external revision, field ownership, or conflict policy. The target
uses qualified bindings and explicit proposed revisions.

### Deletion is inferred from missing IDs

External clients compare ID sets to infer deletion. That cannot explain purge,
move, permission loss, replica lag, or temporary unavailability. Tombstones and
a revision/change feed are needed in the target semantics.

### Live SQLite files are presented as cloud-sync units

Per-vault portability is valuable, but raw cloud-folder synchronization of an
open SQLite/WAL store has concurrency and conflict hazards. The target needs an
explicit checkpoint or replication contract and honest operating modes.

### Multiple processes can believe they own one vault

GUI, headless API, direct CLI, and other processes can open the same SQLite
file. WAL helps SQL concurrency but does not establish migration ownership,
cross-process lifecycle, backup coordination, or addon/run authority.

### Shared inbox is a parallel store

The inbox has its own SQLite store and different move behavior depending on
surface. Its custody, backup, identity, and transfer semantics need to align
with ordinary vaults.

### Chat state is outside vault custody

Chat sessions, prompts, model results, proposals, tool calls, templates, and
audit live in a separate `chat.db`. They can contain full vault content but do
not naturally travel, lock, back up, or expire with the corresponding vault.

### Chat mutation bypasses shared service semantics

The proposal runner and direct-create tool do not uniformly use the shared
item application service. Safety policy should wrap one canonical mutation
path rather than preserve semantic differences per caller.

### Cleartext secrets live in ordinary configuration

The model provider key and NIL API key are stored in `config.json`, and the
settings API returns the API key for display. The target uses external provider
credential references and protected/scoped NIL client credentials.

### One shared API key grants every operation

Loopback API authentication has one cleartext bearer key, no client identity,
no per-vault or effect scope, permissive CORS, and non-constant-time comparison.
MCP inherits the same full authority by reading the config file.

### Provider-specific chat mechanics are embedded in the domain app

The chat bridge hand-maintains Anthropic's wire protocol and tool loop. The
target keeps NIL-specific chat and proposal semantics while putting provider
mechanics behind official or shared provider-neutral adapters.

### Capability and approval are partially conflated

Read, Write, Delete, and DirectCreate are useful beginnings, but approval,
effect authorization, vault scope, data egress, direct execution, and retention
need separate policy concepts. Auto-approved operations still require audit.

### One physical table carries all kind concerns

One item identity is the correct logical model, but a growing set of addons
will make one wide table and application-only kind validation increasingly
fragile. Typed facet storage can preserve unity without forcing irrelevant
columns on every kind.

### Meaningful user definitions are split across localStorage and databases

Saved tabs, session profiles, task templates, themes, chat templates, and vault
policy live in different stores with different backup and portability
semantics. The target distinguishes device preferences from vault-owned user
definitions.

### Telemetry privacy is not a product-level contract

OTel initialization exists, but vault content, paths, queries, chat payloads,
and tool results need an explicit redaction and opt-in policy across every
instrumented boundary.

## Boundary guidance

A capability belongs in NIL core when it protects or expresses personal
task/note semantics:

- vault custody and authorization;
- item identity, revision, body, facets, and task state;
- inbox and triage;
- taxonomy, references, search, and saved views;
- personal recurrence, reminders, and attention;
- mutation proposals, user decisions, and audit;
- sync/export bindings and conflict presentation;
- domain-level API and addon contracts.

A capability belongs in a shared library when it is reusable mechanism without
NIL domain semantics:

- provider-neutral model contracts and bounded tool loops;
- MCP/HTTP plumbing;
- SQLite, migration, backup, and safe materialization helpers;
- plugin manifests and lifecycle;
- structured document codecs;
- telemetry and redaction helpers;
- credential-reference and capability primitives.

A capability belongs behind an adapter when it varies by provider:

- model calls and external agent sessions;
- sync transports and cloud storage;
- import/export formats;
- calendar, notification, and messaging providers;
- knowledge, fragment, task, and publishing integrations;
- OS credentials and encryption mechanisms.

A capability belongs in another application when it has its own durable domain:

- governed project work in Torque;
- knowledge and memory in Tesseract;
- fragment intake and routing in Fragments Engine;
- editorial production and publication in Loom;
- workflow execution in Hadron;
- agent sessions in Nanite;
- gateways and messaging in Tether;
- infrastructure reconciliation in Cerberus.

Warning signs that NIL is crossing its boundary:

- every project task is mirrored into a personal vault without field authority;
- a private note is promoted to portfolio knowledge automatically;
- broad source capture and routing duplicate Fragments Engine;
- a note editor grows into Loom's editorial and publication system;
- local recurrence grows into a general workflow engine;
- chat grows into a second general agent runtime;
- a model result mutates data without proposal/policy/audit;
- an addon receives ambient access to every vault, file, network endpoint, or
  credential;
- a valid API token implies access to every vault and destructive operation;
- unknown vault identity falls back to whichever vault is active;
- raw SQLite file copying is marketed as conflict-safe multi-device sync;
- provider credentials appear in config, vaults, logs, traces, or chat history;
- NIL requires another portfolio app for ordinary capture, editing, or search.

## Questions for the next architecture session

1. What exact stable identity scheme should vaults, items, revisions, relations,
   taxonomy values, and external bindings use?
2. Which item fields belong to immutable content revisions versus operational
   task state or derived projections?
3. How should task/note unity map to typed facets without losing the simplicity
   of one item model?
4. What revision, conflict, and merge semantics support simultaneous GUI,
   external, AI, and sync edits?
5. Is the shared inbox a special vault, a staging store, or a placement inside
   each destination vault?
6. What authority and identity rules apply when moving or copying items between
   vaults?
7. What is NIL's supported simple cloud-folder contract, and what true
   replication model handles concurrent devices?
8. Which durable data belongs inside each vault versus an application-level
   store, especially chat, audit, addons, views, and sync state?
9. What deletion, retention, tombstone, and purge semantics apply to local
   content, replicas, backups, chat, embeddings, and external bindings?
10. Which structured-document schema and extension policy preserves TipTap
    content across editor and addon upgrades?
11. Which kind/facet definitions are portable data and which may execute code?
12. What minimal per-client principal and capability model replaces the shared
    loopback API key while preserving easy local use?
13. Which direct actions, if any, may be auto-approved, and what deterministic
    policy bounds them?
14. What egress and retention controls apply to provider-backed chat, semantic
    search, and delegated agents?
15. Which native chat behavior remains embedded versus delegated to Nanite or
    Tether, and what proposal contract keeps those choices interchangeable?
16. Where is the boundary between NIL personal automations and Hadron workflows?
17. Which first-party addons are item facets/views versus integrations with an
    external authority?
18. What process owns an open vault when the desktop, headless API, CLI, MCP,
    sync, and scheduled reminders are all available?
19. What constitutes a complete, restorable, and verifiable vault backup?
20. Which operational telemetry is useful enough to justify collection without
    exposing private content or behavior?
