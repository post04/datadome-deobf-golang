package deobf

import (
	"fmt"
	"math"
	"slices"
	"strings"

	matheval "github.com/post04/datadome-deobf-golang/mathEval"
	"github.com/post04/go-fast/ast"
	"github.com/post04/go-fast/generator"
	"github.com/post04/go-fast/token"
)

type mathFunction int

const (
	mathFunctionCeil mathFunction = iota
	mathFunctionFloor
	mathFunctionParseInt
	mathFunctionNumber
	mathFunctionUnknown
	mathFunctionInvalid
)

type mathEvaluate struct {
	ast.NoopVisitor

	replacedVarDecs []ast.Id
}

func (v *mathEvaluate) VisitVariableDeclarator(node *ast.VariableDeclarator) {
	if node == nil || node.Target == nil || node.Target.Target == nil {
		return
	}
	if node.Initializer == nil || node.Initializer.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Target.Target.(*ast.Identifier); !ok {
		return
	}
	varDecID := node.Target.Target.(*ast.Identifier).ToId()
	if _, ok := node.Initializer.Expr.(*ast.CallExpression); !ok {
		return
	}
	// make sure the expr is a call expr
	callExpr, ok := node.Initializer.Expr.(*ast.CallExpression)
	if !ok {
		return
	}
	// first check if the callExpression has 2 arguments
	if len(callExpr.ArgumentList) != 1 {
		return
	}
	// make sure the argument is a number literal or unary expression where right is a number literal
	valid := false
	number := 0.0
	for _, arg := range callExpr.ArgumentList {
		if _, ok := arg.Expr.(*ast.NumberLiteral); ok {
			valid = true
			number = arg.Expr.(*ast.NumberLiteral).Value
			continue
		}
		if unaryExpr, ok := arg.Expr.(*ast.UnaryExpression); ok {
			if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
				valid = true
				number = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
				continue
			}
		}
	}
	if !valid {
		return
	}
	// evaluate which window function is being called
	callExprMemberExpr, ok := callExpr.Callee.Expr.(*ast.MemberExpression)
	if !ok {
		return
	}
	function := findMathFunction(callExprMemberExpr)
	if function == mathFunctionInvalid || function == mathFunctionUnknown {
		return
	}
	switch function {
	case mathFunctionNumber:
		node.Initializer.Expr = &ast.NumberLiteral{Value: number}
	case mathFunctionCeil:
		ceil := math.Ceil(number)
		node.Initializer.Expr = &ast.NumberLiteral{Value: ceil}
	case mathFunctionFloor, mathFunctionParseInt:
		floor := math.Floor(number)
		node.Initializer.Expr = &ast.NumberLiteral{Value: floor}
	}
	if !slices.Contains(v.replacedVarDecs, varDecID) {
		v.replacedVarDecs = append(v.replacedVarDecs, varDecID)
	}
}

func findMathFunction(node *ast.MemberExpression) mathFunction {
	if node == nil || node.Object == nil || node.Property == nil {
		return mathFunctionUnknown
	}

	// parseInt, Number
	if nodeIdentifier, ok := node.Object.Expr.(*ast.Identifier); ok {

		if nodeIdentifier == nil {
			return mathFunctionInvalid
		}

		if nodeIdentifier.Name != "window" {
			return mathFunctionInvalid
		}
		if nodeProp, ok := node.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.StringLiteral); ok {
			if nodeProp.Value == "parseInt" {
				return mathFunctionParseInt
			}
			if nodeProp.Value == "Number" {
				return mathFunctionNumber
			}
			return mathFunctionInvalid
		}
	}
	// object should now be a member expression where object is identifier "window" and property is string literal "Math"
	if nodeMemberExpr, ok := node.Object.Expr.(*ast.MemberExpression); ok {
		if nodeMemberExpr == nil || nodeMemberExpr.Object == nil || nodeMemberExpr.Property == nil {
			return mathFunctionInvalid
		}
		if _, ok := nodeMemberExpr.Object.Expr.(*ast.Identifier); !ok {
			return mathFunctionInvalid
		}
		if nodeMemberExpr.Object.Expr.(*ast.Identifier).Name != "window" {
			return mathFunctionInvalid
		}
		// make sure the property is a string literal
		if _, ok := nodeMemberExpr.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.StringLiteral); !ok {
			return mathFunctionInvalid
		}
		if nodeMemberExpr.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.StringLiteral).Value != "Math" {
			return mathFunctionInvalid
		}
		if nodeProp, ok := node.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.StringLiteral); ok {
			if nodeProp.Value == "ceil" {
				return mathFunctionCeil
			}
			if nodeProp.Value == "floor" {
				return mathFunctionFloor
			}
			return mathFunctionInvalid
		}
	}
	return mathFunctionUnknown
}

type replaceMathVisitor struct {
	ast.NoopVisitor

	ReplacedVarDecs []ast.Id
	VarDecs         []*ast.VariableDeclarator
}

func (v *replaceMathVisitor) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Expr.(*ast.Identifier); !ok {
		return
	}
	identifier := node.Expr.(*ast.Identifier)
	id := identifier.ToId()
	if slices.Contains(v.ReplacedVarDecs, id) {
		for _, varDec := range v.VarDecs {
			if varDec == nil || varDec.Target == nil || varDec.Target.Target == nil {
				continue
			}
			if _, ok := varDec.Target.Target.(*ast.Identifier); !ok {
				continue
			}
			if varDec.Target.Target.(*ast.Identifier).ToId() == id {
				node.Expr = varDec.Initializer.Expr
				break
			}
		}
	}
}

type compressBinaryExpressionVisitor struct {
	ast.NoopVisitor
}

func (v *compressBinaryExpressionVisitor) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Expr.(*ast.BinaryExpression); !ok {
		return
	}
	binaryExpr := node.Expr.(*ast.BinaryExpression)
	generated := generator.Generate(binaryExpr)
	valid := matheval.ValidateExpression(generated)
	if valid != nil {
		return
	}
	eval := matheval.Eval(generated)
	if eval < 0 {
		node.Expr = &ast.UnaryExpression{
			Operator: token.Minus,
			Operand:  &ast.Expression{Expr: &ast.NumberLiteral{Value: -float64(eval)}},
		}
	} else {
		node.Expr = &ast.NumberLiteral{Value: float64(eval)}
	}
}

type cleanupFinishedvarDecsVisitor struct {
	ast.NoopVisitor

	ReplacedVarDecs []ast.Id
}

func (v *cleanupFinishedvarDecsVisitor) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if strings.Contains(generator.Generate(node), "nt = -105") && len(generator.Generate(node)) < 200 {
		fmt.Println("TEST")
	}
	if _, ok := node.Expr.(*ast.VariableDeclarator); !ok {
		return
	}
	variableDeclarator := node.Expr.(*ast.VariableDeclarator)
	if variableDeclarator.Target == nil || variableDeclarator.Target.Target == nil {
		return
	}
	if _, ok := variableDeclarator.Target.Target.(*ast.Identifier); !ok {
		return
	}

	id := variableDeclarator.Target.Target.(*ast.Identifier).ToId()
	if slices.Contains(v.ReplacedVarDecs, id) {
		fmt.Println(generator.Generate(node))
		node = nil
	}
}
