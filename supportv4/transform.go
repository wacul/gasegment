package supportv4

import (
	"fmt"
	"strings"

	"github.com/wcl48/gasegment"
	gapi "google.golang.org/api/analyticsreporting/v4"
)

// TransformSegments : transform Seguments to DynamicSegment
func TransformSegments(segments *gasegment.Segments) (*gapi.DynamicSegment, error) {
	name := "-"
	segmentSet := []gasegment.Segment(*segments)
	sessionSegmentFilters := make([]*gapi.SegmentFilter, 0, len(segmentSet))
	userSegmentFilters := make([]*gapi.SegmentFilter, 0, len(segmentSet))
	var sessionSegment *gapi.SegmentDefinition
	var userSegment *gapi.SegmentDefinition

	for _, segment := range segmentSet {
		switch segment.Scope {
		case gasegment.UserScope:
			segmentFilter, err := NewSegmentFilter(&segment)
			if err != nil {
				return nil, err
			}
			userSegmentFilters = append(userSegmentFilters, segmentFilter)
		case gasegment.SessionScope:
			segmentFilter, err := NewSegmentFilter(&segment)
			if err != nil {
				return nil, err
			}
			sessionSegmentFilters = append(sessionSegmentFilters, segmentFilter)
		default:
			return nil, fmt.Errorf("cannot guess segment scope=%v", segment.Scope)
		}
	}

	if len(userSegmentFilters) > 0 {
		userSegment = &gapi.SegmentDefinition{
			SegmentFilters: userSegmentFilters,
		}
	}
	if len(sessionSegmentFilters) > 0 {
		sessionSegment = &gapi.SegmentDefinition{
			SegmentFilters: sessionSegmentFilters,
		}
	}
	return &gapi.DynamicSegment{
		Name:           name,
		SessionSegment: sessionSegment,
		UserSegment:    userSegment,
	}, nil
}

// TransformSegment : transform Segument to DynamicSegment
func TransformSegment(segment *gasegment.Segment) (*gapi.DynamicSegment, error) {
	name := "-"
	switch segment.Scope {
	case gasegment.UserScope:
		segmentFilter, err := NewSegmentFilter(segment)
		if err != nil {
			return nil, err
		}
		return &gapi.DynamicSegment{
			Name: name,
			UserSegment: &gapi.SegmentDefinition{
				SegmentFilters: []*gapi.SegmentFilter{segmentFilter},
			},
		}, nil
	case gasegment.SessionScope:
		segmentFilter, err := NewSegmentFilter(segment)
		if err != nil {
			return nil, err
		}
		return &gapi.DynamicSegment{
			Name: name,
			SessionSegment: &gapi.SegmentDefinition{
				SegmentFilters: []*gapi.SegmentFilter{segmentFilter},
			},
		}, nil
	default:
		return nil, fmt.Errorf("cannot guess segment scope=%v", segment.Scope)
	}
}

// NewSegmentFilter : creates segmentFilter from segment
func NewSegmentFilter(segment *gasegment.Segment) (*gapi.SegmentFilter, error) {
	switch segment.Type {
	case gasegment.ConditionSegment:
		return TransformCondition(&segment.Condition)
	case gasegment.SequenceSegment:
		return TransformSequence(&segment.Sequence)
	default:
		return nil, fmt.Errorf("cannot guess segment type=%v", segment.Type)
	}
}

// TransformSequence : transform Sequence to SegmentFilter
func TransformSequence(sequence *gasegment.Sequence) (*gapi.SegmentFilter, error) {
	steps, err := TransformSequenceSteps(&sequence.SequenceSteps)
	if err != nil {
		return nil, err
	}
	return &gapi.SegmentFilter{
		Not: sequence.Not,
		SequenceSegment: &gapi.SequenceSegment{
			FirstStepShouldMatchFirstHit: sequence.FirstHitMatchesFirstStep,
			SegmentSequenceSteps:         steps,
		},
	}, nil
}

// TransformSequenceSteps : transform SequenceSteps to []*SegmentSequenceStep
func TransformSequenceSteps(src *gasegment.SequenceSteps) ([]*gapi.SegmentSequenceStep, error) {
	steps := []gasegment.SequenceStep(*src)
	dst := make([]*gapi.SegmentSequenceStep, len(steps))
	for i, srcStep := range steps {
		dstStep, err := TransformSequqnceStep(&srcStep)
		if err != nil {
			return nil, err
		}
		dst[i] = dstStep
	}
	return dst, nil
}

// TransformSequqnceStep : transform SequenceStep to SegmentSequenceStep
func TransformSequqnceStep(step *gasegment.SequenceStep) (*gapi.SegmentSequenceStep, error) {
	matchType, err := DetectMatchType(step.Type)
	orSegments, err := TransformAndExpression(&step.AndExpression)
	if err != nil {
		return nil, err
	}
	return &gapi.SegmentSequenceStep{
		MatchType:           matchType,
		OrFiltersForSegment: orSegments,
	}, nil
}

// TransformCondition : transform Condition to SegmentFilter
func TransformCondition(condition *gasegment.Condition) (*gapi.SegmentFilter, error) {
	orSegments, err := TransformAndExpression(&condition.AndExpression)
	if err != nil {
		return nil, err
	}
	return &gapi.SegmentFilter{
		SimpleSegment: &gapi.SimpleSegment{
			OrFiltersForSegment: orSegments,
		},
		Not: condition.Exclude,
	}, nil
}

// TransformAndExpression : transform AndExpression to []*OrFiltersForSegment
func TransformAndExpression(andExpression *gasegment.AndExpression) ([]*gapi.OrFiltersForSegment, error) {
	orExprs := []gasegment.OrExpression(*andExpression)
	orSegments := make([]*gapi.OrFiltersForSegment, len(orExprs))
	for i, orExpr := range orExprs {
		orSegment, err := TransformOrExpression(&orExpr)
		if err != nil {
			return nil, err
		}
		orSegments[i] = orSegment
	}
	return orSegments, nil
}

// TransformOrExpression : transform OrExpression to OrFiltersForSegment
func TransformOrExpression(orExpression *gasegment.OrExpression) (*gapi.OrFiltersForSegment, error) {
	exprs := []gasegment.Expression(*orExpression)
	clauses := make([]*gapi.SegmentFilterClause, len(exprs))
	for i, expr := range exprs {
		clause, err := TransformExpression(&expr)
		if err != nil {
			return nil, err
		}
		clauses[i] = clause
	}
	return &gapi.OrFiltersForSegment{SegmentFilterClauses: clauses}, nil
}

// TransformExpression : transform expression to filter clause
func TransformExpression(expr *gasegment.Expression) (*gapi.SegmentFilterClause, error) {
	ftype, err := DetectFilterType(expr.Target)
	if err != nil {
		return nil, err
	}
	switch ftype {
	case FilterTypeDimension:
		return NewDimensionFilterClause(expr)
	case FilterTypeMetric:
		return NewMetricFilterClause(expr)
	default:
		return nil, fmt.Errorf("cannot guess expression=%v", ftype)
	}
}

// NewDimensionFilterClause : creates filter clause for dimension filter
func NewDimensionFilterClause(expr *gasegment.Expression) (*gapi.SegmentFilterClause, error) {
	op, not, err := DetectOperatorOnDimension(expr.Operator)
	if err != nil {
		return nil, err
	}
	if expr.Operator == gasegment.Between {
		// between operator "<>{minvalue}_{maxvalue}" (see: https://developers.google.com/analytics/devguides/reporting/core/v3/segments?hl=ja)
		vs := strings.SplitN(expr.Value, "_", 2)
		return &gapi.SegmentFilterClause{
			Not: not,
			DimensionFilter: &gapi.SegmentDimensionFilter{
				// CaseSensitive false, // bool `json:"caseSensitive,omitempty"`
				DimensionName:      expr.Target.String(),
				Operator:           op,
				MinComparisonValue: vs[0],
				MaxComparisonValue: vs[1],
			},
		}, nil
	}
	return &gapi.SegmentFilterClause{
		Not: not,
		DimensionFilter: &gapi.SegmentDimensionFilter{
			// CaseSensitive false, // bool `json:"caseSensitive,omitempty"`
			DimensionName: expr.Target.String(),
			Operator:      op,
			Expressions:   []string{expr.Value},
		},
	}, nil
}

// NewMetricFilterClause : creates filter clause for metric filter
func NewMetricFilterClause(expr *gasegment.Expression) (*gapi.SegmentFilterClause, error) {
	op, not, err := DetectOperatorOnMetric(expr.Operator)
	if err != nil {
		return nil, err
	}
	if expr.Operator == gasegment.Between {
		// between operator "<>{minvalue}_{maxvalue}" (see: https://developers.google.com/analytics/devguides/reporting/core/v3/segments?hl=ja)
		vs := strings.SplitN(expr.Value, "_", 2)
		return &gapi.SegmentFilterClause{
			Not: not,
			MetricFilter: &gapi.SegmentMetricFilter{
				// CaseSensitive false, // bool `json:"caseSensitive,omitempty"`
				MetricName:         expr.Target.String(),
				Operator:           op,
				ComparisonValue:    vs[0],
				MaxComparisonValue: vs[1],
			},
		}, nil
	}
	return &gapi.SegmentFilterClause{
		Not: not,
		MetricFilter: &gapi.SegmentMetricFilter{
			// CaseSensitive false, // bool `json:"caseSensitive,omitempty"`
			MetricName:      expr.Target.String(),
			Operator:        op,
			ComparisonValue: expr.Value,
		},
	}, nil
}
