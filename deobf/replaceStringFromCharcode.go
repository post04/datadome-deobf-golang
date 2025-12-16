package deobf

import (
	"github.com/post04/go-fast/ast"
)

type replaceStringFromCharcode struct {
	ast.NoopVisitor

	VarDecs              []*ast.VariableDeclarator
	GotID                bool
	FromCharCodeVarDecId ast.Id
}

func (v *replaceStringFromCharcode) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)

	// check if the expression is a call expression
	callExpr, ok := node.Expr.(*ast.CallExpression)
	if !ok {
		return
	}
	if _, ok := callExpr.Callee.Expr.(*ast.Identifier); !ok {
		return
	}
	// check if the call expression has 1 argument
	if len(callExpr.ArgumentList) != 1 {
		return
	}
	// check if the argument is a number literal
	num, ok := callExpr.ArgumentList[0].Expr.(*ast.NumberLiteral)
	if !ok {
		return
	}
	// check if the number literal is a number
	if num.Value < 0 {
		return
	}
	// make sure the function being called is a varDec of `String.fromCharCode`
	// valid := false
	callExprID := callExpr.Callee.Expr.(*ast.Identifier).ToId()
	if !v.GotID {
		for _, varDec := range v.VarDecs {
			if varDec == nil || varDec.Target == nil || varDec.Target.Target == nil {
				continue
			}
			if varDec.Initializer == nil || varDec.Initializer.Expr == nil {
				continue
			}
			if _, ok := varDec.Target.Target.(*ast.Identifier); !ok {
				continue
			}
			varDecID := varDec.Target.Target.(*ast.Identifier).ToId()
			// check that the varDec init is a member expression where object and property are identifers, object name is "String", and property name is "fromCharCode"
			memberExpr, ok := varDec.Initializer.Expr.(*ast.MemberExpression)
			if !ok {
				continue
			}
			if _, ok := memberExpr.Object.Expr.(*ast.Identifier); !ok {
				continue
			}
			if _, ok := memberExpr.Property.Prop.(*ast.Identifier); !ok {
				continue
			}
			if memberExpr.Object.Expr.(*ast.Identifier).Name != "String" {
				continue
			}
			if memberExpr.Property.Prop.(*ast.Identifier).Name != "fromCharCode" {
				continue
			}
			v.FromCharCodeVarDecId = varDecID
			v.GotID = true

			// if varDecID == v.FromCharCodeVarDecId && varDecID == callExprID {
			// 	valid = true
			// 	break
			// }
		}
	}

	if v.FromCharCodeVarDecId != callExprID {
		return
	}
	// replace the call expression with the string from charcode
	node.Expr = &ast.StringLiteral{Value: string(rune(num.Value))}
}

// replace window
type replaceWindowVisitor struct {
	ast.NoopVisitor

	WindowVarDecId ast.Id
	GotID          bool
	VarDecs        []*ast.VariableDeclarator
}

func (v *replaceWindowVisitor) VisitMemberExpression(node *ast.MemberExpression) {
	if node == nil || node.Object == nil || node.Property == nil {
		return
	}
	node.VisitChildrenWith(v)
	nodeIdentifier, ok := node.Object.Expr.(*ast.Identifier)
	if !ok {
		return
	}

	nodeID := nodeIdentifier.ToId()

	if !v.GotID {
		for _, varDec := range v.VarDecs {
			if varDec == nil || varDec.Target == nil || varDec.Target.Target == nil {
				continue
			}
			if varDec.Initializer == nil || varDec.Initializer.Expr == nil {
				continue
			}
			if _, ok := varDec.Target.Target.(*ast.Identifier); !ok {
				continue
			}
			varDecID := varDec.Target.Target.(*ast.Identifier).ToId()
			if varDecID == nodeID {
				identifier, ok := varDec.Initializer.Expr.(*ast.Identifier)
				if !ok {
					continue
				}
				if identifier.Name == "window" {
					v.WindowVarDecId = varDecID
					v.GotID = true
				}
			}
		}
	}

	if v.WindowVarDecId != nodeID {
		return
	}
	node.Object = &ast.Expression{Expr: &ast.Identifier{Name: "window"}}
}
