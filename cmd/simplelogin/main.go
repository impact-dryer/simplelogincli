package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"simplelogincli/pkg/api"
	"simplelogincli/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var code int
	switch cmd {
	case "set-key":
		code = runSetKey(args, cfg)
	case "whoami":
		code = runWhoAmI(args, cfg)
	case "options":
		code = runOptions(args, cfg)
	case "random":
		code = runRandom(args, cfg)
	case "custom":
		code = runCustom(args, cfg)
	case "help", "-h", "--help":
		usage()
		code = 0
	case "delete", "-d", "--delete":
		code = runDeleteAlias(args, cfg)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
		code = 2
	}
	os.Exit(code)
}

func usage() {
	_, _ = fmt.Println("simplelogincli - Create SimpleLogin email aliases")
	_, _ = fmt.Println()
	_, _ = fmt.Println("Usage:")
	_, _ = fmt.Println("  simplelogin <command> [flags]")
	_, _ = fmt.Println()
	_, _ = fmt.Println("Commands:")
	_, _ = fmt.Println("  set-key     Store API key and base URL")
	_, _ = fmt.Println("  whoami      Show account info for the current API key")
	_, _ = fmt.Println("  options     List available alias suffix options")
	_, _ = fmt.Println("  random      Create a random alias")
	_, _ = fmt.Println("  custom      Create a custom alias from prefix + suffix")
	_, _ = fmt.Println()
	_, _ = fmt.Println("Global env vars:")
	_, _ = fmt.Println("  SIMPLELOGIN_API_KEY   API key (overrides stored key)")
	_, _ = fmt.Println("  SIMPLELOGIN_BASE_URL  Base URL (default:", config.DefaultBaseURL, ")")
	_, _ = fmt.Println()
	_, _ = fmt.Println("Run 'simplelogin <command> -h' for command-specific flags.")
}

func runSetKey(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("set-key", flag.ExitOnError)
	key := fs.String("api-key", "", "API key to store (or use SIMPLELOGIN_API_KEY env)")
	baseURL := fs.String("base-url", cfg.BaseURL, "SimpleLogin base URL")
	_ = fs.Parse(args)
	if *key == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--api-key is required (or set SIMPLELOGIN_API_KEY)")
		return 2
	}
	cfg.APIKey = *key
	cfg.BaseURL = *baseURL
	if err := config.Save(cfg); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to save config:", err)
		return 1
	}
	_, _ = fmt.Println("API key saved.")
	return 0
}

func runWhoAmI(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("whoami", flag.ExitOnError)
	baseURL := fs.String("base-url", cfg.BaseURL, "SimpleLogin base URL")
	apiKey := fs.String("api-key", cfg.APIKey, "API key (overrides stored key)")
	_ = fs.Parse(args)
	if *apiKey == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Missing API key. Use set-key or --api-key or SIMPLELOGIN_API_KEY.")
		return 2
	}
	c := api.NewClient(*baseURL, *apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ui, err := c.UserInfo(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = fmt.Printf("%s (%s) premium=%v\n", ui.Name, ui.Email, ui.IsPremium)
	return 0
}

func runOptions(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("options", flag.ExitOnError)
	baseURL := fs.String("base-url", cfg.BaseURL, "SimpleLogin base URL")
	apiKey := fs.String("api-key", cfg.APIKey, "API key (overrides stored key)")
	hostname := fs.String("hostname", "", "Website hostname to tailor suggestions")
	_ = fs.Parse(args)
	if *apiKey == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Missing API key. Use set-key or --api-key or env.")
		return 2
	}
	c := api.NewClient(*baseURL, *apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := c.AliasOptions(ctx, *hostname)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	sort.Slice(res.Suffixes, func(i, j int) bool { return res.Suffixes[i].Suffix < res.Suffixes[j].Suffix })
	_, _ = fmt.Println("can_create:", res.CanCreate)
	_, _ = fmt.Println("prefix_suggestion:", res.PrefixSuggestion)
	_, _ = fmt.Println("suffixes:")
	for _, s := range res.Suffixes {
		kind := "public"
		if s.IsCustom {
			kind = "custom"
		}
		prem := ""
		if s.IsPremium {
			prem = " (premium)"
		}
		_, _ = fmt.Printf("  - %s [%s]%s\n", s.Suffix, kind, prem)
	}
	return 0
}

func runRandom(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("random", flag.ExitOnError)
	baseURL := fs.String("base-url", cfg.BaseURL, "SimpleLogin base URL")
	apiKey := fs.String("api-key", cfg.APIKey, "API key (overrides stored key)")
	hostname := fs.String("hostname", "", "Website hostname to attach to the alias creation request")
	mode := fs.String("mode", "", "Random alias mode: uuid or word (optional; defaults to user setting)")
	note := fs.String("note", "", "Optional note for the alias")
	_ = fs.Parse(args)
	if *apiKey == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Missing API key. Use set-key or --api-key or env.")
		return 2
	}
	c := api.NewClient(*baseURL, *apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var notePtr *string
	if strings.TrimSpace(*note) != "" {
		n := *note
		notePtr = &n
	}
	a, err := c.CreateRandomAlias(ctx, *hostname, *mode, notePtr)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = fmt.Println(a.Email)
	return 0
}

func runDeleteAlias(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	baseURL := fs.String("base-url", cfg.BaseURL, "SimpleLogin base URL")
	apiKey := fs.String("api-key", cfg.APIKey, "API key (overrides stored key)")
	hostname := fs.String("hostname", "", "Website hostname to attach to the alias creation request")
	email := fs.String("email", "", "Email of the alias to delete (required)")
	_ = fs.Parse(args)
	if *apiKey == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Missing API key. Use set-key or --api-key or env.")
		return 2
	}
	if *email == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--email is required")
		return 2
	}
	c := api.NewClient(*baseURL, *apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.DeleteAliasByEmail(ctx, *hostname, *email); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = fmt.Println("Alias deleted:", *email)
	return 0
}
func runCustom(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("custom", flag.ExitOnError)
	baseURL := fs.String("base-url", cfg.BaseURL, "SimpleLogin base URL")
	apiKey := fs.String("api-key", cfg.APIKey, "API key (overrides stored key)")
	hostname := fs.String("hostname", "", "Website hostname to attach to the alias creation request")
	prefix := fs.String("prefix", "", "Alias prefix to use (required)")
	signedSuffix := fs.String("signed-suffix", "", "Signed suffix token (from options)")
	suffix := fs.String("suffix", "", "Plain suffix to select from options (will auto-pick matching signed suffix)")
	mailboxIDsCSV := fs.String("mailbox-ids", "", "Comma-separated mailbox IDs owning the alias (defaults to default mailbox)")
	note := fs.String("note", "", "Optional note")
	name := fs.String("name", "", "Optional alias name")
	_ = fs.Parse(args)
	if *apiKey == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Missing API key. Use set-key or --api-key or env.")
		return 2
	}
	if *prefix == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--prefix is required")
		return 2
	}
	c := api.NewClient(*baseURL, *apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	ss := strings.TrimSpace(*signedSuffix)
	if ss == "" {
		if strings.TrimSpace(*suffix) == "" {
			opt, err := c.AliasOptions(ctx, *hostname)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				return 1
			}
			if len(opt.Suffixes) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "no suffixes available")
				return 1
			}
			sort.Slice(opt.Suffixes, func(i, j int) bool { return opt.Suffixes[i].Suffix < opt.Suffixes[j].Suffix })
			_, _ = fmt.Println("Available suffixes:")
			for i, s := range opt.Suffixes {
				kind := "public"
				if s.IsCustom {
					kind = "custom"
				}
				prem := ""
				if s.IsPremium {
					prem = " (premium)"
				}
				_, _ = fmt.Printf("  %2d) %s [%s]%s\n", i+1, s.Suffix, kind, prem)
			}
			_, _ = fmt.Print("Pick a suffix [1-", len(opt.Suffixes), "]: ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			idx, err := strconv.Atoi(line)
			if err != nil || idx < 1 || idx > len(opt.Suffixes) {
				_, _ = fmt.Fprintln(os.Stderr, "invalid selection")
				return 2
			}
			ss = opt.Suffixes[idx-1].SignedSuffix
		} else {
			opt, err := c.AliasOptions(ctx, *hostname)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				return 1
			}
			for _, s := range opt.Suffixes {
				if s.Suffix == *suffix {
					ss = s.SignedSuffix
					break
				}
			}
			if ss == "" {
				_, _ = fmt.Fprintf(os.Stderr, "suffix %q not found in available options\n", *suffix)
				return 2
			}
		}
	}
	var ids []int
	if strings.TrimSpace(*mailboxIDsCSV) != "" {
		parts := strings.Split(*mailboxIDsCSV, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			v, err := strconv.Atoi(p)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "invalid mailbox id: %q\n", p)
				return 2
			}
			ids = append(ids, v)
		}
	} else {
		mid, err := c.DefaultMailboxID(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "failed to determine default mailbox:", err)
			return 1
		}
		ids = []int{mid}
	}
	var notePtr, namePtr *string
	if strings.TrimSpace(*note) != "" {
		n := *note
		notePtr = &n
	}
	if strings.TrimSpace(*name) != "" {
		n := *name
		namePtr = &n
	}
	a, err := c.CreateCustomAlias(ctx, *hostname, *prefix, ss, ids, notePtr, namePtr)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = fmt.Println(a.Email)
	return 0
}
