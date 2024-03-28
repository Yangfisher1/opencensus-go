package filter

import (
	"github.com/Yangfisher1/opencensus-go/trace"
)

// SpanFilter is a struct that judge whether a span is normal or not
// according to different rules.

type SpanFilter struct {
	avgLantecies map[string]float64
	rules        []RuleFunc // Need to be initialized before running
	deviation    float64    // Indicates whether the latency is abnormal or not
}

type RuleFunc func(*trace.SpanData, *SpanFilter) bool

// Insert the avg latency of corresponding component
func (sf *SpanFilter) UpdateComponentLatency(component string, latency float64) {
	if sf.avgLantecies == nil {
		sf.avgLantecies = make(map[string]float64)
	}

	sf.avgLantecies[component] = latency
}

// Adding rules into the filter
func (sf *SpanFilter) AddRule(rule RuleFunc) {
	sf.rules = append(sf.rules, rule)
}

func (sf *SpanFilter) Filter(span *trace.SpanData) bool {
	for _, rule := range sf.rules {
		if !rule(span, sf) {
			return false
		}
	}

	return true
}

func NewSpanFilter(deviation float64) *SpanFilter {
	sf := &SpanFilter{
		deviation: deviation,
	}
	sf.AddRule(isErrorByCode)
	sf.AddRule(isErrorByLatency)

	return sf
}
