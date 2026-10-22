package model

// Flags are used by tipb.SelectRequest.Flags to handle execution mode, like how to handle truncate error.
const (
	// FlagIgnoreTruncate indicates if truncate error should be ignored.
	// Read-only statements should ignore truncate error, write statements should not ignore truncate error.
	FlagIgnoreTruncate uint64 = 1
	// FlagTruncateAsWarning indicates if truncate error should be returned as warning.
	// This flag only matters if FlagIgnoreTruncate is not set, in strict sql mode, truncate error should
	// be returned as error, in non-strict sql mode, truncate error should be saved as warning.
	FlagTruncateAsWarning = 1 << 1
	// FlagPadCharToFullLength indicates if sql_mode 'PAD_CHAR_TO_FULL_LENGTH' is set.
	FlagPadCharToFullLength = 1 << 2
	// FlagInInsertStmt indicates if this is a INSERT statement.
	FlagInInsertStmt = 1 << 3
	// FlagInUpdateOrDeleteStmt indicates if this is a UPDATE statement or a DELETE statement.
	FlagInUpdateOrDeleteStmt = 1 << 4
	// FlagInSelectStmt indicates if this is a SELECT statement.
	FlagInSelectStmt = 1 << 5
	// FlagOverflowAsWarning indicates if overflow error should be returned as warning.
	// In strict sql mode, overflow error should be returned as error,
	// in non-strict sql mode, overflow error should be saved as warning.
	FlagOverflowAsWarning = 1 << 6
	// FlagIgnoreZeroInDate indicates if ZeroInDate error should be ignored.
	// Read-only statements should ignore ZeroInDate error.
	// Write statements should not ignore ZeroInDate error in strict sql mode.
	FlagIgnoreZeroInDate = 1 << 7
	// FlagDividedByZeroAsWarning indicates if DividedByZero should be returned as warning.
	FlagDividedByZeroAsWarning = 1 << 8
)
