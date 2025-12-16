package deobf

import (
	"regexp"
	"slices"
	"strings"

	"github.com/post04/go-fast/ast"
	"github.com/post04/go-fast/generator"
	"github.com/post04/go-fast/token"
)

type getDeobfKeyVisitor struct {
	ast.NoopVisitor
	Key                 string
	ArrayDecode         []any
	ArrayBase64         []string
	ArrayDecodeVarDecId ast.Id
	ArrayBase64VarDecId ast.Id
	DecodeFunctionId    ast.Id
	Base64FunctionId    ast.Id

	VarDecs []*ast.VariableDeclarator

	ScriptType scriptType
}

var keyRegex = regexp.MustCompile(`[A-z0-9]+ = \"[A-z0-9+/=]+\"`)

func (v *getDeobfKeyVisitor) VisitFunctionDeclaration(node *ast.FunctionDeclaration) {
	if node == nil || node.Function == nil {
		return
	}
	node.VisitChildrenWith(v)

	var varDecDecl ast.VariableDeclarator
	found := false
	// check that the body is 2 long, first being a variable declaration, second being a return statement
	if v.ScriptType == ScriptTypeTAGS {
		if len(node.Function.Body.List) != 2 {
			return
		}
		if _, ok := node.Function.Body.List[0].Stmt.(*ast.VariableDeclaration); !ok {
			return
		}
		if _, ok := node.Function.Body.List[1].Stmt.(*ast.ReturnStatement); !ok {
			return
		}

		// check that the variable declaration is a var dec, where the init is a member expression

		varDec, ok := node.Function.Body.List[0].Stmt.(*ast.VariableDeclaration)
		if !ok {
			return
		}
		// check that it's 1 long
		if len(varDec.List) != 1 {
			return
		}
		varDecDecl = varDec.List[0]
		// check that the init is a member expression
		if varDecDecl.Initializer == nil || varDecDecl.Initializer.Expr == nil {
			return
		}
		found = true
	} else {
		if len(node.Function.Body.List) != 1 {
			return
		}
		// make sure body is a return statement
		returnStmt, ok := node.Function.Body.List[0].Stmt.(*ast.ReturnStatement)
		if !ok {
			return
		}
		// if the return stmt has an argument of type SequenceExpression, generate it and check if it contains "atob("
		if sequenceExpr, ok := returnStmt.Argument.Expr.(*ast.SequenceExpression); ok {
			// check that sequence expression is 2 long
			if len(sequenceExpr.Sequence) != 2 {
				return
			}
			// check that the second sequence is a CallExpression where the identifier is "atob"
			if _, ok := sequenceExpr.Sequence[1].Expr.(*ast.CallExpression); !ok {
				return
			}
			if _, ok := sequenceExpr.Sequence[1].Expr.(*ast.CallExpression).Callee.Expr.(*ast.Identifier); !ok {
				return
			}
			if sequenceExpr.Sequence[1].Expr.(*ast.CallExpression).Callee.Expr.(*ast.Identifier).Name != "atob" {
				return
			}
			varDecDecl = ast.VariableDeclarator{
				Target: &ast.BindingTarget{
					Target: sequenceExpr.Sequence[0].Expr.(*ast.AssignExpression).Left.Expr.(*ast.Identifier),
				},
				Initializer: &ast.Expression{
					Expr: sequenceExpr.Sequence[0].Expr.(*ast.AssignExpression).Right.Expr,
				},
			}
			found = true
			// fmt.Println(generator.Generate(node), "1")
		}
		// if the return stmt is a ConditionalExpression, check if the test is a BinaryExpression
		if conditionalExpr, ok := returnStmt.Argument.Expr.(*ast.ConditionalExpression); ok {
			if binaryExpr, ok := conditionalExpr.Test.Expr.(*ast.BinaryExpression); ok {
				// check that right is a StringLiteral of "string"
				if _, ok := binaryExpr.Right.Expr.(*ast.StringLiteral); !ok {
					return
				}
				if binaryExpr.Right.Expr.(*ast.StringLiteral).Value != "string" {
					return
				}
				assignExpr := conditionalExpr.Test.Expr.(*ast.BinaryExpression).Left.Expr.(*ast.UnaryExpression).Operand.Expr.(*ast.AssignExpression)
				varDecDecl = ast.VariableDeclarator{
					Target: &ast.BindingTarget{
						Target: assignExpr.Left.Expr.(*ast.Identifier),
					},
					Initializer: &ast.Expression{
						Expr: assignExpr.Right.Expr,
					},
				}
				found = true
			}
		}
	}
	if !found {
		return
	}
	memberExpr, ok := varDecDecl.Initializer.Expr.(*ast.MemberExpression)
	if !ok {
		return
	}

	// check that the object is an identifier
	if _, ok := memberExpr.Object.Expr.(*ast.Identifier); !ok {
		return
	}

	valuesArray := []any{}
	varDecObjectID := memberExpr.Object.Expr.(*ast.Identifier).ToId()
	code := generator.Generate(node)
	isAtob := strings.Contains(code, "atob")
	for _, varDec := range v.VarDecs {
		if varDec == nil || varDec.Target == nil || varDec.Target.Target == nil {
			continue
		}
		if varDec.Target.Target.(*ast.Identifier).ToId() == varDecObjectID {
			for _, value := range varDec.Initializer.Expr.(*ast.ArrayLiteral).Value {
				// check if the value is a number literal or unary expression
				if _, ok := value.Expr.(*ast.NumberLiteral); ok {
					valuesArray = append(valuesArray, value.Expr.(*ast.NumberLiteral).Value)
				}
				if unaryExpr, ok := value.Expr.(*ast.UnaryExpression); ok {
					if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
						if unaryExpr.Operator == token.Minus {
							valuesArray = append(valuesArray, -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value)
						} else {
							// check if it's 1 or 0
							num := unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
							if num == 1 {
								valuesArray = append(valuesArray, false)
							} else {
								valuesArray = append(valuesArray, true)
							}
						}
					}
				}
				// check if it's a bool
				if _, ok := value.Expr.(*ast.BooleanLiteral); ok {
					valuesArray = append(valuesArray, value.Expr.(*ast.BooleanLiteral).Value)
				}
				// check if it's a string literal
				if _, ok := value.Expr.(*ast.StringLiteral); ok {
					valuesArray = append(valuesArray, value.Expr.(*ast.StringLiteral).Value)
				}
				// check if it's null
				if _, ok := value.Expr.(*ast.NullLiteral); ok {
					valuesArray = append(valuesArray, nil)
				}
			}
			if isAtob {
				v.ArrayBase64VarDecId = varDec.Target.Target.(*ast.Identifier).ToId()
			} else {
				v.ArrayDecodeVarDecId = varDec.Target.Target.(*ast.Identifier).ToId()
			}
			break
		}
	}
	if isAtob {
		arr2 := []string{}
		for _, value := range valuesArray {
			if _, ok := value.(string); ok {
				arr2 = append(arr2, value.(string))
			}
		}
		v.ArrayBase64 = arr2
		v.Base64FunctionId = node.Function.Name.ToId()
	} else {
		v.ArrayDecode = valuesArray
		v.DecodeFunctionId = node.Function.Name.ToId()
		key := keyRegex.FindString(code)
		if key == "" {
			return
		}
		v.Key = strings.Split(key, " = ")[1]
		v.Key = v.Key[1 : len(v.Key)-1]
	}
	node.Function = nil
}

type getVarDecVisitor struct {
	ast.NoopVisitor
	VarDecs []*ast.VariableDeclarator
	Remove  []ast.Id
}

func (v *getVarDecVisitor) VisitVariableDeclarator(node *ast.VariableDeclarator) {
	if node == nil {
		return
	}
	node.VisitChildrenWith(v)
	v.VarDecs = append(v.VarDecs, node)
	if node.Initializer == nil || node.Initializer.Expr == nil {
		return
	}
	identifier, ok := node.Target.Target.(*ast.Identifier)
	if !ok {
		return
	}
	if slices.Contains(v.Remove, identifier.ToId()) {
		node.Initializer = nil
		node.Target = nil
		return
	}
	v.VarDecs = append(v.VarDecs, node)
}

type cleanupArrayVisitor struct {
	ast.NoopVisitor
	ArrayDecodeVarDecId ast.Id
	ArrayBase64VarDecId ast.Id
}

func (v *cleanupArrayVisitor) VisitVariableDeclarator(node *ast.VariableDeclarator) {
	if node == nil {
		return
	}
	if node.Target == nil || node.Target.Target == nil {
		return
	}
	node.VisitChildrenWith(v)
	if node.Target.Target.(*ast.Identifier).ToId() == v.ArrayDecodeVarDecId {
		node.Initializer = nil
		node.Target = nil
		return
	}
	if node.Target.Target.(*ast.Identifier).ToId() == v.ArrayBase64VarDecId {
		node.Initializer = nil
		node.Target = nil
		return
	}
}
