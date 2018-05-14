package android

import (
	"fmt"
	"strings"
	"unicode"
)

func Expand(s string, mapping func(string) (string, error)) (string, error) {
	buf := make([]byte, 0, 2*len(s))
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' {
			if j+1 >= len(s) {
				return "", fmt.Errorf("expected character after '$'")
			}
			buf = append(buf, s[i:j]...)
			value, w, err := getMapping(s[j+1:], mapping)
			if err != nil {
				return "", err
			}
			buf = append(buf, value...)
			j += w
			i = j + 1
		}
	}
	return string(buf) + s[i:], nil
}

func getMapping(s string, mapping func(string) (string, error)) (string, int, error) {
	switch s[0] {
	case '(':
		for i := 1; i < len(s); i++ {
			if s[i] == ')' {
				ret, err := mapping(strings.TrimSpace(s[1:i]))
				return ret, i + 1, err
			}
		}
		return "", len(s), fmt.Errorf("missing )")
	case '$':
		return "$$", 1, nil
	default:
		i := strings.IndexFunc(s, unicode.IsSpace)
		if i == 0 {
			return "", 0, fmt.Errorf("unexpected character '%c' after '$'", s[0])
		} else if i == -1 {
			i = len(s)
		}
		return "", 0, fmt.Errorf("expected '(' after '$', did you mean $(%s)?", s[:i])
	}
}
