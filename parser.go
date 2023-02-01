package gasegment

import (
	"fmt"
	"regexp"

	"strings"
)

var SegmentConditionRe,
	SequenceSeparatorRe,
	AndSeparatorRe,
	OrSeparatorRe,
	OpSeparatorRe *regexp.Regexp

func init() {
	SegmentConditionRe = regexp.MustCompile(`(?:\\.|[^\\])(;)(?:users::|sessions::)?(?:condition::|sequence::)`)
	SequenceSeparatorRe = regexp.MustCompile(`(?:\\.|[^\\])(;->>|;->)`)
	AndSeparatorRe = regexp.MustCompile(`(?:\\.|[^\\])(;)`)
	OrSeparatorRe = regexp.MustCompile(`(?:\\.|[^\\])(,)`)

	// ORDER IS IMPORTANT.
	// e.g. using invalid ordering - such as `LessThan, LesthanEqual`, then, regexp is />|>=/, so, ">=" is matching at `>`(LessThan). NG
	ops := []Operator{
		Equal,
		NotEqual,
		LessThanEqual,
		GreaterThanEqual,
		NotBetween,
		Between,
		GreaterThan,
		LessThan,
		NotInList,
		InList,
		ContainsSubstring,
		NotContainsSubstring,
		Regexp,
		NotRegexp,
	}
	opbuf := make([]string, len(ops))
	for i, op := range ops {
		opbuf[i] = regexp.QuoteMeta(op.String())
	}
	OpSeparatorRe = regexp.MustCompile(strings.Join(opbuf, "|"))
}

func MustParse(definition string) Segments {
	s, err := Parse(definition)
	if err != nil {
		panic(err)
	}
	return s
}

func Parse(definition string) (Segments, error) {
	parts := splitByFirstRegexpGroup(definition, SegmentConditionRe)
	ret := []Segment{}
	var lastScope SegmentScope
	for i := 0; i < len(parts); i += 2 {
		sd := parts[i]
		s, err := parseSegment(sd)
		if err != nil {
			return nil, err
		}
		if s.Scope.String() == "" {
			if lastScope.String() == "" {
				return Segments{}, fmt.Errorf("no segment scope (user:: or session::)")
			}
			s.Scope = lastScope
		}
		ret = append(ret, s)
		lastScope = s.Scope
	}
	return Segments(ret), nil
}

func parseSegment(definition string) (Segment, error) {
	s := definition
	sg := Segment{}

	// parse scope
	if strings.HasPrefix(s, "sessions::") {
		sg.Scope = SessionScope
		s = s[len("sessions::"):]
	}
	if strings.HasPrefix(s, "users::") {
		sg.Scope = UserScope
		s = s[len("users::"):]
	}

	// condition
	if strings.HasPrefix(s, "condition::") {
		sg.Type = ConditionSegment
		s = s[len("condition::"):]

		c, err := parseCondition(s)
		if err != nil {
			return Segment{}, err
		}
		sg.Condition = c
	} else if strings.HasPrefix(s, "sequence::") {
		sg.Type = SequenceSegment
		s = s[len("sequence::"):]
		sq, err := parseSequence(s)
		if err != nil {
			return Segment{}, err
		}
		sg.Sequence = sq
	} else {
		return sg, fmt.Errorf(fmt.Sprintf("unknown segment condition %s", s))
	}

	return sg, nil
}

func parseSequence(definition string) (Sequence, error) {
	s := definition
	seq := Sequence{}
	// not?
	if strings.HasPrefix(s, "!") {
		seq.Not = true
		s = s[len("!"):]
	}

	// FirstHitMatchesFirstStep?
	if strings.HasPrefix(s, "^") {
		seq.FirstHitMatchesFirstStep = true
		s = s[len("^"):]
	}

	// Sequence
	steps := []SequenceStep{}
	sParts := splitByFirstRegexpGroup(s, SequenceSeparatorRe)

	first, err := parseAndExpression(sParts[0])
	if err != nil {
		return Sequence{}, err
	}
	steps = append(steps, SequenceStep{
		Type:          FirstStep,
		AndExpression: first,
	})
	for i := 1; i < len(sParts); i += 2 {
		step := SequenceStep{}
		typeStr := sParts[i]
		ae, err := parseAndExpression(sParts[i+1])
		if err != nil {
			return Sequence{}, err
		}
		step.Type = SequenceStepType(typeStr)
		step.AndExpression = ae
		steps = append(steps, step)
	}
	seq.SequenceSteps = SequenceSteps(steps)
	return seq, nil
}

func parseCondition(definition string) (Condition, error) {
	s := definition
	c := Condition{}
	if strings.HasPrefix(s, "!") {
		c.Exclude = true
		s = s[len("!"):]
	}

	ae, err := parseAndExpression(s)
	if err != nil {
		return Condition{}, err
	}

	c.AndExpression = ae

	return c, nil
}

func parseAndExpression(definition string) (AndExpression, error) {
	parts := splitByFirstRegexpGroup(definition, AndSeparatorRe)
	orExpressions := []OrExpression{}
	for i := 0; i < len(parts); i += 2 {
		or, err := parseOrExpression(parts[i])
		if err != nil {
			return AndExpression{}, err
		}
		orExpressions = append(orExpressions, or)
	}

	return AndExpression(orExpressions), nil
}

func parseOrExpression(definition string) (OrExpression, error) {
	parts := splitByFirstRegexpGroup(definition, OrSeparatorRe)
	expressions := []Expression{}
	for i := 0; i < len(parts); i += 2 {
		or, err := parseExpression(parts[i])
		if err != nil {
			return OrExpression{}, err
		}
		expressions = append(expressions, or)
	}

	return OrExpression(expressions), nil
}

func parseExpression(definition string) (Expression, error) {
	e := Expression{}
	s := definition
	mss := []MetricScope{PerHit, PerUser, PerSession}
	for _, ms := range mss {
		if strings.HasPrefix(s, ms.String()) {
			e.MetricScope = ms
			s = s[len(ms.String()):]
			break
		}
	}

	idxes := OpSeparatorRe.FindAllStringIndex(s, -1)
	if len(idxes) == 0 {
		return Expression{}, fmt.Errorf(fmt.Sprintf("invalid expression: %s", definition))
	}
	opi := idxes[0]

	e.Target = DimensionOrMetric(s[:opi[0]])
	if e.Target == "" {
		return Expression{}, fmt.Errorf("empty dimension or metric")
	}
	e.Operator = Operator(s[opi[0]:opi[1]])
	e.Value = UnEscapeExpressionValue(s[opi[1]:])
	return e, nil
}

func splitByFirstRegexpGroup(s string, r *regexp.Regexp) []string {
	indexes := r.FindAllStringSubmatchIndex(s, -1)
	if len(indexes) == 0 {
		return []string{s}
	}
	ret := make([]string, len(indexes)*2+1)
	last := []int{0, 0}
	for i, idx := range indexes {
		if len(idx) < 4 {
			panic("regexp must contains group")
		}
		pos := []int{idx[2], idx[3]}
		ret[i*2] = s[last[1]:pos[0]]
		ret[i*2+1] = s[pos[0]:pos[1]]
		last = pos
	}
	ret[len(ret)-1] = s[last[1]:]
	return ret
}
