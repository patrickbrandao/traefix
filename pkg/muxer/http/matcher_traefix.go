package http

import "fmt"

// RegisterMatcher adds a matcher to the v3 rule syntax (Traefix).
// It must be called from an init function: the matchers are read without lock once the parsers are built,
// and ParseDomains then knows the matcher too.
func RegisterMatcher(name string, builder func(params ...string) (MatcherFunc, error)) {
	httpFuncs[name] = func(tree *matchersTree, params ...string) error {
		matcher, err := builder(params...)
		if err != nil {
			return fmt.Errorf("building matcher: %w", err)
		}

		tree.matcher = matcher
		return nil
	}
}
