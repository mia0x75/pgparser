package types

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	gotime "time"
	"unicode"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/postgres"
)

// Portable analogs of some common call errors.
var (
	ErrInvalidTimeFormat      = postgres.ClassTypes.New(postgres.ErrTruncatedWrongValue, "invalid time format: '%v'")
	ErrInvalidYearFormat      = errors.New("invalid year format")
	ErrInvalidYear            = errors.New("invalid year")
	ErrZeroDate               = errors.New("datetime zero in date")
	ErrIncorrectDatetimeValue = postgres.ClassTypes.New(postgres.ErrTruncatedWrongValue, "Incorrect datetime value: '%s'")
	ErrTruncatedWrongValue    = postgres.ClassTypes.New(postgres.ErrTruncatedWrongValue, postgres.PgErrName[postgres.ErrTruncatedWrongValue])
)

// Time format without fractional seconds precision.
const (
	DateFormat = "2006-01-02"
	TimeFormat = "2006-01-02 15:04:05"
	// TimeFSPFormat is time format with fractional seconds precision.
	TimeFSPFormat = "2006-01-02 15:04:05.000000"
)

// These compatibility bounds are preserved; they are not PostgreSQL date/time ranges.
const (
	MinYear int16 = 1901
	MaxYear int16 = 2155
	// MaxDuration is the maximum for duration.
	MaxDuration int64 = 838*10000 + 59*100 + 59
	MinTime           = -gotime.Duration(838*3600+59*60+59) * gotime.Second
	MaxTime           = gotime.Duration(838*3600+59*60+59) * gotime.Second
	// ZeroDatetimeStr is the string representation of a zero datetime.
	ZeroDatetimeStr = "0000-00-00 00:00:00"
	// ZeroDateStr is the string representation of a zero date.
	ZeroDateStr = "0000-00-00"

	TimeMaxHour         = 838
	TimeMaxMinute       = 59
	TimeMaxSecond       = 59
	TimeMaxValue        = TimeMaxHour*10000 + TimeMaxMinute*100 + TimeMaxSecond
	TimeMaxValueSeconds = TimeMaxHour*3600 + TimeMaxMinute*60 + TimeMaxSecond
)

// Zero values for different types.
var (
	// ZeroDuration is the zero value for Duration type.
	ZeroDuration = Duration{Duration: gotime.Duration(0), Fsp: DefaultFsp}

	// ZeroTime is the zero value for TimeInternal type.
	ZeroTime = PgTime{}

	// ZeroDatetime is the zero value for datetime Time.
	ZeroDatetime = Time{
		Time: ZeroTime,
		Type: postgres.TypeDatetime,
		Fsp:  DefaultFsp,
	}

	// ZeroTimestamp is the zero value for timestamp Time.
	ZeroTimestamp = Time{
		Time: ZeroTime,
		Type: postgres.TypeTimestamp,
		Fsp:  DefaultFsp,
	}

	// ZeroDate is the zero value for date Time.
	ZeroDate = Time{
		Time: ZeroTime,
		Type: postgres.TypeDate,
		Fsp:  DefaultFsp,
	}
)

// Timestamp bounds remain limited to the compatibility representation.
var (
	MinDatetime = FromDate(1000, 1, 1, 0, 0, 0, 0)
	MaxDatetime = FromDate(9999, 12, 31, 23, 59, 59, 999999)

	// BoundTimezone is the timezone for min and max timestamp.
	BoundTimezone = gotime.UTC
	MinTimestamp  = Time{
		Time: FromDate(1970, 1, 1, 0, 0, 1, 0),
		Type: postgres.TypeTimestamp,
		Fsp:  DefaultFsp,
	}
	MaxTimestamp = Time{
		Time: FromDate(2038, 1, 19, 3, 14, 7, 999999),
		Type: postgres.TypeTimestamp,
		Fsp:  DefaultFsp,
	}

	// WeekdayNames lists names of weekdays, which are used in builtin time function `dayname`.
	WeekdayNames = []string{
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
		"Sunday",
	}

	// MonthNames lists names of months, which are used in builtin time function `monthname`.
	MonthNames = []string{
		"January", "February",
		"March", "April",
		"May", "June",
		"July", "August",
		"September", "October",
		"November", "December",
	}
)

// FromGoTime rounds to microseconds for the internal time representation.
func FromGoTime(t gotime.Time) PgTime {
	t = t.Round(gotime.Microsecond)
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	microsecond := t.Nanosecond() / 1000
	return FromDate(year, int(month), day, hour, minute, second, microsecond)
}

// FromDate makes a internal time representation from the given date.
func FromDate(year int, month int, day int, hour int, minute int, second int, microsecond int) PgTime {
	return PgTime{
		uint16(year),
		uint8(month),
		uint8(day),
		hour,
		uint8(minute),
		uint8(second),
		uint32(microsecond),
	}
}

// Clock returns the hour, minute, and second within the day specified by t.
func (t Time) Clock() (hour int, minute int, second int) {
	return t.Time.Hour(), t.Time.Minute(), t.Time.Second()
}

// Time is the struct for handling datetime, timestamp and date.
// TODO: check if need a NewTime function to set Fsp default value?
type Time struct {
	Time PgTime
	Type uint8
	// Fsp is short for Fractional Seconds Precision.
	Fsp int
}

// CurrentTime returns current time with type tp.
func CurrentTime(tp uint8) Time {
	return Time{Time: FromGoTime(gotime.Now()), Type: tp, Fsp: 0}
}

// ConvertTimeZone converts the time value from one timezone to another.
// The input time should be a valid timestamp.
func (t *Time) ConvertTimeZone(from, to *gotime.Location) error {
	if !t.IsZero() {
		raw, err := t.Time.GoTime(from)
		if err != nil {
			return errors.Trace(err)
		}
		converted := raw.In(to)
		t.Time = FromGoTime(converted)
	}
	return nil
}

func (t Time) String() string {
	if t.Type == postgres.TypeDate {
		// We control the format, so no error would occur.
		str, _ := t.DateFormat("%Y-%m-%d")
		return str
	}

	str, _ := t.DateFormat("%Y-%m-%d %H:%i:%s")
	if t.Fsp > 0 {
		// The microsecond buffer is 6 digits; clamp so an out-of-range Fsp
		// (e.g. from a `timestamp(7)` typmod) cannot index out of bounds.
		fsp := t.Fsp
		if fsp > 6 {
			fsp = 6
		}
		tmp := fmt.Sprintf(".%06d", t.Time.Microsecond())
		str += tmp[:1+fsp]
	}

	return str
}

// IsZero returns a boolean indicating whether the time is equal to ZeroTime.
func (t Time) IsZero() bool {
	return compareTime(t.Time, ZeroTime) == 0
}

const numberFormat = "%Y%m%d%H%i%s"
const dateFormat = "%Y%m%d"

// ToNumber returns a formatted number.
// e.g,
// 2012-12-12 -> 20121212
// 2012-12-12T10:10:10 -> 20121212101010
// 2012-12-12T10:10:10.123456 -> 20121212101010.123456
func (t Time) ToNumber() *PgDecimal {
	if t.IsZero() {
		return &PgDecimal{}
	}

	// Fix issue #1046
	// Prevents from converting 2012-12-12 to 20121212000000
	var tfStr string
	if t.Type == postgres.TypeDate {
		tfStr = dateFormat
	} else {
		tfStr = numberFormat
	}

	s, _ := t.DateFormat(tfStr)

	if t.Fsp > 0 {
		// Clamp to the 6-digit microsecond buffer; see String above.
		fsp := t.Fsp
		if fsp > 6 {
			fsp = 6
		}
		s1 := fmt.Sprintf("%s.%06d", s, t.Time.Microsecond())
		s = s1[:len(s)+fsp+1]
	}

	// We skip checking error here because time formatted string can be parsed certainly.
	dec := new(PgDecimal)
	_ = dec.FromString([]byte(s))
	return dec
}

// Compare returns an integer comparing the time instant t to o.
// If t is after o, return 1, equal o, return 0, before o, return -1.
func (t Time) Compare(o Time) int {
	return compareTime(t.Time, o.Time)
}

// compareTime compare two PgTime.
// return:
//
//	0: if a == b
//	1: if a > b
//
// -1: if a < b
func compareTime(a, b PgTime) int {
	ta := datetimeToUint64(a)
	tb := datetimeToUint64(b)

	switch {
	case ta < tb:
		return -1
	case ta > tb:
		return 1
	}

	switch {
	case a.Microsecond() < b.Microsecond():
		return -1
	case a.Microsecond() > b.Microsecond():
		return 1
	}

	return 0
}

// TimestampDiff returns t2 - t1 where t1 and t2 are date or datetime expressions.
// The unit for the result (an integer) is given by the unit argument.
// The legal values for unit are "YEAR" "QUARTER" "MONTH" "DAY" "HOUR" "SECOND" and so on.
func TimestampDiff(unit string, t1 Time, t2 Time) int64 {
	return timestampDiff(unit, t1.Time, t2.Time)
}

// ParseDateFormat parses a formatted date string and returns separated components.
func ParseDateFormat(format string) []string {
	format = strings.TrimSpace(format)

	start := 0
	// Initialize `seps` with capacity of 6. The input `format` is typically
	// a date time of the form "2006-01-02 15:04:05", which has 6 numeric parts
	// (the fractional second part is usually removed by `splitDateTime`).
	// Setting `seps`'s capacity to 6 avoids reallocation in this common case.
	seps := make([]string, 0, 6)
	for i := 0; i < len(format); i++ {
		// Date format must start and end with number.
		if i == 0 || i == len(format)-1 {
			if !unicode.IsNumber(rune(format[i])) {
				return nil
			}

			continue
		}

		// Separator is a single none-number char.
		if !unicode.IsNumber(rune(format[i])) {
			if !unicode.IsNumber(rune(format[i-1])) {
				return nil
			}

			seps = append(seps, format[start:i])
			start = i + 1
		}

	}

	seps = append(seps, format[start:])
	return seps
}

// ParseYear parses a formatted string and returns a year number.
func ParseYear(str string) (int16, error) {
	v, err := strconv.ParseInt(str, 10, 16)
	if err != nil {
		return 0, errors.Trace(err)
	}
	y := int16(v)

	switch len(str) {
	case 4:
		// Nothing to do.
	case 2, 1:
		y = int16(adjustYear(int(y)))
	default:
		return 0, errors.Trace(ErrInvalidYearFormat)
	}

	if y < MinYear || y > MaxYear {
		return 0, errors.Trace(ErrInvalidYearFormat)
	}

	return y, nil
}

// adjustYear adjusts year according to y.
func adjustYear(y int) int {
	if y >= 0 && y <= 69 {
		y = 2000 + y
	} else if y >= 70 && y <= 99 {
		y = 1900 + y
	}
	return y
}

// AdjustYear is used for adjusting year and checking its validation.
func AdjustYear(y int64, shouldAdjust bool) (int64, error) {
	if y == 0 && !shouldAdjust {
		return y, nil
	}
	y = int64(adjustYear(int(y)))
	if y < int64(MinYear) || y > int64(MaxYear) {
		return 0, errors.Trace(ErrInvalidYear)
	}

	return y, nil
}

// Duration retains compatibility fractional precision; it is not PostgreSQL's time type.
type Duration struct {
	gotime.Duration
	// Fsp is short for Fractional Seconds Precision.
	Fsp int
}

// Add adds d to d, returns a duration value.
func (d Duration) Add(v Duration) (Duration, error) {
	dsum, err := AddInt64(int64(d.Duration), int64(v.Duration))
	if err != nil {
		return Duration{}, errors.Trace(err)
	}
	if d.Fsp >= v.Fsp {
		return Duration{Duration: gotime.Duration(dsum), Fsp: d.Fsp}, nil
	}
	return Duration{Duration: gotime.Duration(dsum), Fsp: v.Fsp}, nil
}

// Sub subtracts d to d, returns a duration value.
func (d Duration) Sub(v Duration) (Duration, error) {
	dsum, err := SubInt64(int64(d.Duration), int64(v.Duration))
	if err != nil {
		return Duration{}, errors.Trace(err)
	}
	if d.Fsp >= v.Fsp {
		return Duration{Duration: gotime.Duration(dsum), Fsp: d.Fsp}, nil
	}
	return Duration{Duration: gotime.Duration(dsum), Fsp: v.Fsp}, nil
}

// String returns the time formatted using default TimeFormat and fsp.
func (d Duration) String() string {
	var buf bytes.Buffer

	sign, hours, minutes, seconds, fraction := splitDuration(d.Duration)
	if sign < 0 {
		buf.WriteByte('-')
	}

	fmt.Fprintf(&buf, "%02d:%02d:%02d", hours, minutes, seconds)
	if d.Fsp > 0 {
		buf.WriteString(".")
		buf.WriteString(d.formatFrac(fraction))
	}

	p := buf.String()

	return p
}

func (d Duration) formatFrac(frac int) string {
	s := fmt.Sprintf("%06d", frac)
	return s[0:d.Fsp]
}

// ToNumber changes duration to number format.
// e.g,
// 10:10:10 -> 101010
func (d Duration) ToNumber() *PgDecimal {
	sign, hours, minutes, seconds, fraction := splitDuration(d.Duration)
	var (
		s       string
		signStr string
	)

	if sign < 0 {
		signStr = "-"
	}

	if d.Fsp == 0 {
		s = fmt.Sprintf("%s%02d%02d%02d", signStr, hours, minutes, seconds)
	} else {
		s = fmt.Sprintf("%s%02d%02d%02d.%s", signStr, hours, minutes, seconds, d.formatFrac(fraction))
	}

	// We skip checking error here because time formatted string can be parsed certainly.
	dec := new(PgDecimal)
	_ = dec.FromString([]byte(s))
	return dec
}

// Compare returns an integer comparing the Duration instant t to o.
// If d is after o, return 1, equal o, return 0, before o, return -1.
func (d Duration) Compare(o Duration) int {
	switch {
	case d.Duration > o.Duration:
		return 1
	case d.Duration == o.Duration:
		return 0
	default:
		return -1
	}
}

// Hour returns current hour.
// e.g, hour("11:11:11") -> 11
func (d Duration) Hour() int {
	_, hour, _, _, _ := splitDuration(d.Duration)
	return hour
}

// Minute returns current minute.
// e.g, hour("11:11:11") -> 11
func (d Duration) Minute() int {
	_, _, minute, _, _ := splitDuration(d.Duration)
	return minute
}

// Second returns current second.
// e.g, hour("11:11:11") -> 11
func (d Duration) Second() int {
	_, _, _, second, _ := splitDuration(d.Duration)
	return second
}

// MicroSecond returns current microsecond.
// e.g, hour("11:11:11.11") -> 110000
func (d Duration) MicroSecond() int {
	_, _, _, _, frac := splitDuration(d.Duration)
	return frac
}

// TruncateOverflowPostgreSQLTime clamps to the compatibility duration bounds, not PostgreSQL time limits.
func TruncateOverflowPostgreSQLTime(d gotime.Duration) (gotime.Duration, error) {
	if d > MaxTime {
		return MaxTime, ErrTruncatedWrongVal.GenWithStackByArgs("time", d.String())
	} else if d < MinTime {
		return MinTime, ErrTruncatedWrongVal.GenWithStackByArgs("time", d.String())
	}

	return d, nil
}

func splitDuration(t gotime.Duration) (int, int, int, int, int) {
	sign := 1
	if t < 0 {
		t = -t
		sign = -1
	}

	hours := t / gotime.Hour
	t -= hours * gotime.Hour
	minutes := t / gotime.Minute
	t -= minutes * gotime.Minute
	seconds := t / gotime.Second
	t -= seconds * gotime.Second
	fraction := t / gotime.Microsecond

	return sign, int(hours), int(minutes), int(seconds), int(fraction)
}

var maxDaysInMonth = []int{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

func checkDateType(t PgTime, allowZeroInDate, allowInvalidDate bool) error {
	year, month, day := t.Year(), t.Month(), t.Day()
	if year == 0 && month == 0 && day == 0 {
		return nil
	}

	if !allowZeroInDate && (month == 0 || day == 0) {
		return ErrIncorrectDatetimeValue.GenWithStackByArgs(fmt.Sprintf("%04d-%02d-%02d", year, month, day))
	}

	if err := checkDateRange(t); err != nil {
		return errors.Trace(err)
	}

	if err := checkMonthDay(year, month, day, allowInvalidDate); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func checkDateRange(t PgTime) error {
	if t.Year() < 0 || t.Month() < 0 || t.Day() < 0 {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(t))
	}
	if compareTime(t, MaxDatetime) > 0 {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(t))
	}
	return nil
}

func checkMonthDay(year, month, day int, allowInvalidDate bool) error {
	if month < 0 || month > 12 {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(month))
	}

	maxDay := 31
	if !allowInvalidDate {
		if month > 0 {
			maxDay = maxDaysInMonth[month-1]
		}
		if month == 2 && year%4 != 0 {
			maxDay = 28
		}
	}

	if day < 0 || day > maxDay {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(day))
	}
	return nil
}

func checkDatetimeType(t PgTime, allowZeroInDate, allowInvalidDate bool) error {
	if err := checkDateType(t, allowZeroInDate, allowInvalidDate); err != nil {
		return errors.Trace(err)
	}

	hour, minute, second := t.Hour(), t.Minute(), t.Second()
	if hour < 0 || hour >= 24 {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(hour))
	}
	if minute < 0 || minute >= 60 {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(minute))
	}
	if second < 0 || second >= 60 {
		return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(second))
	}

	return nil
}

// DateFormat returns a textual representation of the time value formatted
// according to layout.
func (t Time) DateFormat(layout string) (string, error) {
	var buf bytes.Buffer
	inPatternMatch := false
	for _, b := range layout {
		if inPatternMatch {
			if err := t.convertDateFormat(b, &buf); err != nil {
				return "", errors.Trace(err)
			}
			inPatternMatch = false
			continue
		}

		// It's not in pattern match now.
		if b == '%' {
			inPatternMatch = true
		} else {
			buf.WriteRune(b)
		}
	}
	return buf.String(), nil
}

var abbrevWeekdayName = []string{
	"Sun", "Mon", "Tue",
	"Wed", "Thu", "Fri", "Sat",
}

func (t Time) convertDateFormat(b rune, buf *bytes.Buffer) error {
	switch b {
	case 'b':
		m := t.Time.Month()
		if m == 0 || m > 12 {
			return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(m))
		}
		buf.WriteString(MonthNames[m-1][:3])
	case 'M':
		m := t.Time.Month()
		if m == 0 || m > 12 {
			return errors.Trace(ErrInvalidTimeFormat.GenWithStackByArgs(m))
		}
		buf.WriteString(MonthNames[m-1])
	case 'm':
		fmt.Fprintf(buf, "%02d", t.Time.Month())
	case 'c':
		fmt.Fprintf(buf, "%d", t.Time.Month())
	case 'D':
		fmt.Fprintf(buf, "%d%s", t.Time.Day(), abbrDayOfMonth(t.Time.Day()))
	case 'd':
		fmt.Fprintf(buf, "%02d", t.Time.Day())
	case 'e':
		fmt.Fprintf(buf, "%d", t.Time.Day())
	case 'j':
		fmt.Fprintf(buf, "%03d", t.Time.YearDay())
	case 'H':
		fmt.Fprintf(buf, "%02d", t.Time.Hour())
	case 'k':
		fmt.Fprintf(buf, "%d", t.Time.Hour())
	case 'h', 'I':
		t := t.Time.Hour()
		if t == 0 || t == 12 {
			fmt.Fprintf(buf, "%02d", 12)
		} else {
			fmt.Fprintf(buf, "%02d", t%12)
		}
	case 'l':
		t := t.Time.Hour()
		if t == 0 || t == 12 {
			fmt.Fprintf(buf, "%d", 12)
		} else {
			fmt.Fprintf(buf, "%d", t%12)
		}
	case 'i':
		fmt.Fprintf(buf, "%02d", t.Time.Minute())
	case 'p':
		hour := t.Time.Hour()
		if hour/12%2 == 0 {
			buf.WriteString("AM")
		} else {
			buf.WriteString("PM")
		}
	case 'r':
		h := t.Time.Hour()
		switch {
		case h == 0:
			fmt.Fprintf(buf, "%02d:%02d:%02d AM", 12, t.Time.Minute(), t.Time.Second())
		case h == 12:
			fmt.Fprintf(buf, "%02d:%02d:%02d PM", 12, t.Time.Minute(), t.Time.Second())
		case h < 12:
			fmt.Fprintf(buf, "%02d:%02d:%02d AM", h, t.Time.Minute(), t.Time.Second())
		default:
			fmt.Fprintf(buf, "%02d:%02d:%02d PM", h-12, t.Time.Minute(), t.Time.Second())
		}
	case 'T':
		fmt.Fprintf(buf, "%02d:%02d:%02d", t.Time.Hour(), t.Time.Minute(), t.Time.Second())
	case 'S', 's':
		fmt.Fprintf(buf, "%02d", t.Time.Second())
	case 'f':
		fmt.Fprintf(buf, "%06d", t.Time.Microsecond())
	case 'U':
		w := t.Time.Week(0)
		fmt.Fprintf(buf, "%02d", w)
	case 'u':
		w := t.Time.Week(1)
		fmt.Fprintf(buf, "%02d", w)
	case 'V':
		w := t.Time.Week(2)
		fmt.Fprintf(buf, "%02d", w)
	case 'v':
		_, w := t.Time.YearWeek(3)
		fmt.Fprintf(buf, "%02d", w)
	case 'a':
		weekday := t.Time.Weekday()
		buf.WriteString(abbrevWeekdayName[weekday])
	case 'W':
		buf.WriteString(t.Time.Weekday().String())
	case 'w':
		fmt.Fprintf(buf, "%d", t.Time.Weekday())
	case 'X':
		year, _ := t.Time.YearWeek(2)
		if year < 0 {
			fmt.Fprintf(buf, "%v", uint64(math.MaxUint32))
		} else {
			fmt.Fprintf(buf, "%04d", year)
		}
	case 'x':
		year, _ := t.Time.YearWeek(3)
		if year < 0 {
			fmt.Fprintf(buf, "%v", uint64(math.MaxUint32))
		} else {
			fmt.Fprintf(buf, "%04d", year)
		}
	case 'Y':
		fmt.Fprintf(buf, "%04d", t.Time.Year())
	case 'y':
		str := fmt.Sprintf("%04d", t.Time.Year())
		buf.WriteString(str[2:])
	default:
		buf.WriteRune(b)
	}

	return nil
}

func abbrDayOfMonth(day int) string {
	var str string
	switch day {
	case 1, 21, 31:
		str = "st"
	case 2, 22:
		str = "nd"
	case 3, 23:
		str = "rd"
	default:
		str = "th"
	}
	return str
}
