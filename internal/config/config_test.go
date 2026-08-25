package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAMLWithEnvironmentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  address: ':9000'\nmllp:\n  address: ':3000'\nstorage:\n  database_url: postgres://example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("HTTP_ADDR", ":9100")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.HTTPAddr != ":9100" || config.MLLPAddr != ":3000" || config.DatabaseURL != "postgres://example" {
		t.Fatalf("config = %#v", config)
	}
}
