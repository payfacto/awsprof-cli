package profiles

import (
	"fmt"
	"maps"
	"slices"
)

// Resolver holds the user-configured ways a typed name maps to a profile.
type Resolver struct {
	// Prefixes are tried in order after exact and alias matches fail.
	Prefixes []string
	// Aliases maps a shortcut to a target, itself resolved by exact name or prefix.
	Aliases map[string]string
}

// Resolve maps a typed name to an existing profile name: exact match first,
// then a configured alias, then each prefix in order. Returns an error if
// nothing matches or if the matching alias points at no known profile.
func Resolve(input string, r Resolver, names []string) (string, error) {
	l := newLookup(r.Prefixes, names)
	if l.known[input] {
		return input, nil
	}
	if target, ok := r.Aliases[input]; ok {
		return l.aliasTarget(input, target)
	}
	if name, ok := l.prefixed(input); ok {
		return name, nil
	}
	return "", fmt.Errorf("unknown profile %q", input)
}

// AliasesByProfile groups the configured aliases by the profile each resolves
// to (aliases sorted per profile). Aliases whose target matches no profile are
// returned as errors instead. Aliases shadowed by an exact profile name are
// omitted, since typing them selects that profile, not the alias target.
func AliasesByProfile(r Resolver, names []string) (map[string][]string, []error) {
	l := newLookup(r.Prefixes, names)
	index := make(map[string][]string)
	var broken []error
	for _, alias := range slices.Sorted(maps.Keys(r.Aliases)) {
		if l.known[alias] {
			continue
		}
		name, err := l.aliasTarget(alias, r.Aliases[alias])
		if err != nil {
			broken = append(broken, err)
			continue
		}
		index[name] = append(index[name], alias)
	}
	return index, broken
}

// lookup answers name queries against a fixed set of known profile names.
type lookup struct {
	known    map[string]bool
	prefixes []string
}

func newLookup(prefixes, names []string) lookup {
	known := make(map[string]bool, len(names))
	for _, n := range names {
		known[n] = true
	}
	return lookup{known: known, prefixes: prefixes}
}

// aliasTarget resolves an alias target by exact name, then prefix. Targets
// are not themselves looked up as aliases, so aliases never chain.
func (l lookup) aliasTarget(alias, target string) (string, error) {
	if l.known[target] {
		return target, nil
	}
	if name, ok := l.prefixed(target); ok {
		return name, nil
	}
	return "", fmt.Errorf("alias %q points to unknown profile %q", alias, target)
}

// prefixed returns the first known profile named prefix+input, in prefix order.
func (l lookup) prefixed(input string) (string, bool) {
	for _, prefix := range l.prefixes {
		if candidate := prefix + input; l.known[candidate] {
			return candidate, true
		}
	}
	return "", false
}
