package modules

import (
	"testing"
)

func TestBatteryModule_parsePmset(t *testing.T) {
	m := BatteryModule{}

	tests := []struct {
		name      string
		output    string
		wantEmpty bool
		wantOK    bool
	}{
		{
			name: "discharging 82%",
			output: `Now drawing from 'Battery Power'
 -InternalBattery-0 (id=7340034)	82%; discharging; 3:42 remaining present: true`,
			wantEmpty: false,
			wantOK:    true,
		},
		{
			name: "charging at 55%",
			output: `Now drawing from 'AC Power'
 -InternalBattery-0 (id=7340034)	55%; charging; 1:20 remaining present: true`,
			wantEmpty: false,
			wantOK:    true,
		},
		{
			name: "charging at 100% — suppress",
			output: `Now drawing from 'AC Power'
 -InternalBattery-0 (id=7340034)	100%; charging; (no estimate) present: true`,
			wantEmpty: true,
			wantOK:    false,
		},
		{
			name:      "empty output",
			output:    "",
			wantEmpty: true,
			wantOK:    false,
		},
		{
			name:      "no percentage",
			output:    "Now drawing from 'AC Power'",
			wantEmpty: true,
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg, ok := m.parsePmset(tt.output)
			if ok != tt.wantOK {
				t.Errorf("ok=%v want %v (seg=%q)", ok, tt.wantOK, seg)
			}
			if tt.wantEmpty && seg != "" {
				t.Errorf("expected empty segment, got %q", seg)
			}
			if !tt.wantEmpty && seg == "" && tt.wantOK {
				t.Errorf("expected non-empty segment")
			}
		})
	}
}

func TestBatteryModule_ShowThreshold(t *testing.T) {
	m := BatteryModule{ShowThreshold: 20}

	// 30% discharging — above threshold, should be suppressed
	output := `-InternalBattery-0 (id=1)	30%; discharging; 2:00 remaining`
	seg, ok := m.parsePmset(output)
	if ok {
		t.Errorf("expected suppressed (pct>threshold), got %q", seg)
	}

	// 15% discharging — below threshold, should show
	output2 := `-InternalBattery-0 (id=1)	15%; discharging; 0:30 remaining`
	seg2, ok2 := m.parsePmset(output2)
	if !ok2 {
		t.Errorf("expected visible (pct<=threshold), got empty")
	}
	if seg2 == "" {
		t.Errorf("expected non-empty segment for 15%% battery")
	}
}

func TestBatteryModule_Name(t *testing.T) {
	m := BatteryModule{}
	if m.Name() != "battery" {
		t.Errorf("Name()=%q want %q", m.Name(), "battery")
	}
}
