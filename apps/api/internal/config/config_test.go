package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFromDotEnv(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("STELLAR_NETWORK_PASSPHRASE", "")
	t.Setenv("STELLAR_RPC_URL", "")
	t.Setenv("STELLAR_HORIZON_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_FORMAT", "")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), strings.Join([]string{
		"APP_ENV=staging",
		"DATABASE_DSN=postgres://postgres:postgres@localhost:5432/nester?sslmode=disable",
		"STELLAR_NETWORK_PASSPHRASE=Test Network",
		"STELLAR_RPC_URL=https://rpc.example.com",
		"STELLAR_HORIZON_URL=https://horizon.example.com",
	}, "\n"))

	chdir(t, dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Environment() != "staging" {
		t.Fatalf("expected environment staging, got %q", cfg.Environment())
	}
	if cfg.Server().Port() != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server().Port())
	}
	if cfg.Log().Format() != "json" {
		t.Fatalf("expected staging to default to json format, got %q", cfg.Log().Format())
	}
	if cfg.Database().PoolSize() != 25 {
		t.Fatalf("expected default pool size 25, got %d", cfg.Database().PoolSize())
	}
	if cfg.Server().GracefulShutdown() != 20*time.Second {
		t.Fatalf("expected default shutdown timeout 20s, got %s", cfg.Server().GracefulShutdown())
	}
}

func TestLoadMissingRequiredFields(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("STELLAR_NETWORK_PASSPHRASE", "")
	t.Setenv("STELLAR_RPC_URL", "")
	t.Setenv("STELLAR_HORIZON_URL", "")

	chdir(t, t.TempDir())

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to fail")
	}

	message := err.Error()
	for _, expected := range []string{
		"DATABASE_DSN is required",
		"STELLAR_NETWORK_PASSPHRASE is required",
		"STELLAR_RPC_URL is required",
		"STELLAR_HORIZON_URL is required",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got %q", expected, message)
		}
	}
}

func TestLoadTypeCoercionErrors(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/nester?sslmode=disable")
	t.Setenv("STELLAR_NETWORK_PASSPHRASE", "Test Network")
	t.Setenv("STELLAR_RPC_URL", "https://rpc.example.com")
	t.Setenv("STELLAR_HORIZON_URL", "https://horizon.example.com")
	t.Setenv("SERVER_PORT", "not-a-number")
	t.Setenv("DATABASE_CONNECTION_TIMEOUT", "forever")

	chdir(t, t.TempDir())

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to fail")
	}

	message := err.Error()
	if !strings.Contains(message, `SERVER_PORT must be an integer, got "not-a-number"`) {
		t.Fatalf("expected integer coercion error, got %q", message)
	}
	if !strings.Contains(message, `DATABASE_CONNECTION_TIMEOUT must be a valid duration, got "forever"`) {
		t.Fatalf("expected duration coercion error, got %q", message)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
