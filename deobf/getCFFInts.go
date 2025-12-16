package deobf

import (
	"github.com/post04/go-fast/ast"
	"github.com/post04/go-fast/generator"
)

type getCFFInts struct {
	ast.NoopVisitor

	Ints                         []int
	ExpressionString             string
	ExpressionIdentifiersInOrder []string

	Functions []*ast.FunctionDeclaration
}

func (v *getCFFInts) VisitStatement(node *ast.Statement) {
	if node == nil || node.Stmt == nil {
		return
	}
	node.VisitChildrenWith(v)

	// check if the expression is a VariableDeclaration
	varDeclaration, ok := node.Stmt.(*ast.VariableDeclaration)
	if !ok {
		return
	}
	// check if the variable declaration has 1 variable declarator
	if len(varDeclaration.List) != 1 {
		return
	}
	// make sure it has a target and the target is an identifier
	if varDeclaration.List[0].Target == nil || varDeclaration.List[0].Target.Target == nil {
		return
	}
	// identifier, ok := varDeclaration.List[0].Target.Target.(*ast.Identifier)
	// if !ok {
	// 	return
	// }
	// check if the variable declarator has an initializer
	if varDeclaration.List[0].Initializer == nil {
		return
	}
	// check if the initializer is a call expression
	callExpr, ok := varDeclaration.List[0].Initializer.Expr.(*ast.CallExpression)
	if !ok {
		return
	}
	// check that the callee is a functionliteral
	functionLiteral, ok := callExpr.Callee.Expr.(*ast.FunctionLiteral)
	if !ok {
		return
	}
	// check that the function literal has 1 parameter
	if len(functionLiteral.ParameterList.List) != 2 {
		return
	}
	// make sure body is 4 long, with the last one being a return statement
	if len(functionLiteral.Body.List) != 4 {
		return
	}
	if _, ok := functionLiteral.Body.List[3].Stmt.(*ast.ReturnStatement); !ok {
		return
	}
	forStmt, ok := functionLiteral.Body.List[2].Stmt.(*ast.ForStatement)
	if !ok {
		return
	}
	CallExpr, ok := forStmt.Body.Stmt.(*ast.ForStatement).Body.Stmt.(*ast.ExpressionStatement).Expression.Expr.(*ast.AssignExpression).Right.Expr.(*ast.MemberExpression).Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.CallExpression)
	if !ok {
		return
	}
	ints := []int{}
	for _, arg := range CallExpr.ArgumentList {
		if _, ok := arg.Expr.(*ast.NumberLiteral); ok {
			ints = append(ints, int(arg.Expr.(*ast.NumberLiteral).Value))
		}
		if unaryExpr, ok := arg.Expr.(*ast.UnaryExpression); ok {
			if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
				ints = append(ints, int(-unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value))
			}
		}
		if _, ok := arg.Expr.(*ast.Identifier); ok {
			ints = append(ints, 696969)
		}
	}
	ints = append(ints, int(functionLiteral.Body.List[3].Stmt.(*ast.ReturnStatement).Argument.Expr.(*ast.MemberExpression).Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.NumberLiteral).Value))
	v.Ints = ints
	function := getFunctionByNode(v.Functions, CallExpr)
	if function == nil {
		return
	}
	v.ExpressionString = generator.Generate(function.Function.Body.List[0].Stmt.(*ast.ReturnStatement).Argument.Expr)
	identifiers := []string{}
	for _, param := range function.Function.ParameterList.List {
		identifiers = append(identifiers, param.Target.Target.(*ast.Identifier).Name)
	}
	v.ExpressionIdentifiersInOrder = identifiers
}
