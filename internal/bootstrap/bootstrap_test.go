package bootstrap

import (
	"context"
	"testing"

	"github.com/baddiamond/prive/internal/engines"
	"github.com/baddiamond/prive/internal/keys"
)

func TestOpenAtIsRepeatable(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	kc := keys.NewMemoryKeychain()

	l, err := OpenAt(ctx, dir, kc)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Store.SetSetting(ctx, "k", "v"); err != nil {
		t.Fatal(err)
	}
	cfg := l.ManagerConfig(engines.NewRegistry(), nil, nil)
	if cfg.OpenDB == nil || cfg.Repo == nil || cfg.DataDir != dir {
		t.Errorf("incomplete manager config: %+v", cfg)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := OpenAt(ctx, dir, kc)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if v, _, _ := again.Store.Setting(ctx, "k"); v != "v" {
		t.Errorf("reopen lost data: %q", v)
	}
}
