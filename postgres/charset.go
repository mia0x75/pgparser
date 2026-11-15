package postgres

// The parser only needs the defaults; the full collation tables were
// unreachable outside their own tests.
const (
	DefaultCharset       = "utf8mb4"
	DefaultCollationName = "utf8mb4_bin"
)
