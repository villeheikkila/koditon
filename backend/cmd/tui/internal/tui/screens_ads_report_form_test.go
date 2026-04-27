package tui

import "testing"

func TestAdsReportFormBuildParamsValid(t *testing.T) {
	screen := newAdsReportFormScreen(&appContext{styles: defaultStyles()}, action{Title: "Search Reports"}, nil, "x").(*adsReportFormScreen)
	screen.form.fields[0].Input.SetValue("helsinki")
	screen.form.fields[3].Input.SetValue("100000")
	screen.form.fields[4].Input.SetValue("200000")
	screen.form.fields[10].ChoiceIndex = 1
	params, _, err := screen.buildParams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.PageSize != 50 {
		t.Fatalf("expected page size 50, got %d", params.PageSize)
	}
	if params.Query != "helsinki" {
		t.Fatalf("expected query helsinki, got %s", params.Query)
	}
}

func TestAdsReportFormBuildParamsInvalidRange(t *testing.T) {
	screen := newAdsReportFormScreen(&appContext{styles: defaultStyles()}, action{Title: "Search Reports"}, nil, "x").(*adsReportFormScreen)
	screen.form.fields[3].Input.SetValue("300000")
	screen.form.fields[4].Input.SetValue("200000")
	_, _, err := screen.buildParams()
	if err == nil {
		t.Fatalf("expected range validation error")
	}
}
