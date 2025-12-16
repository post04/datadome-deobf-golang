package deobf

import (
	"github.com/post04/go-fast/ast"
	"github.com/post04/go-fast/token"
)

type StringConcatVisitor struct {
	ast.NoopVisitor
}

func (v *StringConcatVisitor) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Expr.(*ast.BinaryExpression); !ok {
		return
	}
	binaryExpr := node.Expr.(*ast.BinaryExpression)
	if binaryExpr.Operator != token.Plus {
		return
	}
	if _, ok := binaryExpr.Left.Expr.(*ast.StringLiteral); !ok {
		return
	}
	if _, ok := binaryExpr.Right.Expr.(*ast.StringLiteral); !ok {
		return
	}
	*node = ast.Expression{
		Expr: &ast.StringLiteral{
			Value: binaryExpr.Left.Expr.(*ast.StringLiteral).Value + binaryExpr.Right.Expr.(*ast.StringLiteral).Value,
		},
	}
}
