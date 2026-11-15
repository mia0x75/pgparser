package types

import (
	"testing"
)

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
