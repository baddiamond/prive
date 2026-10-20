package engines

import (
	"testing"

	"github.com/baddiamond/prive/internal/core"
)

type stubEngine struct{ m core.Manifest }

func (s stubEngine) Manifest() core.Manifest { return s.m }
func (s stubEngine) NewSession(core.Account, core.SessionDeps) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

func manifest(id string) core.Manifest {
	return core.Manifest{ID: id, Name: id, ShortName: "XX", LoginMethods: []core.LoginMethod{core.LoginQR}}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stubEngine{manifest("beta")}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(stubEngine{manifest("alpha")}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(stubEngine{manifest("alpha")}); err == nil {
		t.Error("duplicate id accepted")
	}
	if err := r.Register(stubEngine{manifest("Bad ID")}); err == nil {
		t.Error("invalid manifest accepted")
	}
	if _, ok := r.Engine("alpha"); !ok {
		t.Error("alpha not found")
	}
	if _, ok := r.Engine("gamma"); ok {
		t.Error("unknown engine found")
	}
	got := r.Manifests()
	if len(got) != 2 || got[0].ID != "alpha" || got[1].ID != "beta" {
		t.Errorf("Manifests() = %+v, want alpha, beta", got)
	}
}
