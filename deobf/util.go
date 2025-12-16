package deobf

import (
	"strings"
	"unicode"

	"github.com/post04/go-fast/ast"
)

func decodeWithAlphabet(input, alphabet string) string {
	if alphabet == "" {
		return input
	}
	var filtered []rune
	for _, r := range input {
		if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '+' || r == '/' || r == '=' {
			filtered = append(filtered, r)
		}
	}
	t := string(filtered)

	var out []byte
	n := 0

	for n < len(t) {
		if n+4 > len(t) {
			break
		}
		d := strings.IndexRune(alphabet, rune(t[n]))
		n++
		r := strings.IndexRune(alphabet, rune(t[n]))
		n++
		o := strings.IndexRune(alphabet, rune(t[n]))
		n++
		e := strings.IndexRune(alphabet, rune(t[n]))
		n++
		a := (d << 2) | (r >> 4)
		u := ((r & 15) << 4) | (o >> 2)
		i := ((o & 3) << 6) | e

		out = append(out, byte(a))
		if o != 64 {
			out = append(out, byte(u))
		}
		if e != 64 {
			out = append(out, byte(i))
		}
	}

	return string(out)
}

func deobfStringsWithKey(number int64, key string, array []any) any {
	var value any
	value = array[int(number)]
	if _, ok := value.(string); ok {
		return decodeWithAlphabet(value.(string), key)
	}
	return value
}

func getFunctionByNode(functions []*ast.FunctionDeclaration, node *ast.CallExpression) *ast.FunctionDeclaration {
	if _, ok := node.Callee.Expr.(*ast.Identifier); !ok {
		return nil
	}
	callExprID := node.Callee.Expr.(*ast.Identifier).ToId()
	for _, function := range functions {
		if function == nil || function.Function == nil {
			continue
		}
		if function.Function.Name.ToId() == callExprID {
			return function
		}
	}
	return nil
}
