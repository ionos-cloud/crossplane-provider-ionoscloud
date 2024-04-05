package utils

import (
	"encoding/json"

	"github.com/golang/mock/gomock"
)

// MatchEqFormatter returns a gomock.Eq that uses formatterFunc for want and got values
func MatchEqFormatter(object any, f formatterFunc) gomock.Matcher {
	return matcherFormatter(object, gomock.Eq, f)
}

// MatchEqDefaultFormatter returns a gomock.Eq that uses the default formatterFunc for want and got values
func MatchEqDefaultFormatter(object any) gomock.Matcher {
	return matcherFormatter(object, gomock.Eq, mustMarshal)
}

// matcherFormatter returns a gomock.Matcher that uses formatter for want and got values
func matcherFormatter(object any, m withMatcher, f formatterFunc) gomock.Matcher {
	return gomock.GotFormatterAdapter(gomock.GotFormatterFunc(f), gomock.WantFormatter(gomock.StringerFunc(func() string { return f(object) }), m(object)))
}

// MatchFuncFormatter returns a gomock.Matcher that uses matcherFunc matching logic and formatterFunc for want and got values
func MatchFuncFormatter(object any, matchFn matcherFunc, formatFn formatterFunc) gomock.Matcher {
	return gomock.GotFormatterAdapter(gomock.GotFormatterFunc(formatFn), matcher{x: object, matchFunc: matchFn, formatFunc: formatFn})
}

// MatchFuncDefaultFormatter returns a gomock.Matcher that uses matcherFunc matching logic and the default formatter for want and got values
func MatchFuncDefaultFormatter(object any, matchFn matcherFunc) gomock.Matcher {
	return MatchFuncFormatter(object, matchFn, mustMarshal)
}

// withMatcher function that takes in an object and returns a gomock.Matcher implementation for it, ie: Eq, Len, Nil
type withMatcher func(object any) gomock.Matcher

// matcher implements gomock.Matcher for arbitrary values using matchFunc matching logic and formatFunc formatter
type matcher struct {
	x          any
	matchFunc  matcherFunc
	formatFunc formatterFunc
}

func (m matcher) Matches(x any) bool {
	return m.matchFunc(m.x, x)
}

func (m matcher) String() string {
	return m.formatFunc(m.x)
}

// matcherFunc function that implements matching logic between two objects
type matcherFunc func(a, b any) bool

// formatter function that takes in an object and returns its serialization as a string
type formatterFunc func(object any) string

// mustMarshal default object formatterFunc,
// this is useful when mocking function calls with ionos sdk structures as it modifies the formatting to show the expanded struct with values
// instead of the default %v provided by gomock which only shows addresses for pointers
func mustMarshal(object any) string {
	bytes, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
