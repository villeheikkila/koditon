package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScreenSnapshots(t *testing.T) {
	ctx := &appContext{styles: defaultStyles(), runtime: newJobRuntime(), subsystems: buildSubsystems()}
	home := newHomeScreen(ctx)
	home.Resize(100, 34)
	actions := newActionsScreen(ctx, 0)
	actions.Resize(100, 34)
	prompt := newPromptScreen(ctx, ctx.subsystems[0].Actions[1], nil, 0, "Subsystems > Frontdoor > Actions")
	prompt.Resize(100, 34)
	city := newCityPickerScreen(ctx, ctx.subsystems[2].Actions[3], nil, "Subsystems > Prices > Actions")
	city.Resize(100, 34)
	cases := []struct {
		name   string
		screen Screen
	}{
		{name: "home", screen: home},
		{name: "actions", screen: actions},
		{name: "prompt", screen: prompt},
		{name: "city_loading", screen: city},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.screen.ShellState()
			state.Body = tc.screen.View()
			got := renderShell(ctx.styles, state)
			assertGolden(t, tc.name+".golden", got)
		})
	}
}

func assertGolden(t *testing.T, file string, got string) {
	t.Helper()
	path := filepath.Join("testdata", file)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(wantBytes) {
		t.Fatalf("snapshot mismatch for %s", file)
	}
}
