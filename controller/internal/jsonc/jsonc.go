// Package jsonc provides a small JSON-with-comments preprocessor.
package jsonc

import "fmt"

// StripComments removes // and /* */ comments outside quoted JSON strings.
// Newlines are preserved so downstream parsers retain useful line numbers.
func StripComments(input []byte) ([]byte, error) {
	output := append([]byte(nil), input...)
	inString := false
	escaped := false
	for i := 0; i < len(output); i++ {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch output[i] {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch output[i] {
		case '"':
			inString = true
		case '/':
			if i+1 >= len(output) {
				continue
			}
			switch output[i+1] {
			case '/':
				output[i], output[i+1] = ' ', ' '
				for i += 2; i < len(output) && output[i] != '\n' && output[i] != '\r'; i++ {
					output[i] = ' '
				}
				i--
			case '*':
				start := i
				output[i], output[i+1] = ' ', ' '
				i += 2
				closed := false
				for ; i < len(output); i++ {
					if output[i] == '*' && i+1 < len(output) && output[i+1] == '/' {
						output[i], output[i+1] = ' ', ' '
						i++
						closed = true
						break
					}
					if output[i] != '\n' && output[i] != '\r' {
						output[i] = ' '
					}
				}
				if !closed {
					return nil, fmt.Errorf("unterminated block comment at byte %d", start)
				}
			}
		}
	}
	return output, nil
}
