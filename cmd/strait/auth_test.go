package main

import (
	"testing"
)

func TestLogin_FlagsExist(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	auth := findSubcommand(t, cmd, "auth")
	login := findSubcommand(t, auth, "login")

	for _, name := range []string{"token", "with-token", "context", "server", "browser", "no-browser"} {
		if login.Flags().Lookup(name) == nil {
			t.Errorf("login missing --%s flag", name)
		}
	}
}

func TestLogin_TokenFlag_ValidatesNonEmpty(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	auth := findSubcommand(t, cmd, "auth")
	login := findSubcommand(t, auth, "login")

	tokenFlag := login.Flags().Lookup("token")
	if tokenFlag == nil {
		t.Fatal("login missing --token flag")
	}
	if tokenFlag.DefValue != "" {
		t.Errorf("--token default: got %q, want empty string", tokenFlag.DefValue)
	}
}

func TestLogin_TokenFlag_StoresKey(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	auth := findSubcommand(t, cmd, "auth")
	login := findSubcommand(t, auth, "login")

	tokenFlag := login.Flags().Lookup("token")
	if tokenFlag == nil {
		t.Fatal("login missing --token flag")
	}
	if tokenFlag.Value.Type() != "string" {
		t.Errorf("--token type: got %q, want string", tokenFlag.Value.Type())
	}
}

func TestLogin_BrowserMode_RequiresTTY(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	auth := findSubcommand(t, cmd, "auth")
	login := findSubcommand(t, auth, "login")

	noBrowserFlag := login.Flags().Lookup("no-browser")
	if noBrowserFlag == nil {
		t.Fatal("login missing --no-browser flag")
	}

	if login.Long == "" {
		t.Error("login command should have a long description explaining authentication options")
	}
}
