// Package broker consumes the published Basalisk capture protocol. It contains
// no owner API, issuer enrollment, FE key, or arbitrary URL-fetching facility.
package broker

const CaptureScope = "basalisk.fragments.create"
const ContextScope = "basalisk.activity.read"

type Version struct {
	Version int `json:"version"`
}
type Capabilities struct {
	SchemaVersion    int               `json:"schemaVersion"`
	DeviceID         string            `json:"deviceID"`
	Scopes           []string          `json:"scopes"`
	PairingProtocol  string            `json:"pairingProtocol"`
	MetadataVersions []int             `json:"captureMetadataVersions"`
	Context          ContextCapability `json:"captureContext"`
	Recovery         Version           `json:"captureRecovery"`
	Readiness        Version           `json:"captureReadiness"`
	Limits           CaptureLimits     `json:"limits"`
}

type Connection struct {
	DeviceID      string `json:"deviceId"`
	IntegrationID string `json:"integrationId"`
	Name          string `json:"name"`
}

type Binding struct {
	SchemaVersion int    `json:"schemaVersion"`
	BindingID     string `json:"bindingID"`
	CaptureID     string `json:"captureID"`
	DeviceID      string `json:"deviceID"`
	Origin        string `json:"origin"`
	DraftAt       string `json:"draftAt"`
	State         string `json:"state"`
}

type ClientMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type Reference struct {
	Relation string `json:"relation"`
	Provider string `json:"provider"`
	Instance string `json:"instance"`
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Revision string `json:"revision,omitempty"`
	Label    string `json:"label,omitempty"`
}
type Preset struct {
	ID        string `json:"id"`
	Revision  string `json:"revision"`
	AppliedBy string `json:"appliedBy"`
}
type Metadata struct {
	SchemaVersion int            `json:"schemaVersion"`
	Client        ClientMetadata `json:"client"`
	Title         string         `json:"title,omitempty"`
	Kind          string         `json:"kind,omitempty"`
	Projects      []string       `json:"projects,omitempty"`
	Contexts      []string       `json:"contexts,omitempty"`
	Priority      string         `json:"priority,omitempty"`
	DueDate       string         `json:"dueDate,omitempty"`
	References    []Reference    `json:"references,omitempty"`
	Preset        *Preset        `json:"preset,omitempty"`
}
type Request struct {
	CaptureID        string   `json:"captureID"`
	IntegrationID    string   `json:"integrationID"`
	Content          string   `json:"content"`
	Tags             []string `json:"tags"`
	DraftAt          string   `json:"draftAt"`
	SubmittedAt      string   `json:"submittedAt"`
	Metadata         Metadata `json:"metadata"`
	ContextBindingID string   `json:"contextBindingID,omitempty"`
}
type Status struct {
	Sequence           int64  `json:"sequence"`
	CaptureID          string `json:"captureID"`
	DeviceID           string `json:"deviceID"`
	IntegrationID      string `json:"integrationID"`
	State              string `json:"state"`
	Code               string `json:"code"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	FragmentID         string `json:"fragmentID,omitempty"`
	FragmentRevisionID string `json:"fragmentRevisionID,omitempty"`
	MayHaveSent        bool   `json:"mayHaveSent"`
	Delivering         bool   `json:"delivering"`
	NextAttempt        int64  `json:"nextAttempt,omitempty"`
}

func (s Status) Confirmed() bool {
	return s.State == "saved" && s.FragmentID != "" && s.FragmentRevisionID != ""
}

type StatusPage struct {
	Items      []Status `json:"items"`
	NextBefore int64    `json:"nextBefore"`
}

type Readiness struct {
	SchemaVersion   int           `json:"schemaVersion"`
	GeneratedAt     string        `json:"generatedAt"`
	ValidForSeconds int           `json:"validForSeconds"`
	Source          string        `json:"source"`
	UpstreamHealth  string        `json:"upstreamHealth"`
	PortfolioHealth string        `json:"portfolioHealth"`
	Apps            []AppEvidence `json:"apps"`
}
type AppEvidence struct {
	DeviceID       string             `json:"deviceID"`
	Authorization  string             `json:"authorization"`
	ContextAllowed bool               `json:"contextAllowed"`
	Connection     EvidenceConnection `json:"connection"`
	Worker         EvidenceWorker     `json:"worker"`
	Queue          EvidenceQueue      `json:"queue"`
	LastConfirmed  *Status            `json:"lastConfirmed,omitempty"`
}

type ContextCapability struct {
	Version int  `json:"version"`
	Allowed bool `json:"allowed"`
}
type CaptureLimits struct {
	RequestBytes  int `json:"requestBytes"`
	ContentBytes  int `json:"contentBytes"`
	MetadataBytes int `json:"metadataBytes"`
	Tags          int `json:"tags"`
	References    int `json:"references"`
}

type EvidenceConnection struct {
	State         string `json:"state"`
	IntegrationID string `json:"integrationID"`
	Name          string `json:"name"`
}

type EvidenceWorker struct {
	State           string `json:"state"`
	ObservedAt      string `json:"observedAt"`
	ValidForSeconds int    `json:"validForSeconds"`
}

type EvidenceQueue struct {
	Queued         int `json:"queued"`
	NeedsAttention int `json:"needsAttention"`
	Saved          int `json:"saved"`
	Stopped        int `json:"stopped"`
	Delivering     int `json:"delivering"`
}
