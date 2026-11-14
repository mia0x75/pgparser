package postgres

import "testing"

func TestSQLModeBitValues(t *testing.T) {
	tests := []struct {
		name string
		mode SQLMode
		want SQLMode
	}{
		{name: "before reserved bits", mode: ModeNoFieldOptions, want: 1 << 16},
		{name: "after reserved bits", mode: ModeANSI, want: 1 << 19},
		{name: "last mode", mode: ModeAllowInvalidDates, want: 1 << 33},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mode != tt.want {
				t.Fatalf("got mode %#x, want %#x", tt.mode, tt.want)
			}
		})
	}
}

func TestSQLModeMappings(t *testing.T) {
	const reservedModes SQLMode = 1<<17 | 1<<18
	for name, mode := range Str2SQLMode {
		if mode&reservedModes != 0 {
			t.Fatalf("mode %q uses a removed compatibility bit", name)
		}
	}
	for name, parts := range CombinationSQLMode {
		want, ok := Str2SQLMode[name]
		if !ok {
			t.Fatalf("combination %q has no mode", name)
		}
		for _, part := range parts {
			mode, ok := Str2SQLMode[part]
			if !ok {
				t.Fatalf("combination %q contains unknown mode %q", name, part)
			}
			want |= mode
		}
		got, err := GetSQLMode(FormatSQLModeStr(name))
		if err != nil || got != want {
			t.Fatalf("combination %q: got %#x, %v; want %#x", name, got, err, want)
		}
	}
}
