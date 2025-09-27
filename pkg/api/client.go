package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://app.simplelogin.io"

type Client struct {
	baseURL string
	hc      *http.Client
	apiKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc:      &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
	}
}

func (c *Client) newReq(ctx context.Context, method, path string, body any, query url.Values) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	full := c.baseURL + path
	if len(query) > 0 {
		if strings.Contains(full, "?") {
			full += "&" + query.Encode()
		} else {
			full += "?" + query.Encode()
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, full, r)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authentication", c.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(b, &e) == nil && e.Error != "" {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, e.Error)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			return err
		}
	}
	return nil
}

// Models

type UserInfo struct {
	Name              string `json:"name"`
	IsPremium         bool   `json:"is_premium"`
	Email             string `json:"email"`
	InTrial           bool   `json:"in_trial"`
	ProfilePictureURL string `json:"profile_picture_url"`
	MaxAliasFreePlan  int    `json:"max_alias_free_plan"`
}

type Alias struct {
	ID                int     `json:"id"`
	Email             string  `json:"email"`
	Name              *string `json:"name"`
	Enabled           bool    `json:"enabled"`
	CreationTimestamp int64   `json:"creation_timestamp"`
	Note              *string `json:"note"`
	NbBlock           int     `json:"nb_block"`
	NbForward         int     `json:"nb_forward"`
	NbReply           int     `json:"nb_reply"`
	Pinned            bool    `json:"pinned"`
}

type SuffixOption struct {
	SignedSuffix string `json:"signed_suffix"`
	Suffix       string `json:"suffix"`
	IsCustom     bool   `json:"is_custom"`
	IsPremium    bool   `json:"is_premium"`
}

type AliasOptionsResponse struct {
	CanCreate        bool           `json:"can_create"`
	PrefixSuggestion string         `json:"prefix_suggestion"`
	Suffixes         []SuffixOption `json:"suffixes"`
}

type Mailbox struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Default  bool   `json:"default"`
	Verified bool   `json:"verified"`
}

type MailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
}

// Requests

type createRandomAliasRequest struct {
	Note *string `json:"note,omitempty"`
}

type createCustomAliasRequest struct {
	AliasPrefix  string  `json:"alias_prefix"`
	SignedSuffix string  `json:"signed_suffix"`
	MailboxIDs   []int   `json:"mailbox_ids"`
	Note         *string `json:"note,omitempty"`
	Name         *string `json:"name,omitempty"`
}

// API methods

func (c *Client) UserInfo(ctx context.Context) (UserInfo, error) {
	req, err := c.newReq(ctx, http.MethodGet, "/api/user_info", nil, nil)
	if err != nil {
		return UserInfo{}, err
	}
	var out UserInfo
	return out, c.doJSON(req, &out)
}

func (c *Client) AliasOptions(ctx context.Context, hostname string) (AliasOptionsResponse, error) {
	q := url.Values{}
	if strings.TrimSpace(hostname) != "" {
		q.Set("hostname", hostname)
	}
	req, err := c.newReq(ctx, http.MethodGet, "/api/v5/alias/options", nil, q)
	if err != nil {
		return AliasOptionsResponse{}, err
	}
	var out AliasOptionsResponse
	return out, c.doJSON(req, &out)
}

func (c *Client) CreateRandomAlias(ctx context.Context, hostname, mode string, note *string) (Alias, error) {
	q := url.Values{}
	if strings.TrimSpace(hostname) != "" {
		q.Set("hostname", hostname)
	}
	if m := strings.ToLower(strings.TrimSpace(mode)); m != "" {
		q.Set("mode", m)
	}
	var body *createRandomAliasRequest
	if note != nil && *note != "" {
		body = &createRandomAliasRequest{Note: note}
	}
	req, err := c.newReq(ctx, http.MethodPost, "/api/alias/random/new", body, q)
	if err != nil {
		return Alias{}, err
	}
	var out Alias
	return out, c.doJSON(req, &out)
}

func (c *Client) CreateCustomAlias(ctx context.Context, hostname, aliasPrefix, signedSuffix string, mailboxIDs []int, note, name *string) (Alias, error) {
	q := url.Values{}
	if strings.TrimSpace(hostname) != "" {
		q.Set("hostname", hostname)
	}
	body := createCustomAliasRequest{
		AliasPrefix:  aliasPrefix,
		SignedSuffix: signedSuffix,
		MailboxIDs:   mailboxIDs,
		Note:         note,
		Name:         name,
	}
	req, err := c.newReq(ctx, http.MethodPost, "/api/v3/alias/custom/new", body, q)
	if err != nil {
		return Alias{}, err
	}
	var out Alias
	return out, c.doJSON(req, &out)
}

func (c *Client) Mailboxes(ctx context.Context) (MailboxesResponse, error) {
	req, err := c.newReq(ctx, http.MethodGet, "/api/v2/mailboxes", nil, nil)
	if err != nil {
		return MailboxesResponse{}, err
	}
	var out MailboxesResponse
	return out, c.doJSON(req, &out)
}

func (c *Client) DefaultMailboxID(ctx context.Context) (int, error) {
	m, err := c.Mailboxes(ctx)
	if err != nil {
		return 0, err
	}
	if len(m.Mailboxes) == 0 {
		return 0, errors.New("no mailboxes found in account")
	}
	for _, mb := range m.Mailboxes {
		if mb.Default {
			return mb.ID, nil
		}
	}
	for _, mb := range m.Mailboxes {
		if mb.Verified {
			return mb.ID, nil
		}
	}
	return m.Mailboxes[0].ID, nil
}

// DeleteAlias removes an alias by id (DELETE /api/aliases/:alias_id)
func (c *Client) DeleteAlias(ctx context.Context, aliasID int) error {
	path := "/api/aliases/" + strconv.Itoa(aliasID)
	req, err := c.newReq(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, nil)
}
