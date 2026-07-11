package profiles

import "fmt"

// Resolve maps a typed name to an existing profile name: exact match first,
// then each prefix in order. Returns an error if nothing matches.
func Resolve(input string, prefixes []string, names []string) (string, error) {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	if set[input] {
		return input, nil
	}
	for _, pre := range prefixes {
		if cand := pre + input; set[cand] {
			return cand, nil
		}
	}
	return "", fmt.Errorf("unknown profile %q", input)
}
