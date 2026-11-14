package types

import "testing"

// These tests preserve internal compatibility behavior rather than PostgreSQL date semantics.
func TestCheckDatetimeTypeInternal(t *testing.T) {
	tests := []struct {
		name             string
		input            PgTime
		allowZeroInDate  bool
		allowInvalidDate bool
		wantErr          bool
	}{
		{name: "valid datetime", input: PgTime{year: 2000, month: 1, day: 2, hour: 3, minute: 4, second: 5}},
		{name: "maximum datetime", input: MaxDatetime},
		{name: "zero datetime"},
		{name: "year out of range", input: PgTime{year: 10000, month: 1, day: 1}, wantErr: true},
		{name: "month out of range", input: PgTime{year: 2000, month: 13, day: 1}, wantErr: true},
		{name: "day out of range", input: PgTime{year: 2000, month: 1, day: 32}, wantErr: true},
		{name: "hour out of range", input: PgTime{year: 2000, month: 1, day: 1, hour: 24}, wantErr: true},
		{name: "minute out of range", input: PgTime{year: 2000, month: 1, day: 1, minute: 60}, wantErr: true},
		{name: "second out of range", input: PgTime{year: 2000, month: 1, day: 1, second: 60}, wantErr: true},
		{name: "zero month rejected", input: PgTime{year: 2000, day: 5}, wantErr: true},
		{name: "zero month allowed", input: PgTime{year: 2000, day: 5}, allowZeroInDate: true},
		{name: "leap day", input: PgTime{year: 2000, month: 2, day: 29}},
		{name: "non leap day rejected", input: PgTime{year: 2001, month: 2, day: 29}, wantErr: true},
		{name: "invalid date rejected", input: PgTime{year: 2000, month: 2, day: 30}, wantErr: true},
		{name: "invalid date allowed", input: PgTime{year: 2000, month: 2, day: 30}, allowInvalidDate: true},
		{
			name:             "invalid date still bounded",
			input:            PgTime{year: 2000, month: 1, day: 32},
			allowInvalidDate: true,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDatetimeType(tt.input, tt.allowZeroInDate, tt.allowInvalidDate)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkDatetimeType(%+v) = %v; want error %t", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestStrToDateAndPgTimeFixInternal(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		format string
		want   PgTime
	}{
		{
			name:   "microseconds",
			input:  "20000101010203123456",
			format: "%Y%m%d%H%i%s%f",
			want:   PgTime{year: 2000, month: 1, day: 1, hour: 1, minute: 2, second: 3, microsecond: 123456},
		},
		{
			name:   "midnight am",
			input:  "120000AM",
			format: "%I%i%s%p",
			want:   PgTime{year: 2000, month: 1, day: 1},
		},
		{
			name:   "noon pm",
			input:  "120000PM",
			format: "%I%i%s%p",
			want:   PgTime{year: 2000, month: 1, day: 1, hour: 12},
		},
		{
			name:   "morning am",
			input:  "090000AM",
			format: "%I%i%s%p",
			want:   PgTime{year: 2000, month: 1, day: 1, hour: 9},
		},
		{
			name:   "evening pm",
			input:  "090000PM",
			format: "%I%i%s%p",
			want:   PgTime{year: 2000, month: 1, day: 1, hour: 21},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PgTime{year: 2000, month: 1, day: 1}
			ctx := map[string]int{}
			if !strToDate(&got, tt.input, tt.format, ctx) {
				t.Fatalf("strToDate(%q, %q) = false, want true", tt.input, tt.format)
			}
			if err := pgTimeFix(&got, ctx); err != nil {
				t.Fatalf("pgTimeFix(%q) = %v, want nil", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("got time %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUnfinishedTokenHelpersReject(t *testing.T) {
	tests := []struct {
		name  string
		input string
		parse func(*PgTime, string, map[string]int) (string, bool)
	}{
		{name: "recognized weekday", input: "Sun", parse: abbreviatedWeekday},
		{name: "unknown weekday", input: "Foo", parse: abbreviatedWeekday},
		{name: "ordinal suffix", input: "1st", parse: dayOfMonthWithSuffix},
		{name: "teen ordinal", input: "15th", parse: dayOfMonthWithSuffix},
		{name: "mismatched suffix", input: "1nd", parse: dayOfMonthWithSuffix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := PgTime{year: 2000, month: 1, day: 1}
			got := want
			ctx := map[string]int{}
			remain, ok := tt.parse(&got, tt.input, ctx)
			if ok || remain != tt.input {
				t.Fatalf("parse(%q) = (%q, %t), want (%q, false)", tt.input, remain, ok, tt.input)
			}
			if got != want || len(ctx) != 0 {
				t.Fatalf("rejection changed time or context: %+v, %v", got, ctx)
			}
		})
	}

	t.Run("ordinal token", func(t *testing.T) {
		want := PgTime{year: 2000, month: 1, day: 1}
		got := want
		ctx := map[string]int{}
		if strToDate(&got, "1st", "%D", ctx) {
			t.Fatal("strToDate accepted an unsupported ordinal token")
		}
		if got != want || len(ctx) != 0 {
			t.Fatalf("rejection changed time or context: %+v, %v", got, ctx)
		}
	})
}
