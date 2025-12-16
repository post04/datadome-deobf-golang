package deobf

import (
	"encoding/base64"
	"strings"

	"github.com/post04/go-fast/ast"
)

type deobfStrings struct {
	ast.NoopVisitor
	DecodeFunctionId ast.Id
	ArrayDecode      []any
	Key              string

	Base64FunctionId ast.Id
	ArrayBase64      []string
}

var (
	test = []string{}
)

func (v *deobfStrings) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)

	// make sure the expr is a call expr
	callExpr, ok := node.Expr.(*ast.CallExpression)
	if !ok {
		return
	}
	// first check if the callExpression has 1 argument of a number literal
	if len(callExpr.ArgumentList) != 1 {
		return
	}
	numValue := 0.0
	num, ok := callExpr.ArgumentList[0].Expr.(*ast.NumberLiteral)
	if ok && num != nil {
		numValue = num.Value
	}
	unaryExpr, ok := callExpr.ArgumentList[0].Expr.(*ast.UnaryExpression)
	if ok && unaryExpr != nil {
		// make sure it's a number literal
		if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); !ok {
			return
		}
		numValue = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
	}
	if numValue == 0 {
		return
	}

	// check for base64 function
	identifier, ok := callExpr.Callee.Expr.(*ast.Identifier)
	if !ok {
		return
	}
	identifierID := identifier.ToId()
	if identifierID == v.Base64FunctionId {
		base64Encoded := v.ArrayBase64[int(numValue)]
		if 4-(len(base64Encoded)%4) != 4 {
			base64Encoded += strings.Repeat("=", 4-(len(base64Encoded)%4))
		}
		base64Decoded, err := base64.StdEncoding.DecodeString(base64Encoded)
		if err != nil {
			panic(err.Error())
		}
		node.Expr = &ast.StringLiteral{Value: string(base64Decoded)}
		return
	}

	// get the function by the call expression
	if identifierID != v.DecodeFunctionId {
		return
	}
	if int(numValue) > len(v.ArrayDecode)-1 {
		return
	}
	// replace the call expression with the deobfuscated string
	value := deobfStringsWithKey(int64(numValue), v.Key, v.ArrayDecode)
	// test = append(test, generator.Generate(node)+" == "+fmt.Sprint(value))
	// os.WriteFile("test.txt", []byte(strings.Join(test, "\n")), 0666)
	// check if it's a string
	if val, ok := value.(string); ok {
		node.Expr = &ast.StringLiteral{Value: val}
	}
	// check if it's a float64
	if val, ok := value.(float64); ok {
		node.Expr = &ast.NumberLiteral{Value: val}
	}
	// check if it's a int64
	if val, ok := value.(int64); ok {
		node.Expr = &ast.NumberLiteral{Value: float64(val)}
	}
	// check if it's a bool
	if val, ok := value.(bool); ok {
		node.Expr = &ast.BooleanLiteral{Value: val}
	}
	// check if it's nil
	if value == nil {
		node.Expr = &ast.NullLiteral{}
	}

}
