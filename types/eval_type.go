package types

// EvalType indicates the specified types that arguments and result of a built-in function should be.
type EvalType byte

const (
	// ETInt represents type INT in evaluation.
	ETInt EvalType = iota
	// ETReal represents type REAL in evaluation.
	ETReal
	// ETDecimal represents type DECIMAL in evaluation.
	ETDecimal
	// ETString represents type STRING in evaluation.
	ETString
	// ETDatetime represents type DATETIME in evaluation.
	ETDatetime
	// ETTimestamp represents type TIMESTAMP in evaluation.
	ETTimestamp
	// ETDuration represents type DURATION in evaluation.
	ETDuration
	// ETJson represents type JSON in evaluation.
	ETJson
)

// IsStringKind returns true for ETString, ETDatetime, ETTimestamp, ETDuration, ETJson EvalTypes.
func (et EvalType) IsStringKind() bool {
	return et == ETString || et == ETDatetime ||
		et == ETTimestamp || et == ETDuration || et == ETJson
}
