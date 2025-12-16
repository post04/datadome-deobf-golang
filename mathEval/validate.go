package matheval

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func ValidateExpression(expr string) error {
	allowedOps := "+-*&|^~"
	parens := 0
	i := 0
	runes := []rune(expr)
	lastNonSpace := rune(0)

	isOp := func(r rune) bool {
		return strings.ContainsRune(allowedOps, r)
	}

	isShiftStart := func(idx int) bool {
		if idx >= len(runes) {
			return false
		}
		// Check for >> or <<
		if runes[idx] == '>' && idx+1 < len(runes) && runes[idx+1] == '>' {
			return true
		}
		if runes[idx] == '<' && idx+1 < len(runes) && runes[idx+1] == '<' {
			return true
		}
		return false
	}

	for i < len(runes) {
		r := runes[i]

		// Skip spaces
		if unicode.IsSpace(r) {
			i++
			continue
		}

		// Check for shift operators (multi-character)
		if isShiftStart(i) {
			if runes[i] == '>' && i+2 < len(runes) && runes[i+1] == '>' && runes[i+2] == '>' {
				// >>> operator
				if lastNonSpace == 0 || isOp(lastNonSpace) {
					return errors.New("shift operator in invalid position")
				}
				lastNonSpace = '>'
				i += 3
				continue
			} else if (runes[i] == '>' || runes[i] == '<') && i+1 < len(runes) && runes[i+1] == runes[i] {
				// >> or << operator
				if lastNonSpace == 0 || isOp(lastNonSpace) {
					return errors.New("shift operator in invalid position")
				}
				lastNonSpace = runes[i]
				i += 2
				continue
			}
		}

		// Validate characters
		if !unicode.IsDigit(r) &&
			!strings.ContainsRune(allowedOps, r) &&
			r != '(' && r != ')' {

			return fmt.Errorf("invalid character: %q", r)
		}

		// Parentheses count
		if r == '(' {
			parens++
		}
		if r == ')' {
			parens--
			if parens < 0 {
				return errors.New("parentheses mismatch")
			}
		}

		// Operator position validation
		if isOp(r) {
			// Cannot start with binary operator except unary + - ~
			if lastNonSpace == 0 && !(r == '+' || r == '-' || r == '~') {
				return fmt.Errorf("expression cannot start with '%c'", r)
			}

			// Cannot have two binary operators in a row
			if isOp(lastNonSpace) &&
				!(r == '+' || r == '-' || r == '~') &&
				!(lastNonSpace == '(') {

				return fmt.Errorf("invalid operator sequence: '%c%c'", lastNonSpace, r)
			}
		}

		// No ")(" without operator
		if lastNonSpace == ')' && r == '(' {
			return errors.New("missing operator between ')' and '('")
		}

		// No empty parentheses "()"
		if lastNonSpace == '(' && r == ')' {
			return errors.New("empty parentheses '()' are not allowed")
		}

		lastNonSpace = r
		i++
	}

	if parens != 0 {
		return errors.New("unbalanced parentheses")
	}

	// Cannot end with an operator (single char ops or shift ops)
	if strings.ContainsRune(allowedOps, lastNonSpace) {
		return fmt.Errorf("expression cannot end with operator '%c'", lastNonSpace)
	}
	// Also check for ending with < or > (could be incomplete shift operator)
	if lastNonSpace == '<' || lastNonSpace == '>' {
		return errors.New("expression cannot end with incomplete operator")
	}

	return nil
}
