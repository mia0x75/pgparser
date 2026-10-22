package types

import (
	"math"
	"strconv"
	"strings"

	"github.com/mia0x75/pgparser/errors"
)

const (
	// UnspecifiedFsp is the unspecified fractional seconds part.
	UnspecifiedFsp = -1
	// MaxFsp is the maximum digit of fractional seconds part.
	MaxFsp = 6
	// MinFsp is the minimum digit of fractional seconds part.
	MinFsp = 0
	// DefaultFsp is the default digit of fractional seconds part.
	// MySQL use 0 as the default Fsp.
	DefaultFsp = 0
)

// CheckFsp checks whether fsp is in valid range.
func CheckFsp(fsp int) (int, error) {
	if fsp == UnspecifiedFsp {
		return DefaultFsp, nil
	}
	if fsp < MinFsp || fsp > MaxFsp {
		return DefaultFsp, errors.Errorf("Invalid fsp %d", fsp)
	}
	return fsp, nil
}

// ParseFrac parses the input string according to fsp, returns the microsecond,
// and also a bool value to indice overflow. eg:
// "999" fsp=2 will overflow.
func ParseFrac(s string, fsp int) (v int, overflow bool, err error) {
	if len(s) == 0 {
		return 0, false, nil
	}

	fsp, err = CheckFsp(fsp)
	if err != nil {
		return 0, false, errors.Trace(err)
	}

	if fsp >= len(s) {
		tmp, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			return 0, false, errors.Trace(e)
		}
		v = int(float64(tmp) * math.Pow10(MaxFsp-len(s)))
		return
	}

	// Round when fsp < string length.
	tmp, e := strconv.ParseInt(s[:fsp+1], 10, 64)
	if e != nil {
		return 0, false, errors.Trace(e)
	}
	tmp = (tmp + 5) / 10

	if float64(tmp) >= math.Pow10(fsp) {
		// overflow
		return 0, true, nil
	}

	// Get the final frac, with 6 digit number
	//  1236 round 3 -> 124 -> 124000
	//  0312 round 2 -> 3 -> 30000
	//  999 round 2 -> 100 -> overflow
	v = int(float64(tmp) * math.Pow10(MaxFsp-fsp))
	return
}

// alignFrac is used to generate alignment frac, like `100` -> `100000`
func alignFrac(s string, fsp int) string {
	sl := len(s)
	if sl < fsp {
		return s + strings.Repeat("0", fsp-sl)
	}

	return s
}
