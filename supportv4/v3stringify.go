package supportv4

import (
	"fmt"
	"regexp"
	"strings"

	gapi "google.golang.org/api/analyticsreporting/v4"
)

// V4[AST] -> V3[string]

func escapedJoin(xs []string, sep string, escaped string) string {
	ys := make([]string, len(xs))
	for i, x := range xs {
		ys[i] = strings.Replace(x, sep, escaped, -1)
	}
	return strings.Join(ys, sep)
}

// V3StringifyDynamicSegment :
func V3StringifyDynamicSegment(node *gapi.DynamicSegment) (string, error) {
	statements := make([]string, 0, 2)
	if node.UserSegment != nil {
		inner, err := V3StringifySegmentDefinition(node.UserSegment)
		if err != nil {
			return "", err
		}
		statements = append(statements, "users::"+inner)
	}
	if node.SessionSegment != nil {
		inner, err := V3StringifySegmentDefinition(node.SessionSegment)
		if err != nil {
			return "", err
		}
		statements = append(statements, "sessions::"+inner)
	}
	if len(statements) <= 0 {
		return "", fmt.Errorf("at least either a session or a user segment")
	}
	return strings.Join(statements, ";"), nil
}

// V3StringifySegmentDefinition :
func V3StringifySegmentDefinition(node *gapi.SegmentDefinition) (string, error) {
	if node == nil {
		return "", nil
	}
	outer := make([]string, 0, len(node.SegmentFilters))
	for _, segmentFilter := range node.SegmentFilters {
		inner, err := V3StringifySegmentFilter(segmentFilter)
		if err != nil {
			return "", err
		}
		outer = append(outer, inner)
	}
	return strings.Join(outer, ";"), nil
}

// V3StringifySegmentFilter :
func V3StringifySegmentFilter(node *gapi.SegmentFilter) (string, error) {
	if node == nil {
		return "", nil
	}
	if node.SimpleSegment != nil {
		inner, err := V3StringifySimpleSegment(node.SimpleSegment)
		if err != nil {
			return "", err
		}
		if node.Not {
			inner = "!" + inner
		}
		return "condition::" + inner, nil
	}
	if node.SequenceSegment != nil {
		// from gasegment's implementation: FirstStepShouldMatchFirstHit + Not is "!^"
		inner, err := V3StringifySequenceSegment(node.SequenceSegment)
		if err != nil {
			return "", err
		}
		if node.Not {
			inner = "!" + inner
		}
		return "sequence::" + inner, nil
	}
	return "", fmt.Errorf("at least either a simple or a sequence segment")
}

// V3StringifySequenceSegment :
func V3StringifySequenceSegment(node *gapi.SequenceSegment) (string, error) {
	if node == nil {
		return "", nil
	}
	prefix := ""
	if node.FirstStepShouldMatchFirstHit {
		prefix = "^"
	}
	outer := make([]string, 0, len(node.SegmentSequenceSteps))
	for i, step := range node.SegmentSequenceSteps {
		var bop string
		isFirst := i == 0
		if !isFirst {
			// see also: ./detect.go DetectMatchType
			prevStep := node.SegmentSequenceSteps[i-1]
			switch prevStep.MatchType {
			case MatchTypeUnspecfied:
				bop = ";->>" // Unspecified match type is treated as precedes.
			case MatchTypePrecedes:
				bop = ";->>"
			case MatchTypeImmediatelyPrecedes:
				bop = ";->"
			}
		}
		// step.MatchType
		inner := make([]string, 0, len(step.OrFiltersForSegment))
		for _, orFilter := range step.OrFiltersForSegment {
			expression, err := V3StringifyOrFiltersForSegment(orFilter)
			if err != nil {
				return "", err
			}
			if expression != "" {
				inner = append(inner, expression)
			}
		}
		if len(inner) > 0 {
			if isFirst {
				outer = append(outer, escapedJoin(inner, ";", `\;`))
			} else {
				outer = append(outer, bop, escapedJoin(inner, ";", `\;`))
			}
		}
	}
	return fmt.Sprintf("%s%s", prefix, strings.Join(outer, "")), nil
}

// V3StringifySimpleSegment :
func V3StringifySimpleSegment(node *gapi.SimpleSegment) (string, error) {
	if node == nil {
		return "", nil
	}
	outer := make([]string, 0, len(node.OrFiltersForSegment))
	for _, orFilter := range node.OrFiltersForSegment {
		inner, err := V3StringifyOrFiltersForSegment(orFilter)
		if err != nil {
			return "", err
		}
		if inner != "" {
			outer = append(outer, inner)
		}
	}
	return escapedJoin(outer, ";", `\;`), nil
}

// V3StringifyOrFiltersForSegment :
func V3StringifyOrFiltersForSegment(node *gapi.OrFiltersForSegment) (string, error) {
	outer := make([]string, 0, len(node.SegmentFilterClauses))
	for _, segmentFilter := range node.SegmentFilterClauses {
		inner, err := V3StringifySegmentFilterClause(segmentFilter)
		if err != nil {
			return "", err
		}
		if inner != "" {
			outer = append(outer, inner)
		}
	}
	return escapedJoin(outer, ",", `\,`), nil
}

// V3StringifySegmentFilterClause :
func V3StringifySegmentFilterClause(node *gapi.SegmentFilterClause) (string, error) {
	if node == nil {
		return "", nil
	}
	if node.DimensionFilter != nil {
		return V3StringifySegmentDimensionFilter(node.DimensionFilter, node.Not)
	}
	if node.MetricFilter != nil {
		return V3StringifySegmentMetricFilter(node.MetricFilter, node.Not)
	}
	return "", fmt.Errorf("must be wither a metric or a dimension filter")
}

// V3StringifySegmentDimensionFilter :
func V3StringifySegmentDimensionFilter(node *gapi.SegmentDimensionFilter, not bool) (string, error) {
	if node == nil {
		return "", nil
	}

	// "OPERATOR_UNSPECIFIED" - If the match type is unspecified, it is treated as a REGEXP.
	op := node.Operator
	if op == "OPERATOR_UNSPECIFIED" || op == "" {
		op = OperatorRegexp
	}
	if len(node.Expressions) == 0 && op != OperatorNumericBetween {
		return "", fmt.Errorf("invalid expression. at least length >= 1")
	}
	// see also: ./detect.go DetectOperatorOnDimension

	// TODO: node.CaseSensitive
	switch op {
	case OperatorRegexp:
		if not {
			return fmt.Sprintf("%s!~%s", node.DimensionName, node.Expressions[0]), nil
		}
		return fmt.Sprintf("%s=~%s", node.DimensionName, node.Expressions[0]), nil
	case OperatorBeginsWith:
		if not {
			return fmt.Sprintf("%s!~%s", node.DimensionName, "^"+regexp.QuoteMeta(node.Expressions[0])), nil
		}
		return fmt.Sprintf("%s=~%s", node.DimensionName, "^"+regexp.QuoteMeta(node.Expressions[0])), nil
	case OperatorEndsWith:
		if not {
			return fmt.Sprintf("%s!~%s", node.DimensionName, regexp.QuoteMeta(node.Expressions[0])+"$"), nil
		}
		return fmt.Sprintf("%s=~%s", node.DimensionName, regexp.QuoteMeta(node.Expressions[0])+"$"), nil
	case OperatorPartial:
		if not {
			return fmt.Sprintf("%s!@%s", node.DimensionName, node.Expressions[0]), nil
		}
		return fmt.Sprintf("%s=@%s", node.DimensionName, node.Expressions[0]), nil
	case OperatorExact, OperatorNumericEquals:
		if not {
			return fmt.Sprintf("%s!=%s", node.DimensionName, node.Expressions[0]), nil
		}
		return fmt.Sprintf("%s==%s", node.DimensionName, node.Expressions[0]), nil
	case OperatorInList:
		// TODO: limitation of number of expressions <= 10
		var op string
		if not {
			op = "![]"
		} else {
			op = "[]"
		}
		return fmt.Sprintf("%s%s%s", node.DimensionName, op, escapedJoin(node.Expressions, "|", `\|`)), nil
	case OperatorNumericLessThan:
		if not {
			return fmt.Sprintf("%s>=%s", node.DimensionName, node.Expressions[0]), nil
		}
		return fmt.Sprintf("%s<%s", node.DimensionName, node.Expressions[0]), nil
	case OperatorNumericGreaterThan:
		if not {
			return fmt.Sprintf("%s<=%s", node.DimensionName, node.Expressions[0]), nil
		}
		return fmt.Sprintf("%s>%s", node.DimensionName, node.Expressions[0]), nil
	case OperatorNumericBetween:
		var op string
		if not {
			op = "!<>"
		} else {
			op = "<>"
		}
		minValue := strings.Replace(node.MinComparisonValue, "|", `\|`, -1)
		maxValue := strings.Replace(node.MaxComparisonValue, "|", `\|`, -1)
		return fmt.Sprintf("%s%s%s_%s", node.DimensionName, op, minValue, maxValue), nil
	default:
		return "", fmt.Errorf("unsupported dimension operator: %s", op)
	}
}

// V3StringifySegmentMetricFilter :
func V3StringifySegmentMetricFilter(node *gapi.SegmentMetricFilter, not bool) (string, error) {
	if node == nil {
		return "", nil
	}

	// "OPERATOR_UNSPECIFIED" - If the match type is unspecified, it is treated as a REGEXP.
	op := node.Operator
	if op == "OPERATOR_UNSPECIFIED" {
		op = OperatorEqual
	}
	// see also: ./detect.go DetectOperatorOnMetric

	scopePrefix := ""
	switch node.Scope {
	case "PRODUCT":
		scopePrefix = "perProduct::" // Product scope.
	case "HIT":
		scopePrefix = "perHit::" // Hit scope.
	case "USER":
		scopePrefix = "perUser::" // User scope.
	case "SESSION":
		scopePrefix = "perSession::" // Session scope.
	}
	switch op {
	case OperatorEqual:
		if not {
			return fmt.Sprintf("%s%s!=%s", scopePrefix, node.MetricName, node.ComparisonValue), nil
		}
		return fmt.Sprintf("%s%s==%s", scopePrefix, node.MetricName, node.ComparisonValue), nil
	case OperatorLessThan:
		if not {
			return fmt.Sprintf("%s%s>=%s", scopePrefix, node.MetricName, node.ComparisonValue), nil
		}
		return fmt.Sprintf("%s%s<%s", scopePrefix, node.MetricName, node.ComparisonValue), nil
	case OperatorGreaterThan:
		if not {
			return fmt.Sprintf("%s%s<=%s", scopePrefix, node.MetricName, node.ComparisonValue), nil
		}
		return fmt.Sprintf("%s%s>%s", scopePrefix, node.MetricName, node.ComparisonValue), nil
	case OperatorBetween:
		if not {
			return fmt.Sprintf("%s%s!<>%s_%s", scopePrefix, node.MetricName, node.ComparisonValue, node.MaxComparisonValue), nil
		}
		return fmt.Sprintf("%s%s<>%s_%s", scopePrefix, node.MetricName, node.ComparisonValue, node.MaxComparisonValue), nil
	default:
		return "", fmt.Errorf("unsupported metric operator: %s", op)
	}
}
