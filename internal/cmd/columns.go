package cmd

import (
	"fmt"
	"strings"
)

// resolveColumns turns a column spec into an ordered column list against the
// given defaults. Spec forms:
//   - "" -> defaults
//   - "a,b,c" (no +/- tokens) -> exactly those columns
//   - "+a,-b" (any +/- token) -> defaults with additions/removals
//
// Mixing plain and +/- tokens, or naming an unknown column, is an error.
func resolveColumns(defaults []string, spec string) ([]string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return defaults, nil
	}
	valid := map[string]bool{}
	for _, c := range defaults {
		valid[c] = true
	}

	tokens := strings.Split(spec, ",")
	hasMod, hasPlain := false, false
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if strings.HasPrefix(tok, "+") || strings.HasPrefix(tok, "-") {
			hasMod = true
		} else {
			hasPlain = true
		}
	}
	if hasMod && hasPlain {
		return nil, fmt.Errorf("cannot mix column replacement and +/- modifiers in %q", spec)
	}

	validList := strings.Join(defaults, ", ")

	if !hasMod {
		return resolveReplace(valid, validList, tokens)
	}
	return resolveModify(valid, validList, defaults, tokens)
}

func resolveReplace(valid map[string]bool, validList string, tokens []string) ([]string, error) {
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		name := strings.TrimSpace(tok)
		if !valid[name] {
			return nil, fmt.Errorf("unknown column %q (valid: %s)", name, validList)
		}
		out = append(out, name)
	}
	return out, nil
}

func resolveModify(valid map[string]bool, validList string, defaults []string, tokens []string) ([]string, error) {
	out := append([]string(nil), defaults...)
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		op, name := tok[0], strings.TrimSpace(tok[1:])
		if !valid[name] {
			return nil, fmt.Errorf("unknown column %q (valid: %s)", name, validList)
		}
		switch op {
		case '-':
			filtered := out[:0]
			for _, c := range out {
				if c != name {
					filtered = append(filtered, c)
				}
			}
			out = filtered
		case '+':
			present := false
			for _, c := range out {
				if c == name {
					present = true
					break
				}
			}
			if !present {
				out = append(out, name)
			}
		}
	}
	return out, nil
}
