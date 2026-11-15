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
	"github.com/mia0x75/pgparser/sqlerr"
)

// Portable analogs of some common call errors.
var (
	ErrInvalidTimeFormat      = sqlerr.ClassTypes.New(postgres.ErrTruncatedWrongValue, "invalid time format: '%v'")
	ErrInvalidYearFormat      = errors.New("invalid year format")
	ErrInvalidYear            = errors.New("invalid year")
	ErrZeroDate               = errors.New("datetime zero in date")
	ErrIncorrectDatetimeValue = sqlerr.ClassTypes.New(postgres.ErrTruncatedWrongValue, "Incorrect datetime value: '%s'")
	ErrTruncatedWrongValue    = sqlerr.ClassTypes.New(postgres.ErrTruncatedWrongValue, postgres.PgErrName[postgres.ErrTruncatedWrongValue])
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
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	// Nanosecond plus 500 then divided 1000 means rounding to microseconds.
	microsecond := (t.Nanosecond() + 500) / 1000
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

// MaxPostgreSQLTime uses the compatibility duration bound, not PostgreSQL's time-of-day range.
func MaxPostgreSQLTime(fsp int) Time {
	return Time{Time: FromDate(0, 0, 0, TimeMaxHour, TimeMaxMinute, TimeMaxSecond, 0), Type: postgres.TypeDuration, Fsp: fsp}
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
		tmp := fmt.Sprintf(".%06d", t.Time.Microsecond())
		str += tmp[:1+t.Fsp]
	}

	return str
}

// IsZero returns a boolean indicating whether the time is equal to ZeroTime.
func (t Time) IsZero() bool {
	return compareTime(t.Time, ZeroTime) == 0
}

// InvalidZero returns a boolean indicating whether the month or day is zero.
func (t Time) InvalidZero() bool {
	return t.Time.Month() == 0 || t.Time.Day() == 0
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
		s1 := fmt.Sprintf("%s.%06d", s, t.Time.Microsecond())
		s = s1[:len(s)+t.Fsp+1]
	}

	// We skip checking error here because time formatted string can be parsed certainly.
	dec := new(PgDecimal)
	_ = dec.FromString([]byte(s))
	return dec
}

// ConvertToDuration discards the date so only the time-of-day contributes to the duration.
// e.g,
// 2012-12-12T10:10:10 -> 10:10:10
// 2012-12-12 -> 0
func (t Time) ConvertToDuration() (Duration, error) {
	if t.IsZero() {
		return ZeroDuration, nil
	}

	hour, minute, second := t.Clock()
	frac := t.Time.Microsecond() * 1000

	d := gotime.Duration(hour*3600+minute*60+second)*gotime.Second + gotime.Duration(frac)
	// TODO: check convert validation
	return Duration{Duration: d, Fsp: t.Fsp}, nil
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

// GetFsp gets the fsp of a string.
func GetFsp(s string) (fsp int) {
	fsp = len(s) - strings.LastIndex(s, ".") - 1
	if fsp == len(s) {
		fsp = 0
	} else if fsp > 6 {
		fsp = 6
	}
	return
}

// RoundFrac rounds fractional seconds precision with new fsp and returns a new one.
// We will use the “round half up” rule, e.g, >= 0.5 -> 1, < 0.5 -> 0,
// so 2011:11:11 10:10:10.888888 round 0 -> 2011:11:11 10:10:11
// and 2011:11:11 10:10:10.111111 round 0 -> 2011:11:11 10:10:10
func RoundFrac(t gotime.Time, fsp int) (gotime.Time, error) {
	_, err := CheckFsp(fsp)
	if err != nil {
		return t, errors.Trace(err)
	}
	return t.Round(gotime.Duration(math.Pow10(9-fsp)) * gotime.Nanosecond), nil
}

// ToPackedUint encodes Time to a packed uint64 value.
//
//	 1 bit  0
//	17 bits year*13+month   (year 0-9999, month 0-12)
//	 5 bits day             (0-31)
//	 5 bits hour            (0-23)
//	 6 bits minute          (0-59)
//	 6 bits second          (0-59)
//	24 bits microseconds    (0-999999)
//
//	Total: 64 bits = 8 bytes
//
//	0YYYYYYY.YYYYYYYY.YYdddddh.hhhhmmmm.mmssssss.ffffffff.ffffffff.ffffffff
func (t Time) ToPackedUint() (uint64, error) {
	tm := t.Time
	if t.IsZero() {
		return 0, nil
	}
	year, month, day := tm.Year(), tm.Month(), tm.Day()
	hour, minute, sec := tm.Hour(), tm.Minute(), tm.Second()
	ymd := uint64(((year*13 + month) << 5) | day)
	hms := uint64(hour<<12 | minute<<6 | sec)
	micro := uint64(tm.Microsecond())
	return ((ymd<<17 | hms) << 24) | micro, nil
}

// FromPackedUint decodes Time from a packed uint64 value.
func (t *Time) FromPackedUint(packed uint64) error {
	if packed == 0 {
		t.Time = ZeroTime
		return nil
	}
	ymdhms := packed >> 24
	ymd := ymdhms >> 17
	day := int(ymd & (1<<5 - 1))
	ym := ymd >> 5
	month := int(ym % 13)
	year := int(ym / 13)

	hms := ymdhms & (1<<17 - 1)
	second := int(hms & (1<<6 - 1))
	minute := int((hms >> 6) & (1<<6 - 1))
	hour := int(hms >> 12)
	microsec := int(packed % (1 << 24))

	t.Time = FromDate(year, month, day, hour, minute, second, microsec)

	return nil
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

// RoundFrac rounds fractional seconds precision with new fsp and returns a new one.
// We will use the “round half up” rule, e.g, >= 0.5 -> 1, < 0.5 -> 0,
// so 10:10:10.999999 round 0 -> 10:10:11
// and 10:10:10.000000 round 0 -> 10:10:10
func (d Duration) RoundFrac(fsp int) (Duration, error) {
	fsp, err := CheckFsp(fsp)
	if err != nil {
		return d, errors.Trace(err)
	}

	if fsp == d.Fsp {
		return d, nil
	}

	n := gotime.Date(0, 0, 0, 0, 0, 0, 0, gotime.Local)
	nd := n.Add(d.Duration).Round(gotime.Duration(math.Pow10(9-fsp)) * gotime.Nanosecond).Sub(n)
	return Duration{Duration: nd, Fsp: fsp}, nil
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

// TimeFromDays Converts a day number to a date.
func TimeFromDays(num int64) Time {
	if num < 0 {
		return Time{
			Time: FromDate(0, 0, 0, 0, 0, 0, 0),
			Type: postgres.TypeDate,
			Fsp:  0,
		}
	}
	year, month, day := getDateFromDaynr(uint(num))

	return Time{
		Time: FromDate(int(year), int(month), int(day), 0, 0, 0, 0),
		Type: postgres.TypeDate,
		Fsp:  0,
	}
}

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

// ExtractDatetimeNum extracts time value number from datetime unit and format.
func ExtractDatetimeNum(t *Time, unit string) (int64, error) {
	switch strings.ToUpper(unit) {
	case "DAY":
		return int64(t.Time.Day()), nil
	case "WEEK":
		week := t.Time.Week(0)
		return int64(week), nil
	case "MONTH":
		// TODO: Consider time_zone variable.
		t1, err := t.Time.GoTime(gotime.Local)
		if err != nil {
			return 0, errors.Trace(err)
		}
		return int64(t1.Month()), nil
	case "QUARTER":
		m := int64(t.Time.Month())
		// 1 - 3 -> 1
		// 4 - 6 -> 2
		// 7 - 9 -> 3
		// 10 - 12 -> 4
		return (m + 2) / 3, nil
	case "YEAR":
		return int64(t.Time.Year()), nil
	case "DAY_MICROSECOND":
		h, m, s := t.Clock()
		d := t.Time.Day()
		return int64(d*1000000+h*10000+m*100+s)*1000000 + int64(t.Time.Microsecond()), nil
	case "DAY_SECOND":
		h, m, s := t.Clock()
		d := t.Time.Day()
		return int64(d)*1000000 + int64(h)*10000 + int64(m)*100 + int64(s), nil
	case "DAY_MINUTE":
		h, m, _ := t.Clock()
		d := t.Time.Day()
		return int64(d)*10000 + int64(h)*100 + int64(m), nil
	case "DAY_HOUR":
		h, _, _ := t.Clock()
		d := t.Time.Day()
		return int64(d)*100 + int64(h), nil
	case "YEAR_MONTH":
		y, m := t.Time.Year(), t.Time.Month()
		return int64(y)*100 + int64(m), nil
	default:
		return 0, errors.Errorf("invalid unit %s", unit)
	}
}

// ExtractDurationNum extracts duration value number from duration unit and format.
func ExtractDurationNum(d *Duration, unit string) (int64, error) {
	switch strings.ToUpper(unit) {
	case "MICROSECOND":
		return int64(d.MicroSecond()), nil
	case "SECOND":
		return int64(d.Second()), nil
	case "MINUTE":
		return int64(d.Minute()), nil
	case "HOUR":
		return int64(d.Hour()), nil
	case "SECOND_MICROSECOND":
		return int64(d.Second())*1000000 + int64(d.MicroSecond()), nil
	case "MINUTE_MICROSECOND":
		return int64(d.Minute())*100000000 + int64(d.Second())*1000000 + int64(d.MicroSecond()), nil
	case "MINUTE_SECOND":
		return int64(d.Minute()*100 + d.Second()), nil
	case "HOUR_MICROSECOND":
		return int64(d.Hour())*10000000000 + int64(d.Minute())*100000000 + int64(d.Second())*1000000 + int64(d.MicroSecond()), nil
	case "HOUR_SECOND":
		return int64(d.Hour())*10000 + int64(d.Minute())*100 + int64(d.Second()), nil
	case "HOUR_MINUTE":
		return int64(d.Hour())*100 + int64(d.Minute()), nil
	default:
		return 0, errors.Errorf("invalid unit %s", unit)
	}
}

func extractSingleTimeValue(unit string, format string) (int64, int64, int64, float64, error) {
	fv, err := strconv.ParseFloat(format, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}
	iv := int64(math.Round(fv))

	switch strings.ToUpper(unit) {
	case "MICROSECOND":
		return 0, 0, 0, fv * float64(gotime.Microsecond), nil
	case "SECOND":
		return 0, 0, 0, fv * float64(gotime.Second), nil
	case "MINUTE":
		return 0, 0, 0, float64(iv * int64(gotime.Minute)), nil
	case "HOUR":
		return 0, 0, 0, float64(iv * int64(gotime.Hour)), nil
	case "DAY":
		return 0, 0, iv, 0, nil
	case "WEEK":
		return 0, 0, 7 * iv, 0, nil
	case "MONTH":
		return 0, iv, 0, 0, nil
	case "QUARTER":
		return 0, 3 * iv, 0, 0, nil
	case "YEAR":
		return iv, 0, 0, 0, nil
	}

	return 0, 0, 0, 0, errors.Errorf("invalid singel timeunit - %s", unit)
}

// extractSecondMicrosecond extracts second and microsecond from a string and its format is `SS.FFFFFF`.
func extractSecondMicrosecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, ".")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	microseconds, err := strconv.ParseFloat(alignFrac(fields[1], MaxFsp), 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, 0, seconds*float64(gotime.Second) + microseconds*float64(gotime.Microsecond), nil
}

// extractMinuteMicrosecond extracts minutes and microsecond from a string and its format is `MM:SS.FFFFFF`.
func extractMinuteMicrosecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, ":")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	minutes, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	_, _, _, value, err := extractSecondMicrosecond(fields[1])
	if err != nil {
		return 0, 0, 0, 0, errors.Trace(err)
	}

	return 0, 0, 0, minutes*float64(gotime.Minute) + value, nil
}

// extractMinuteSecond extracts minutes and second from a string and its format is `MM:SS`.
func extractMinuteSecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, ":")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	minutes, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	seconds, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, 0, minutes*float64(gotime.Minute) + seconds*float64(gotime.Second), nil
}

// extractHourMicrosecond extracts hour and microsecond from a string and its format is `HH:MM:SS.FFFFFF`.
func extractHourMicrosecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, ":")
	if len(fields) != 3 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	hours, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	minutes, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	_, _, _, value, err := extractSecondMicrosecond(fields[2])
	if err != nil {
		return 0, 0, 0, 0, errors.Trace(err)
	}

	return 0, 0, 0, hours*float64(gotime.Hour) + minutes*float64(gotime.Minute) + value, nil
}

// extractHourSecond extracts hour and second from a string and its format is `HH:MM:SS`.
func extractHourSecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, ":")
	if len(fields) != 3 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	hours, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	minutes, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	seconds, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, 0, hours*float64(gotime.Hour) + minutes*float64(gotime.Minute) + seconds*float64(gotime.Second), nil
}

// extractHourMinute extracts hour and minute from a string and its format is `HH:MM`.
func extractHourMinute(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, ":")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	hours, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	minutes, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, 0, hours*float64(gotime.Hour) + minutes*float64(gotime.Minute), nil
}

// extractDayMicrosecond extracts day and microsecond from a string and its format is `DD HH:MM:SS.FFFFFF`.
func extractDayMicrosecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, " ")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	days, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	_, _, _, value, err := extractHourMicrosecond(fields[1])
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, days, value, nil
}

// extractDaySecond extracts day and hour from a string and its format is `DD HH:MM:SS`.
func extractDaySecond(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, " ")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	days, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	_, _, _, value, err := extractHourSecond(fields[1])
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, days, value, nil
}

// extractDayMinute extracts day and minute from a string and its format is `DD HH:MM`.
func extractDayMinute(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, " ")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	days, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	_, _, _, value, err := extractHourMinute(fields[1])
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, days, value, nil
}

// extractDayHour extracts day and hour from a string and its format is `DD HH`.
func extractDayHour(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, " ")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	days, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	hours, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return 0, 0, days, hours * float64(gotime.Hour), nil
}

// extractYearMonth extracts year and month from a string and its format is `YYYY-MM`.
func extractYearMonth(format string) (int64, int64, int64, float64, error) {
	fields := strings.Split(format, "-")
	if len(fields) != 2 {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	years, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	months, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, ErrIncorrectDatetimeValue.GenWithStackByArgs(format)
	}

	return years, months, 0, 0, nil
}

// ExtractTimeValue extracts time value from time unit and format.
func ExtractTimeValue(unit string, format string) (int64, int64, int64, float64, error) {
	switch strings.ToUpper(unit) {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "QUARTER", "YEAR":
		return extractSingleTimeValue(unit, format)
	case "SECOND_MICROSECOND":
		return extractSecondMicrosecond(format)
	case "MINUTE_MICROSECOND":
		return extractMinuteMicrosecond(format)
	case "MINUTE_SECOND":
		return extractMinuteSecond(format)
	case "HOUR_MICROSECOND":
		return extractHourMicrosecond(format)
	case "HOUR_SECOND":
		return extractHourSecond(format)
	case "HOUR_MINUTE":
		return extractHourMinute(format)
	case "DAY_MICROSECOND":
		return extractDayMicrosecond(format)
	case "DAY_SECOND":
		return extractDaySecond(format)
	case "DAY_MINUTE":
		return extractDayMinute(format)
	case "DAY_HOUR":
		return extractDayHour(format)
	case "YEAR_MONTH":
		return extractYearMonth(format)
	default:
		return 0, 0, 0, 0, errors.Errorf("invalid singel timeunit - %s", unit)
	}
}

// IsClockUnit returns true when unit is interval unit with hour, minute or second.
func IsClockUnit(unit string) bool {
	switch strings.ToUpper(unit) {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR",
		"SECOND_MICROSECOND", "MINUTE_MICROSECOND", "MINUTE_SECOND",
		"HOUR_MICROSECOND", "HOUR_SECOND", "HOUR_MINUTE",
		"DAY_MICROSECOND", "DAY_SECOND", "DAY_MINUTE", "DAY_HOUR":
		return true
	default:
		return false
	}
}

// IsDateFormat returns true when the specified time format could contain only date.
func IsDateFormat(format string) bool {
	format = strings.TrimSpace(format)
	seps := ParseDateFormat(format)
	length := len(format)
	switch len(seps) {
	case 1:
		if (length == 8) || (length == 6) {
			return true
		}
	case 3:
		return true
	}
	return false
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
