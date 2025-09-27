//go:build integration

package api

import (
	"context"
	"os"
	"testing"
	"time"
)

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return v
}

func clientFromEnv(t *testing.T) *Client {
	apiKey := requireEnv(t, "SIMPLELOGIN_API_KEY")
	baseURL := os.Getenv("SIMPLELOGIN_BASE_URL")
	return NewClient(baseURL, apiKey)
}

func TestIntegration_RandomAliasCreateAndDelete(t *testing.T) {
	c := clientFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Validate key works
	if _, err := c.UserInfo(ctx); err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	// Create random alias
	note := "cli-itest"
	a, err := c.CreateRandomAlias(ctx, "example.com", "word", &note)
	if err != nil {
		t.Fatalf("CreateRandomAlias: %v", err)
	}
	if a.Email == "" || a.ID == 0 {
		t.Fatalf("unexpected alias: %#v", a)
	}
	// Cleanup
	if err := c.DeleteAlias(ctx, a.ID); err != nil {
		t.Fatalf("DeleteAlias: %v", err)
	}
}

func TestIntegration_CustomAliasCreateAndDelete(t *testing.T) {
	c := clientFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	opts, err := c.AliasOptions(ctx, "example.com")
	if err != nil {
		t.Fatalf("AliasOptions: %v", err)
	}
	if !opts.CanCreate || len(opts.Suffixes) == 0 {
		t.Skip("cannot create alias on this account or no suffixes available")
	}
	// Choose the first suffix
	ss := opts.Suffixes[0].SignedSuffix
	if ss == "" {
		t.Skip("no signed suffix available")
	}
	mid, err := c.DefaultMailboxID(ctx)
	if err != nil {
		t.Fatalf("DefaultMailboxID: %v", err)
	}
	prefix := "cli-itest-" + time.Now().Format("150405")
	a, err := c.CreateCustomAlias(ctx, "example.com", prefix, ss, []int{mid}, nil, nil)
	if err != nil {
		t.Fatalf("CreateCustomAlias: %v", err)
	}
	if a.Email == "" || a.ID == 0 {
		t.Fatalf("unexpected alias: %#v", a)
	}
	// Cleanup
	if err := c.DeleteAlias(ctx, a.ID); err != nil {
		t.Fatalf("DeleteAlias: %v", err)
	}
}
