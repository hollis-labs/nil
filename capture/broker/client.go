package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hollis-labs/libs/security/pairing"
	"github.com/hollis-labs/libs/security/secrets"
)

// HTTPError exposes a bounded status, never a remote error body, token, URL,
// echoed request, or transport diagnostic. All paths are fixed protocol routes.
type HTTPError struct{ Status int }

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Basalisk request refused (HTTP %d); retain the capture and repair pairing/connection", e.Status)
}
func IsStatus(err error, status int) bool {
	var e *HTTPError
	return errors.As(err, &e) && e.Status == status
}

type Client struct {
	Origin     string
	DeviceID   string
	credential func(context.Context) (secrets.Secret, error)
	http       *http.Client
}

func New(origin, deviceID string, credential func(context.Context) (secrets.Secret, error), transport http.RoundTripper) (*Client, error) {
	normalized, err := pairing.Origin(origin)
	if err != nil {
		return nil, errors.New("use an HTTPS origin or authenticated loopback tunnel origin, without path or credentials")
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{Origin: normalized, DeviceID: deviceID, credential: credential, http: &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("basalisk redirects are refused") }}}, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any, authenticated bool) (int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Origin+path, reader)
	if err != nil {
		return 0, errors.New("cannot construct Basalisk request")
	}
	req.Header.Set(pairing.ProtocolHeader, pairing.Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		if c.credential == nil {
			return 0, errors.New("pair Nil with Basalisk before delivery")
		}
		token, resolveErr := c.credential(ctx)
		if resolveErr != nil {
			return 0, errors.New("native Basalisk credential unavailable; repair pairing")
		}
		if token.Reveal() == "" {
			return 0, errors.New("native Basalisk credential is empty")
		}
		req.Header.Set("Authorization", "Bearer "+token.Reveal())
	}
	response, err := c.http.Do(req)
	if err != nil {
		return 0, errors.New("basalisk transport unavailable; the outcome may be ambiguous")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, &HTTPError{response.StatusCode}
	}
	if out != nil {
		data, readErr := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
		if readErr != nil || len(data) > 256<<10 {
			return response.StatusCode, errors.New("basalisk response unavailable or too large")
		}
		if json.Unmarshal(data, out) != nil {
			return response.StatusCode, errors.New("basalisk response does not match capture protocol")
		}
	}
	return response.StatusCode, nil
}

// Exchange uses the exact shared protocol DTO. Only native callers may receive
// this result; never return pairing.Exchange from a renderer-bound signature.
func (c *Client) Exchange(ctx context.Context, code string) (pairing.Exchange, error) {
	var result pairing.Exchange
	if code == "" || len(code) > 256 {
		return result, errors.New("enter a one-use Capture app code from Basalisk")
	}
	data, err := json.Marshal(pairing.ExchangeRequest{Code: code, Audience: "basalisk", Scopes: []string{ContextScope, CaptureScope}})
	if err != nil {
		return result, err
	}
	_, err = c.do(ctx, http.MethodPost, pairing.ExchangePath, data, &result, false)
	if err != nil {
		return pairing.Exchange{}, err
	}
	if !validDevice(result.Device) || result.Token == "" || !strings.EqualFold(result.TokenType, "Bearer") {
		return pairing.Exchange{}, errors.New("basalisk pairing result has unexpected identity or permissions")
	}
	return result, nil
}

func validDevice(d pairing.Device) bool {
	if d.ID == "" || d.Audience != "basalisk" || d.RevokedAt != nil || !d.ExpiresAt.After(time.Now()) || !slices.Contains(d.Scopes, CaptureScope) {
		return false
	}
	for _, scope := range d.Scopes {
		if scope != CaptureScope && scope != ContextScope {
			return false
		}
	}
	return true
}
func (c *Client) Renew(ctx context.Context) (pairing.Device, error) {
	var d pairing.Device
	_, err := c.do(ctx, http.MethodPost, pairing.RenewPath, []byte("{}"), &d, true)
	if err == nil && (!validDevice(d) || d.ID != c.DeviceID) {
		err = errors.New("renewal returned a different identity or permissions")
	}
	return d, err
}
func (c *Client) Discover(ctx context.Context) (Capabilities, error) {
	var result Capabilities
	_, err := c.do(ctx, http.MethodGet, "/companion/capture-capabilities", nil, &result, true)
	if err == nil && (result.SchemaVersion != 1 || result.DeviceID != c.DeviceID || result.PairingProtocol != pairing.Version || !slices.Contains(result.Scopes, CaptureScope)) {
		err = errors.New("basalisk capture capabilities do not match this paired identity")
	}
	return result, err
}
func (c *Client) Connection(ctx context.Context) (Connection, error) {
	var result Connection
	_, err := c.do(ctx, http.MethodGet, "/companion/fragment-connection", nil, &result, true)
	if err == nil && (result.DeviceID != c.DeviceID || result.IntegrationID == "") {
		err = errors.New("basalisk connection does not match this paired identity")
	}
	return result, err
}
func (c *Client) Bind(ctx context.Context, id string) (Binding, int, error) {
	var result Binding
	if !ValidID(id) {
		return result, 0, errors.New("invalid capture ID")
	}
	data, _ := json.Marshal(struct {
		CaptureID string `json:"captureID"`
	}{id})
	status, err := c.do(ctx, http.MethodPost, "/companion/capture-context", data, &result, true)
	if err == nil {
		origin, originErr := pairing.Origin(result.Origin)
		_, timeErr := time.Parse(time.RFC3339Nano, result.DraftAt)
		if result.SchemaVersion != 1 || result.CaptureID != id || result.DeviceID != c.DeviceID || originErr != nil || origin != c.Origin || timeErr != nil || !ValidID(result.BindingID) || !slices.Contains([]string{"active", "paused", "none"}, result.State) || (status != 200 && status != 201) {
			err = errors.New("basalisk draft binding does not match this capture, origin or identity")
		}
	}
	return result, status, err
}
func (c *Client) Submit(ctx context.Context, raw []byte) (Status, error) {
	var result Status
	_, err := c.do(ctx, http.MethodPost, "/companion/fragments", raw, &result, true)
	return result, err
}
func (c *Client) Status(ctx context.Context, id string) (Status, error) {
	var result Status
	if !ValidID(id) {
		return result, errors.New("invalid capture ID")
	}
	_, err := c.do(ctx, http.MethodGet, "/companion/fragments/"+id, nil, &result, true)
	return result, err
}
func (c *Client) List(ctx context.Context, before int64) (StatusPage, error) {
	var result StatusPage
	path := "/companion/fragments"
	if before > 0 {
		path += "?before=" + strconv.FormatInt(before, 10)
	}
	_, err := c.do(ctx, http.MethodGet, path, nil, &result, true)
	return result, err
}
func (c *Client) Action(ctx context.Context, id, action string) (Status, error) {
	var result Status
	if !ValidID(id) || (action != "retry" && action != "stop") {
		return result, errors.New("invalid capture recovery action")
	}
	// Recovery requires an EMPTY body; neither {} nor a new payload is sent.
	_, err := c.do(ctx, http.MethodPost, "/companion/fragments/"+id+"/"+action, nil, &result, true)
	return result, err
}
func (c *Client) Evidence(ctx context.Context) (Readiness, error) {
	var result Readiness
	_, err := c.do(ctx, http.MethodGet, "/companion/capture-readiness", nil, &result, true)
	if err == nil && (result.SchemaVersion != 1 || len(result.Apps) != 1 || result.Apps[0].DeviceID != c.DeviceID) {
		err = errors.New("basalisk readiness does not match this paired identity")
	}
	return result, err
}
func ValidID(id string) bool {
	if len(id) < 16 || len(id) > 100 {
		return false
	}
	for _, c := range id {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}
