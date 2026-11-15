package postgres

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"

	"github.com/mia0x75/pgparser/errors"
)

// precisionToInt parses a precision string, rejecting values that overflow int.
func precisionToInt(precision string) (int, error) {
	n, err := strconv.Atoi(precision)
	if err != nil {
		return 0, errors.Trace(err)
	}
	if n < 0 {
		return 0, errors.Errorf("precision %d out of range", n)
	}
	return n, nil
}

func formatENUS(number string, precision string) (string, error) {
	var buffer bytes.Buffer
	if unicode.IsDigit(rune(precision[0])) {
		for i, v := range precision {
			if unicode.IsDigit(v) {
				continue
			}
			precision = precision[:i]
			break
		}
	} else {
		precision = "0"
	}
	if number[0] == '-' && number[1] == '.' {
		number = strings.Replace(number, "-", "-0", 1)
	} else if number[0] == '.' {
		number = strings.Replace(number, ".", "0.", 1)
	}

	if (number[:1] == "-" && !unicode.IsDigit(rune(number[1]))) ||
		(!unicode.IsDigit(rune(number[0])) && number[:1] != "-") {
		buffer.Write([]byte{'0'})
		n, err := precisionToInt(precision)
		if err != nil {
			return "", err
		}
		if n > 0 {
			buffer.Write([]byte{'.'})
			buffer.WriteString(strings.Repeat("0", n))
		}
		return buffer.String(), nil
	} else if number[:1] == "-" {
		buffer.Write([]byte{'-'})
		number = number[1:]
	}

	for i, v := range number {
		switch {
		case unicode.IsDigit(v), i == 1 && number[1] == '.', v == '.' && number[1] != '.':
			continue
		}
		number = number[:i]
		break
	}

	comma := []byte{','}
	parts := strings.Split(number, ".")
	pos := 0
	if len(parts[0])%3 != 0 {
		pos += len(parts[0]) % 3
		buffer.WriteString(parts[0][:pos])
		buffer.Write(comma)
	}
	for ; pos < len(parts[0]); pos += 3 {
		buffer.WriteString(parts[0][pos : pos+3])
		buffer.Write(comma)
	}
	buffer.Truncate(buffer.Len() - 1)

	n, err := precisionToInt(precision)
	if err != nil {
		return "", err
	}
	if n > 0 {
		buffer.Write([]byte{'.'})
		if len(parts) == 2 {
			if len(parts[1]) >= n {
				buffer.WriteString(parts[1][:n])
			} else {
				buffer.WriteString(parts[1])
				buffer.WriteString(strings.Repeat("0", n-len(parts[1])))
			}
		} else {
			buffer.WriteString(strings.Repeat("0", n))
		}
	}

	return buffer.String(), nil
}

func formatZHCN(number string, precision string) (string, error) {
	return "", errors.New("not implemented")
}

func formatNotSupport(number string, precision string) (string, error) {
	return "", errors.New("not support for the specific locale")
}
