package parser

import (
	"strconv"
	"strings"

	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types"
)

// pgTypeTp maps names to internal FieldType IDs rather than PostgreSQL OIDs.
func pgTypeTp(name string) byte {
	switch strings.ToLower(strings.Join(strings.Fields(name), " ")) {
	case "smallint", "int2", "smallserial", "serial2":
		return postgres.TypeShort
	case "int", "integer", "int4", "serial", "serial4":
		return postgres.TypeLong
	case "bigint", "int8", "bigserial", "serial8":
		return postgres.TypeLonglong
	case "real", "float4":
		return postgres.TypeFloat
	case "double", "double precision", "float8", "float":
		return postgres.TypeDouble
	case "numeric", "decimal", "money":
		return postgres.TypeNewDecimal
	case "boolean", "bool":
		return postgres.TypeTiny
	case "character", "char", "bpchar", "nchar":
		return postgres.TypeString
	case "character varying", "varchar", "nvarchar", "varying character":
		return postgres.TypeVarchar
	case "text", "bytea", "clob", "xml", "cidr", "inet", "macaddr", "macaddr8", "tsvector", "tsquery", "uuid":
		return postgres.TypeBlob
	case "date":
		return postgres.TypeDate
	case "time", "timetz", "time without time zone", "time with time zone", "interval":
		return postgres.TypeDuration
	case "timestamp", "timestamptz", "timestamp without time zone", "timestamp with time zone":
		return postgres.TypeTimestamp
	case "json", "jsonb":
		return postgres.TypeJSON
	case "bit", "bit varying", "varbit":
		return postgres.TypeBit
	case "year":
		return postgres.TypeYear
	default:
		return postgres.TypeVarString
	}
}

func newDataType(name string, flen, decimal int) *types.FieldType {
	tp := types.NewFieldType(pgTypeTp(name))
	tp.PGName = []string{name}
	if flen >= 0 {
		tp.PGModifiers = append(tp.PGModifiers, strconv.Itoa(flen))
	}
	if decimal >= 0 {
		tp.PGModifiers = append(tp.PGModifiers, strconv.Itoa(decimal))
	}
	if flen >= 0 {
		tp.Flen = flen
	}
	if decimal >= 0 {
		tp.Decimal = decimal
	}
	return tp
}

// explainFormat returns the format name from EXPLAIN options.
func explainFormat(options []string) string {
	for _, opt := range options {
		fields := strings.Fields(opt)
		if len(fields) == 2 && strings.EqualFold(fields[0], "format") {
			return fields[1]
		}
	}
	return "text"
}
