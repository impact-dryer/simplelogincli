package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestUserInfo_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user_info" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authentication"); got != "k" {
			t.Fatalf("Authentication header = %q, want k", got)
		}
		_ = json.NewEncoder(w).Encode(UserInfo{Name: "John", Email: "john@example.com", IsPremium: true})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "k")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ui, err := c.UserInfo(ctx)
	if err != nil {
		t.Fatalf("UserInfo() error = %v", err)
	}
	if ui.Email != "john@example.com" || ui.Name != "John" || !ui.IsPremium {
		t.Fatalf("UserInfo = %#v", ui)
	}
}

func TestAliasOptions_WithHostname(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/alias/options" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("hostname") != "example.com" {
			t.Fatalf("want hostname query")
		}
		_ = json.NewEncoder(w).Encode(AliasOptionsResponse{CanCreate: true, PrefixSuggestion: "ex", Suffixes: []SuffixOption{{Suffix: ".a@b", SignedSuffix: ".a@b.sig", IsCustom: false, IsPremium: false}}})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "k")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.AliasOptions(ctx, "example.com")
	if err != nil {
		t.Fatalf("AliasOptions() error = %v", err)
	}
	if !res.CanCreate || res.PrefixSuggestion != "ex" || len(res.Suffixes) != 1 {
		t.Fatalf("res = %#v", res)
	}
}

func TestCreateRandomAlias_QueryAndBody(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatal("method")
		}
		if r.URL.Path != "/api/alias/random/new" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("mode") != "word" {
			t.Fatalf("mode query")
		}
		if r.URL.Query().Get("hostname") != "ex.com" {
			t.Fatalf("hostname query")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["note"].(string) != "n" {
			t.Fatalf("note body = %#v", body)
		}
		called = true
		_ = json.NewEncoder(w).Encode(Alias{ID: 1, Email: "rand@sl"})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "k")
	n := "n"
	ctx := context.Background()
	a, err := c.CreateRandomAlias(ctx, "ex.com", "word", &n)
	if err != nil {
		t.Fatalf("CreateRandomAlias() error = %v", err)
	}
	if a.Email != "rand@sl" || !called {
		t.Fatalf("alias = %#v called=%v", a, called)
	}
}

func TestCreateCustomAlias_BodyFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatal("method")
		}
		if r.URL.Path != "/api/v3/alias/custom/new" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body createCustomAliasRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.AliasPrefix != "p" || body.SignedSuffix != ".x@y.sig" || len(body.MailboxIDs) != 2 || body.MailboxIDs[0] != 1 || body.MailboxIDs[1] != 2 {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(Alias{ID: 2, Email: "p.x@y"})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "k")
	ctx := context.Background()
	name := "Name"
	note := "Note"
	a, err := c.CreateCustomAlias(ctx, "", "p", ".x@y.sig", []int{1, 2}, &note, &name)
	if err != nil {
		t.Fatalf("CreateCustomAlias() error = %v", err)
	}
	if a.Email != "p.x@y" {
		t.Fatalf("alias = %#v", a)
	}
}

func TestDefaultMailboxID_PrefersDefaultThenVerified(t *testing.T) {
	cases := []struct {
		mailboxes []Mailbox
		want      int
	}{{
		mailboxes: []Mailbox{{ID: 3, Verified: true}, {ID: 1, Default: true}},
		want:      1,
	}, {
		mailboxes: []Mailbox{{ID: 9, Verified: true}, {ID: 5}},
		want:      9,
	}, {
		mailboxes: []Mailbox{{ID: 7}},
		want:      7,
	}}
	for i, tc := range cases {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v2/mailboxes" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(MailboxesResponse{Mailboxes: tc.mailboxes})
		}))
		c := NewClient(ts.URL, "k")
		got, err := c.DefaultMailboxID(context.Background())
		ts.Close()
		if err != nil {
			t.Fatalf("case %d err = %v", i, err)
		}
		if got != tc.want {
			t.Fatalf("case %d got = %d want = %d", i, got, tc.want)
		}
	}
}

func TestErrorHandling_PropagatesAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad input"})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "k")
	_, err := c.UserInfo(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "bad input") {
		t.Fatalf("err = %v", err)
	}
}

func TestNoAPIKeyHeaderWhenEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("Authentication"); v != "" {
			t.Fatalf("unexpected Authentication header: %q", v)
		}
		_ = json.NewEncoder(w).Encode(UserInfo{})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "")
	_, _ = c.UserInfo(context.Background())
}

// Ensure request contains JSON content-type when body is present
func TestContentTypeWhenBody(t *testing.T) {
	var got string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Content-Type")
		_ = json.NewEncoder(w).Encode(Alias{Email: "ok"})
	}))
	defer ts.Close()
	c := NewClient(ts.URL, "k")
	note := "n"
	_, _ = c.CreateRandomAlias(context.Background(), "", "", &note)
	if !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content-type = %q", got)
	}
}

// Sanity check that JSON tags match request struct fields
func TestRequestStructTags(t *testing.T) {
	v := createCustomAliasRequest{AliasPrefix: "p", SignedSuffix: "s", MailboxIDs: []int{1}, Note: nil, Name: nil}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	// Marshal then unmarshal to map and verify keys exist
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"alias_prefix", "signed_suffix", "mailbox_ids"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing key %s in %s", k, string(b))
		}
	}
	// MailboxIDs should be []any (slice) after roundtrip
	if _, ok := m["mailbox_ids"].([]any); !ok {
		t.Fatalf("mailbox_ids type = %T", m["mailbox_ids"])
	}
	if !reflect.DeepEqual(v.MailboxIDs, []int{1}) {
		t.Fatalf("ids mutated")
	}
}
