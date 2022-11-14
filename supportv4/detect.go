package supportv4

import (
	"fmt"

	"github.com/wacul/gasegment"
)

// MatchType
const (
	MatchTypeUnspecfied          = "" // "UNSPECIFIED_MATCH_TYPE" // omitempty
	MatchTypePrecedes            = "PRECEDES"
	MatchTypeImmediatelyPrecedes = "IMMEDIATELY_PRECEDES"
)

// DetectMatchType : SequenceStepType -> MatchType string
func DetectMatchType(stepType gasegment.SequenceStepType) (string, error) {
	// MatchType: Specifies if the step immediately precedes or can be any
	// time before the
	// next step.
	//
	// Possible values:
	//   "UNSPECIFIED_MATCH_TYPE" - Unspecified match type is treated as
	// precedes.
	//   "PRECEDES" - Operator indicates that the previous step precedes the
	// next step.
	//   "IMMEDIATELY_PRECEDES" - Operator indicates that the previous step
	// immediately precedes the next
	// step.
	switch stepType {
	case gasegment.FirstStep:
		return MatchTypeUnspecfied, nil
	case gasegment.Precedes:
		return MatchTypePrecedes, nil
	case gasegment.ImmediatelyPrecedes:
		return MatchTypeImmediatelyPrecedes, nil
	default:
		return "", fmt.Errorf("unspecified match type: %v", stepType)
	}
}

// Scope
const (
	ScopeUnspecified = "" // UNSPECIFIED_SCOPE // omitempty
	ScopeProduct     = "PRODUCT"
	ScopeHit         = "HIT"
	ScopeSession     = "SESSION"
	ScopeUser        = "USER"
)

// DetectScope : MetricScope -> scope string
func DetectScope(metricScope gasegment.MetricScope) (string, error) {
	// FIXME: it is not found, in gaesegment, that the expression about PerProduct
	switch metricScope {
	case gasegment.Default:
		return ScopeUnspecified, nil
	case gasegment.PerHit:
		return ScopeHit, nil
	case gasegment.PerSession:
		return ScopeSession, nil
	case gasegment.PerUser:
		return ScopeUser, nil
	default:
		return "", fmt.Errorf("unspecified scope: %v", metricScope)
	}
}

// Operator : string representation for analyticsreporting.operator
const (
	OperatorUnspecified        = "" // "UNSPECIFIED_OPERATOR" // omitempty
	OperatorUnspecified2       = "" // "OPERATOR_UNSPECIFIED" // omitempty
	OperatorLessThan           = "LESS_THAN"
	OperatorGreaterThan        = "GREATER_THAN"
	OperatorEqual              = "EQUAL"
	OperatorBetween            = "BETWEEN"
	OperatorRegexp             = "REGEXP"
	OperatorBeginsWith         = "BEGINS_WITH"
	OperatorEndsWith           = "ENDS_WITH"
	OperatorPartial            = "PARTIAL"
	OperatorExact              = "EXACT"
	OperatorInList             = "IN_LIST"
	OperatorNumericLessThan    = "NUMERIC_LESS_THAN"
	OperatorNumericGreaterThan = "NUMERIC_GREATER_THAN"
	OperatorNumericBetween     = "NUMERIC_BETWEEN"
	OperatorNumericEquals      = "NUMERIC_EQUALS"
)

// DetectOperatorOnMetric :
func DetectOperatorOnMetric(op gasegment.Operator) (opStr string, not bool, err error) {
	// Operator: Specifies is the operation to perform to compare the
	// metric. The default
	// is `EQUAL`.
	//
	// Possible values:
	//   "UNSPECIFIED_OPERATOR" - Unspecified operator is treated as
	// `LESS_THAN` operator.
	//   "LESS_THAN" - Checks if the metric value is less than comparison
	// value.
	//   "GREATER_THAN" - Checks if the metric value is greater than
	// comparison value.
	//   "EQUAL" - Equals operator.
	//   "BETWEEN" - For between operator, both the minimum and maximum are
	// exclusive.
	// We will use `LT` and `GT` for comparison.
	switch op {
	case gasegment.Equal:
		return OperatorEqual, false, nil
	case gasegment.NotEqual:
		return OperatorEqual, true, nil
	case gasegment.LessThan:
		return OperatorLessThan, false, nil
	case gasegment.LessThanEqual:
		// x <= y == !(x > y)
		return OperatorGreaterThan, true, nil
	case gasegment.GreaterThan:
		return OperatorGreaterThan, false, nil
	case gasegment.GreaterThanEqual:
		// x >= y == !(x < y)
		return OperatorLessThan, true, nil
	case gasegment.Between:
		return OperatorBetween, false, nil
	case gasegment.NotBetween:
		return OperatorBetween, true, nil
	// case gasegment.InList:
	// case gasegment.ContainsSubstring:
	// case gasegment.NotContainsSubstring:
	// case gasegment.Regexp:
	// case gasegment.NotRegexp:
	default:
		return "", false, fmt.Errorf("unspecified operator on metric %v", op)
	}
}

// DetectOperatorOnDimension :
func DetectOperatorOnDimension(op gasegment.Operator) (opStr string, not bool, err error) {
	// Operator: The operator to use to match the dimension with the
	// expressions.
	//
	// Possible values:
	//   "OPERATOR_UNSPECIFIED" - If the match type is unspecified, it is
	// treated as a REGEXP.
	//   "REGEXP" - The match expression is treated as a regular expression.
	// All other match
	// types are not treated as regular expressions.
	//   "BEGINS_WITH" - Matches the values which begin with the match
	// expression provided.
	//   "ENDS_WITH" - Matches the values which end with the match
	// expression provided.
	//   "PARTIAL" - Substring match.
	//   "EXACT" - The value should match the match expression entirely.
	//   "IN_LIST" - This option is used to specify a dimension filter whose
	// expression can
	// take any value from a selected list of values. This helps
	// avoiding
	// evaluating multiple exact match dimension filters which are OR'ed
	// for
	// every single response row. For example:
	//
	//     expressions: ["A", "B", "C"]
	//
	// Any response row whose dimension has it is value as A, B or C,
	// matches
	// this DimensionFilter.
	//   "NUMERIC_LESS_THAN" - Integer comparison filters.
	// case sensitivity is ignored for these and the expression
	// is assumed to be a string representing an integer.
	// Failure conditions:
	//
	// - if expression is not a valid int64, the client should expect
	//   an error.
	// - input dimensions that are not valid int64 values will never match
	// the
	//   filter.
	//
	// Checks if the dimension is numerically less than the match
	// expression.
	//   "NUMERIC_GREATER_THAN" - Checks if the dimension is numerically
	// greater than the match
	// expression.
	//   "NUMERIC_BETWEEN" - Checks if the dimension is numerically between
	// the minimum and maximum
	// of the match expression, boundaries excluded.
	switch op {
	case gasegment.Equal:
		return OperatorExact, false, nil
	case gasegment.NotEqual:
		return OperatorExact, true, nil
	case gasegment.LessThan:
		return OperatorNumericLessThan, false, nil
	case gasegment.LessThanEqual:
		// x <= y == !(x > y)
		return OperatorNumericGreaterThan, true, nil
	case gasegment.GreaterThan:
		return OperatorNumericGreaterThan, false, nil
	case gasegment.GreaterThanEqual:
		// x >= y == !(x < y)
		return OperatorNumericLessThan, true, nil
	case gasegment.Between:
		return OperatorNumericBetween, false, nil
	case gasegment.NotBetween:
		return OperatorNumericBetween, true, nil
	case gasegment.InList:
		return OperatorInList, false, nil
	case gasegment.NotInList:
		return OperatorInList, true, nil
	case gasegment.ContainsSubstring:
		return OperatorPartial, false, nil
	case gasegment.NotContainsSubstring:
		return OperatorPartial, true, nil
	case gasegment.Regexp:
		return OperatorRegexp, false, nil
	case gasegment.NotRegexp:
		return OperatorRegexp, true, nil
	default:
		return "", false, fmt.Errorf("unspecified operator on dimension %v", op)
	}
}

// FilterType : gaesegment.Expression doesn't have a information about theirself is Metric or Dimention.. so needs detecting it..
type FilterType string

// FilterType :
const (
	FilterTypeDimension   = FilterType("dimension")
	FilterTypeMetric      = FilterType("metric")
	FilterTypeUnspecified = FilterType("unspecified")
)

var filterTypeMap map[gasegment.DimensionOrMetric]FilterType

func init() {
	filterTypeMap = map[gasegment.DimensionOrMetric]FilterType{}
}

// detectFilterType : detects filter type (primitive)
func detectFilterType(dm gasegment.DimensionOrMetric) (FilterType, error) {
	attr, err := gasegment.GetDimensionOrMetricAttributes(dm.String())
	if err != nil {
		return FilterTypeUnspecified, err
	}
	switch attr.Type {
	case "METRIC":
		return FilterTypeMetric, nil
	case "DIMENSION":
		return FilterTypeDimension, nil
	default:
		return FilterTypeUnspecified, nil
	}
}

// DetectFilterType : detects filter type
func DetectFilterType(dm gasegment.DimensionOrMetric) (FilterType, error) {
	ftype, ok := filterTypeMap[dm]
	if ok {
		return ftype, nil
	}
	ftype, err := detectFilterType(dm)
	filterTypeMap[dm] = ftype
	if err != nil {
		return ftype, err
	}
	if ftype == FilterTypeUnspecified {
		return ftype, fmt.Errorf("unspecified filter type %v", dm)
	}
	return ftype, nil
}
