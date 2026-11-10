//go:build wasm

package tests

import (
	"testing"

	"github.com/veltylabs/chat_room/seed"
	"webtyp.com/crud"
	"webtyp.com/time"
	"webtyp.com/view"
)

func TestDemo_Valid(t *testing.T) {
	demo, err := seed.Demo()
	if err != nil {
		t.Fatalf("seed.Demo() error: %v", err)
	}
	if err := demo.Validate(); err != nil {
		t.Fatalf("demo.Validate() error: %v", err)
	}
	zone, err := time.LoadZone(demo.Timezone)
	if err != nil {
		t.Fatalf("time.LoadZone(%q) error: %v", demo.Timezone, err)
	}
	caller := view.NewRecords().Caller(crud.LoopbackActingAs(nil, demo.ActingAs, demo.Modules...))
	screens := demo.Screens(view.BrowserDeps{Caller: caller, Zone: zone})
	if len(screens) == 0 {
		t.Fatalf("expected at least one screen, got 0")
	}
}
