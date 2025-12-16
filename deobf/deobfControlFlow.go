package deobf

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	matheval "github.com/post04/datadome-deobf-golang/mathEval"
	"github.com/post04/go-fast/ast"
	"github.com/post04/go-fast/generator"
	"github.com/post04/go-fast/token"
)

type deobfControlFlow struct {
	ast.NoopVisitor

	VarDecs []*ast.VariableDeclarator
	D       *Row
}

var (
	// idk if this is gonna work but it's worth a try
	stopper = false
)

type caseClause struct {
	num1 int
	num2 int
	body ast.Statements
}

func (v *deobfControlFlow) VisitStatement(node *ast.Statement) {
	if node == nil || node.Stmt == nil {
		return
	}
	node.VisitChildrenWith(v)
	forStmt, ok := node.Stmt.(*ast.ForStatement)
	if !ok {
		return
	}
	if stopper {
		return
	}
	// make sure the test is just a boolean true
	if _, ok := forStmt.Test.Expr.(*ast.BooleanLiteral); !ok {
		return
	}
	if !forStmt.Test.Expr.(*ast.BooleanLiteral).Value {
		return
	}
	if _, ok := forStmt.Body.Stmt.(*ast.BlockStatement); !ok {
		return
	}
	switchStmt, ok := forStmt.Body.Stmt.(*ast.BlockStatement).List[0].Stmt.(*ast.SwitchStatement)
	if !ok {
		switchStmt, ok = forStmt.Body.Stmt.(*ast.BlockStatement).List[1].Stmt.(*ast.SwitchStatement)
		if !ok {
			return
		}
	}
	// make sure the switch statement's disc is an Identifier
	switchStmtDisc, ok := switchStmt.Discriminant.Expr.(*ast.Identifier)
	if !ok {
		return
	}
	switchStmtDiscID := switchStmtDisc.ToId()
	// make sure the switch statement's disc is in the var decs
	switchStmtDiscVariable := &ast.VariableDeclarator{}
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
		if varDec.Target.Target.(*ast.Identifier).ToId() == switchStmtDiscID {
			switchStmtDiscVariable = varDec
			break
		}
	}
	if switchStmtDiscVariable == nil {
		return
	}
	// make sure the switch statement's disc is a member expression
	memberExpression, ok := switchStmtDiscVariable.Initializer.Expr.(*ast.MemberExpression)
	if !ok {
		return
	}
	// check if the initilizer of the forStmt is an assignment expression, if it is, override the member expression
	if forStmt.Initializer == nil {
		return
	}
	// fmt.Println(generator.Generate(forStmt.Initializer))
	// fmt.Println(reflect.TypeOf(forStmt.Initializer.Initializer))
	num1, num2 := getNums(memberExpression)
	if exprStmt, ok := forStmt.Initializer.Initializer.(*ast.Expression); ok {
		// fmt.Println(generator.Generate(exprStmt), "TESTING", reflect.TypeOf(exprStmt.Expr))
		if assignExpr, ok := exprStmt.Expr.(*ast.AssignExpression); ok {
			// fmt.Println(generator.Generate(assignExpr), "TESTING1", reflect.TypeOf(assignExpr.Right.Expr))
			if memberExpr, ok := assignExpr.Right.Expr.(*ast.MemberExpression); ok {
				memberExpression = memberExpr
				num1, num2 = getNums(memberExpr)
				// fmt.Println(generator.Generate(memberExpression), "TESTING2", num1, num2)
			}
		}
	}
	// cleanup x[123][456] == x[123][234]
	controlFlowDeobfExtraBinaryExpressionVisitor := &controlFlowDeobfExtraBinaryExpression{D: v.D}
	controlFlowDeobfExtraBinaryExpressionVisitor.V = controlFlowDeobfExtraBinaryExpressionVisitor
	forStmt.VisitChildrenWith(controlFlowDeobfExtraBinaryExpressionVisitor)
	// cleanup true && 123 ? true : false
	controlFlowDeobfExtraLogicalExpressionVisitor := &controlFlowDeobfExtraLogicalExpression{}
	controlFlowDeobfExtraLogicalExpressionVisitor.V = controlFlowDeobfExtraLogicalExpressionVisitor
	forStmt.VisitChildrenWith(controlFlowDeobfExtraLogicalExpressionVisitor)
	// // gather all return statements into an array of their generated code
	// getAllReturnStatementsVisitor := &getAllReturnStatementsVisitor{}
	// getAllReturnStatementsVisitor.V = getAllReturnStatementsVisitor
	// forStmt.VisitChildrenWith(getAllReturnStatementsVisitor)
	// // unroll x, y
	// controlFlowDeobfExtraSequenceExpressionVisitor := &controlFlowDeobfExtraSequenceExpression{ReturnStatements: getAllReturnStatementsVisitor.ReturnStatements}
	// controlFlowDeobfExtraSequenceExpressionVisitor.V = controlFlowDeobfExtraSequenceExpressionVisitor
	// forStmt.VisitChildrenWith(controlFlowDeobfExtraSequenceExpressionVisitor)
	// cleanup if (true && 123) { ... } else { ... }
	controlFlowDeobfExtraIfStmtVisitor := &controlFlowDeobfExtraIfStmt{}
	controlFlowDeobfExtraIfStmtVisitor.V = controlFlowDeobfExtraIfStmtVisitor
	forStmt.VisitChildrenWith(controlFlowDeobfExtraIfStmtVisitor)
	// cleanup true && 123 ? true : false, doing this again cause sometimes they're doubled up and it doesn't get them
	controlFlowDeobfExtraLogicalExpressionVisitor2 := &controlFlowDeobfExtraLogicalExpression{}
	controlFlowDeobfExtraLogicalExpressionVisitor2.V = controlFlowDeobfExtraLogicalExpressionVisitor2
	forStmt.VisitChildrenWith(controlFlowDeobfExtraLogicalExpressionVisitor2)

	// fmt.Println(generator.Generate(forStmt))

	// fmt.Println(num1, num2, "TESTING3")
	body := []ast.Statement{}
	cases := []*caseClause{}
	for _, caseStmt := range switchStmt.Body {
		caseClause1, ok := caseStmt.Test.Expr.(*ast.MemberExpression)
		if !ok {
			continue
		}
		num3, num4 := getNums(caseClause1)
		cases = append(cases, &caseClause{num1: num3, num2: num4, body: caseStmt.Consequent})
	}
	// fmt.Println(num1, num2)
	// fmt.Println(generator.Generate(forStmt))
	// recursively go thru all the cases in the switch statement
	nilClause := false
	for {
		var clause *caseClause
		hitEnd := false
		for i, c := range cases {
			if v.D.cells[num1].cells[num2] == v.D.cells[c.num1].cells[c.num2] {
				// fmt.Println(c.num1)
				hit := false
				if clause == nil {
					clause = c
					if len(clause.body) != 0 {
						break
					}
				}
				if len(clause.body) != 0 {
					clause = c
					break
				}
				for !hit {
					// fmt.Println("TEST")
					if len(clause.body) == 0 {
						i++
						if len(cases) == i {
							hitEnd = true
							hit = true
							continue
						}
						clause = cases[i]
						continue
					}
					hit = true
				}

				break
			}
		}
		if hitEnd {
			break
		}
		if clause == nil {
			// fmt.Println(generator.Generate(&ast.BlockStatement{List: body}))
			//n[510][459]
			// fmt.Println(v.D.cells[510].cells[459] == v.D.cells[num1].cells[num2])
			// fmt.Println(num1, num2)
			nilClause = true
			break
		}
		// fmt.Println(clause.num1)
		// fmt.Println(generator.Generate(&ast.BlockStatement{List: clause.body}))
		// get last body, see if it's a continue, break, or return statement
		shouldBreak := false
		isContinueStatement := false
		lastStmt := clause.body[len(clause.body)-1]
		for i := 0; i < len(clause.body); i++ {
			if _, ok := clause.body[i].Stmt.(*ast.ReturnStatement); ok {
				shouldBreak = true
				lastStmt = clause.body[i]
				break
			}
		}
		if shouldBreak {
			for i := 0; i < len(clause.body); i++ {
				body = append(body, clause.body[i])
				if _, ok := clause.body[i].Stmt.(*ast.ReturnStatement); ok {
					break
				}
			}
			break
		}
		if _, ok := lastStmt.Stmt.(*ast.BreakStatement); ok {
			if len(clause.body) == 1 {
				break
			}
			shouldBreak = true
		}
		for _, stmt := range clause.body {
			if _, ok := stmt.Stmt.(*ast.ReturnStatement); ok {
				shouldBreak = true
			}
		}
		if _, ok := lastStmt.Stmt.(*ast.ContinueStatement); ok {
			if !shouldBreak {
				isContinueStatement = true
			}

		}
		// fmt.Println(num1, num2)
		// if it's a AssignmentExpression, that's the next num1 and num2
		foundAssignment := false
		assignmentWasSecond := false
		if !isContinueStatement && !shouldBreak {
			if assignmentExpr, ok := lastStmt.Stmt.(*ast.ExpressionStatement).Expression.Expr.(*ast.AssignExpression); ok {
				num1, num2 = getNums(assignmentExpr.Right.Expr.(*ast.MemberExpression))
				foundAssignment = true
			}
		}
		wasSequenceExpr := false
		seqExprBody := []ast.Statement{}
		wasBlockStmt := false
		test123 := []ast.Statement{}
		if !foundAssignment && !shouldBreak {
			// now assignment is probably len-1
			assignmentExpr := clause.body[len(clause.body)-2].Stmt

			wasIfStmt := false

			// fmt.Println(generator.Generate(assignmentExpr))
			if _, ok := assignmentExpr.(*ast.ExpressionStatement); ok {
				if _, ok := assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.BlockStatement); ok {
					test123 = assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.BlockStatement).List[:len(assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.BlockStatement).List)-1]
					assignmentExpr = assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.BlockStatement).List[len(assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.BlockStatement).List)-1].Stmt
					// fmt.Println(generator.Generate(assignmentExpr)+"RAAA123", generator.Generate(assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.AssignExpression)))
					wasBlockStmt = true
				}
			}

			if blockStmt, ok := assignmentExpr.(*ast.BlockStatement); ok {
				wasBlockStmt = true
				test123 = blockStmt.List[:len(blockStmt.List)-1]
				assignmentExpr = blockStmt.List[len(blockStmt.List)-1].Stmt
				// fmt.Println(generator.Generate(assignmentExpr) + "RAAA2")
			}
			if assignmentExpr, ok := assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.AssignExpression); ok {
				num1, num2 = getNums(assignmentExpr.Right.Expr.(*ast.MemberExpression))
				foundAssignment = true
				if !wasIfStmt {
					assignmentWasSecond = true
				}
			}
			// check if it's a sequence expression
			if sequenceExpr, ok := assignmentExpr.(*ast.ExpressionStatement).Expression.Expr.(*ast.SequenceExpression); ok {
				if assignmentExpr, ok := sequenceExpr.Sequence[len(sequenceExpr.Sequence)-1].Expr.(*ast.AssignExpression); ok {
					num1, num2 = getNums(assignmentExpr.Right.Expr.(*ast.MemberExpression))
					foundAssignment = true
					wasSequenceExpr = true
					for _, stmt := range sequenceExpr.Sequence[:len(sequenceExpr.Sequence)-1] {
						seqExprBody = append(seqExprBody, ast.Statement{Stmt: &ast.ExpressionStatement{Expression: &ast.Expression{Expr: stmt.Expr}}})
					}
				}
			}
		}
		if nilClause {
			break
		}
		if !foundAssignment && !shouldBreak {
			panic("found no new nums?" + generator.Generate(&ast.BlockStatement{List: clause.body}))
		}
		if wasBlockStmt {
			// fmt.Println(generator.Generate(&ast.BlockStatement{List: test123}), "TEST123")
			body = append(body, test123...)
			// fmt.Println(num1, num2, "TESTING123")
			if shouldBreak {
				break
			}

			continue
		}
		if assignmentWasSecond && !shouldBreak {
			body = append(body, clause.body[:len(clause.body)-2]...)
		} else if !shouldBreak && !wasSequenceExpr {
			body = append(body, clause.body[:len(clause.body)-1]...)
		} else if wasSequenceExpr {
			body = append(body, seqExprBody...)
		} else {
			body = append(body, clause.body...)
		}
		// fmt.Println(num1, num2)
		if shouldBreak {
			break
		}

	}
	if nilClause {
		return
	}
	// fmt.Println("\n\n")
	// fmt.Println(generator.Generate(&ast.BlockStatement{List: body}))
	node.Stmt = &ast.BlockStatement{List: body}
	// stopper = true
}

type controlFlowDeobfExtraBinaryExpression struct {
	ast.NoopVisitor

	D *Row
}

func (v *controlFlowDeobfExtraBinaryExpression) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Expr.(*ast.BinaryExpression); !ok {
		return
	}
	binaryExpr := node.Expr.(*ast.BinaryExpression)
	// left and right are member expressions
	if _, ok := binaryExpr.Left.Expr.(*ast.MemberExpression); !ok {
		return
	}
	if _, ok := binaryExpr.Right.Expr.(*ast.MemberExpression); !ok {
		return
	}
	leftMemberExpr := binaryExpr.Left.Expr.(*ast.MemberExpression)
	rightMemberExpr := binaryExpr.Right.Expr.(*ast.MemberExpression)
	// make sure prop is a numberliteral and object is a member expression
	if _, ok := leftMemberExpr.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.NumberLiteral); !ok {
		return
	}
	if _, ok := rightMemberExpr.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.NumberLiteral); !ok {
		return
	}
	num1, num2 := getNums(leftMemberExpr)
	num3, num4 := getNums(rightMemberExpr)
	// fmt.Println(num1, num2)
	// fmt.Println(num3, num4)
	// check if it's true or false
	if memberExprIsTrue(num1, num2, num3, num4, binaryExpr.Operator, v.D) {
		node.Expr = &ast.BooleanLiteral{Value: true}
	} else {
		node.Expr = &ast.BooleanLiteral{Value: false}
	}
}

type controlFlowDeobfExtraLogicalExpression struct {
	ast.NoopVisitor
}

func (v *controlFlowDeobfExtraLogicalExpression) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)

	if _, ok := node.Expr.(*ast.ConditionalExpression); !ok {
		return
	}
	conditionalExpr := node.Expr.(*ast.ConditionalExpression)

	switch conditionalExpr.Test.Expr.(type) {
	case *ast.BinaryExpression:

		binaryExpr := conditionalExpr.Test.Expr.(*ast.BinaryExpression)
		t, err := complexIsTrue(binaryExpr)
		if err != nil {
			return
		}

		if t {
			node.Expr = conditionalExpr.Consequent.Expr
			return
		}
		node.Expr = conditionalExpr.Alternate.Expr
		return
	case *ast.NumberLiteral:
		if conditionalExpr.Test.Expr.(*ast.NumberLiteral).Value != 0 {
			node.Expr = conditionalExpr.Consequent.Expr
			return
		}
		node.Expr = conditionalExpr.Alternate.Expr
		return
	case *ast.BooleanLiteral:
		if conditionalExpr.Test.Expr.(*ast.BooleanLiteral).Value {
			node.Expr = conditionalExpr.Consequent.Expr
			return
		}
		node.Expr = conditionalExpr.Alternate.Expr
		return
	case *ast.UnaryExpression:
		unaryExpr := conditionalExpr.Test.Expr.(*ast.UnaryExpression)
		if unaryExpr.Operator == token.Not {
			t, err := isTrue(unaryExpr.Operand)
			if err != nil {
				return
			}
			if t {
				node.Expr = conditionalExpr.Alternate.Expr
			} else {
				node.Expr = conditionalExpr.Consequent.Expr
			}
			return
		}
		// get numeric literal
		if numberLiteral, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
			if numberLiteral.Value != 0 {
				node.Expr = conditionalExpr.Consequent.Expr
			} else {
				node.Expr = conditionalExpr.Alternate.Expr
			}
			return
		}
	}
}

func complexIsTrue(binaryExpr *ast.BinaryExpression) (bool, error) {

	// fmt.Println(generator.Generate(binaryExpr))
	// fmt.Println(reflect.TypeOf(binaryExpr.Left.Expr))
	// fmt.Println(reflect.TypeOf(binaryExpr.Right.Expr))
	true1 := false
	num1, num2 := 0.0, 0.0
	true2 := false
	switch binaryExpr.Left.Expr.(type) {
	case *ast.BooleanLiteral:
		true1 = binaryExpr.Left.Expr.(*ast.BooleanLiteral).Value
	case *ast.NumberLiteral:
		true1 = binaryExpr.Left.Expr.(*ast.NumberLiteral).Value != 0
		num1 = binaryExpr.Left.Expr.(*ast.NumberLiteral).Value
	case *ast.UnaryExpression:
		unaryExpr := binaryExpr.Left.Expr.(*ast.UnaryExpression)
		if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); !ok {
			return false, errors.New("not a number literal")
		}
		if unaryExpr.Operator == token.Not {
			true1 = unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value == 0
		} else {
			true1 = unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value != 0
			num1 = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
		}
	case *ast.BinaryExpression:
		binaryExpr := binaryExpr.Left.Expr.(*ast.BinaryExpression)
		if binaryExpr.Operator == token.Greater || binaryExpr.Operator == token.GreaterOrEqual || binaryExpr.Operator == token.Less || binaryExpr.Operator == token.LessOrEqual {
			leftNum := 0.0
			rightNum := 0.0
			if _, ok := binaryExpr.Left.Expr.(*ast.NumberLiteral); ok {
				leftNum = binaryExpr.Left.Expr.(*ast.NumberLiteral).Value
			}
			if _, ok := binaryExpr.Left.Expr.(*ast.UnaryExpression); ok {
				unaryExpr := binaryExpr.Left.Expr.(*ast.UnaryExpression)
				if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
					leftNum = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
				}
			}
			if _, ok := binaryExpr.Right.Expr.(*ast.NumberLiteral); ok {
				rightNum = binaryExpr.Right.Expr.(*ast.NumberLiteral).Value
			}
			if _, ok := binaryExpr.Right.Expr.(*ast.UnaryExpression); ok {
				unaryExpr := binaryExpr.Right.Expr.(*ast.UnaryExpression)
				if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
					rightNum = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
				}
			}
			switch binaryExpr.Operator {
			case token.Greater:
				true1 = leftNum > rightNum
			case token.GreaterOrEqual:
				true1 = leftNum >= rightNum
			case token.Less:
				true1 = leftNum < rightNum
			case token.LessOrEqual:
				true1 = leftNum <= rightNum
			default:
				t, err := complexIsTrue(binaryExpr)
				if err != nil {
					return false, err
				}
				true1 = t
			}
		} else {
			t, err := complexIsTrue(binaryExpr)
			if err != nil {
				return false, err
			}
			true1 = t
		}
	case *ast.ConditionalExpression:
		conditionalExpr := binaryExpr.Left.Expr.(*ast.ConditionalExpression)
		t, err := complexIsTrue(conditionalExpr.Test.Expr.(*ast.BinaryExpression))
		if err != nil {
			return false, err
		}
		true1 = t
	default:
		return false, errors.New("not a boolean literal or number literal or binary expression or conditional expression")
	}
	switch binaryExpr.Right.Expr.(type) {
	case *ast.BooleanLiteral:
		true2 = binaryExpr.Right.Expr.(*ast.BooleanLiteral).Value
	case *ast.NumberLiteral:
		true2 = binaryExpr.Right.Expr.(*ast.NumberLiteral).Value != 0
		num2 = binaryExpr.Right.Expr.(*ast.NumberLiteral).Value
	case *ast.UnaryExpression:
		unaryExpr := binaryExpr.Right.Expr.(*ast.UnaryExpression)
		if _, ok := unaryExpr.Operand.Expr.(*ast.BinaryExpression); ok {
			binaryExpr := unaryExpr.Operand.Expr.(*ast.BinaryExpression)
			t, err := complexIsTrue(binaryExpr)
			if err != nil {
				return false, err
			}
			true2 = !t
		} else {
			if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); !ok {
				return false, errors.New("not a number literal")
			}
			if unaryExpr.Operator == token.Not {
				true2 = unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value == 0
			} else {
				true2 = unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value != 0
				num2 = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
			}
		}

	case *ast.BinaryExpression:
		binaryExpr := binaryExpr.Right.Expr.(*ast.BinaryExpression)
		if binaryExpr.Operator == token.Greater || binaryExpr.Operator == token.GreaterOrEqual || binaryExpr.Operator == token.Less || binaryExpr.Operator == token.LessOrEqual {
			leftNum := 0.0
			rightNum := 0.0
			if _, ok := binaryExpr.Left.Expr.(*ast.NumberLiteral); ok {
				leftNum = binaryExpr.Left.Expr.(*ast.NumberLiteral).Value
			}
			if _, ok := binaryExpr.Left.Expr.(*ast.UnaryExpression); ok {
				unaryExpr := binaryExpr.Left.Expr.(*ast.UnaryExpression)
				if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
					leftNum = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
				}
			}
			if _, ok := binaryExpr.Right.Expr.(*ast.NumberLiteral); ok {
				rightNum = binaryExpr.Right.Expr.(*ast.NumberLiteral).Value
			}
			if _, ok := binaryExpr.Right.Expr.(*ast.UnaryExpression); ok {
				unaryExpr := binaryExpr.Right.Expr.(*ast.UnaryExpression)
				if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
					rightNum = -unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value
				}
			}
			switch binaryExpr.Operator {
			case token.Greater:
				true2 = leftNum > rightNum
			case token.GreaterOrEqual:
				true2 = leftNum >= rightNum
			case token.Less:
				true2 = leftNum < rightNum
			case token.LessOrEqual:
				true2 = leftNum <= rightNum
			default:
				t, err := complexIsTrue(binaryExpr)
				if err != nil {
					return false, err
				}
				true2 = t
			}
		} else {
			t, err := complexIsTrue(binaryExpr)
			if err != nil {
				return false, err
			}
			true2 = t
		}
	case *ast.ConditionalExpression:
		conditionalExpr := binaryExpr.Right.Expr.(*ast.ConditionalExpression)
		t, err := complexIsTrue(conditionalExpr.Test.Expr.(*ast.BinaryExpression))
		if err != nil {
			return false, err
		}
		true2 = t
	default:
		return false, errors.New("not a boolean literal or number literal or binary expression or conditional expression")
	}
	switch binaryExpr.Operator {
	case token.LogicalAnd:
		return true1 && true2, nil
	case token.LogicalOr:
		return true1 || true2, nil
	case token.Equal, token.StrictEqual:
		return true1 == true2, nil
	case token.NotEqual, token.StrictNotEqual:
		return true1 != true2, nil
	case token.Greater:
		return num1 > num2, nil
	case token.GreaterOrEqual:
		return num1 >= num2, nil
	case token.Less:
		return num1 < num2, nil
	case token.LessOrEqual:
		return num1 <= num2, nil
	}
	// fmt.Println(generator.Generate(binaryExpr))
	// fmt.Println(binaryExpr.Operator)
	return false, errors.New("not a logical and or or")
}

type getAllReturnStatementsVisitor struct {
	ast.NoopVisitor
	ReturnStatements []string
}

func (v *getAllReturnStatementsVisitor) VisitStatement(node *ast.Statement) {
	if node == nil || node.Stmt == nil {
		return
	}
	node.VisitChildrenWith(v)
	returnStatement, ok := node.Stmt.(*ast.ReturnStatement)
	if !ok {
		return
	}
	v.ReturnStatements = append(v.ReturnStatements, generator.Generate(returnStatement.Argument))
}

type controlFlowDeobfExtraSequenceExpression struct {
	ast.NoopVisitor

	ReturnStatements []string
}

func (v *controlFlowDeobfExtraSequenceExpression) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	sequenceExpr, ok := node.Expr.(*ast.SequenceExpression)
	if !ok {
		return
	}
	if slices.Contains(v.ReturnStatements, generator.Generate(node)) {
		return
	}
	// unroll the sequence expression
	seqExprBody := []ast.Statement{}
	for _, stmt := range sequenceExpr.Sequence {
		seqExprBody = append(seqExprBody, ast.Statement{Stmt: &ast.ExpressionStatement{Expression: &ast.Expression{Expr: stmt.Expr}}})
	}
	node.Expr = &ast.BlockStatement{List: seqExprBody}
}

type controlFlowDeobfExtraIfStmt struct {
	ast.NoopVisitor
}

func (v *controlFlowDeobfExtraIfStmt) VisitStatement(node *ast.Statement) {
	if node == nil || node.Stmt == nil {
		return
	}
	node.VisitChildrenWith(v)
	ifStmt, ok := node.Stmt.(*ast.IfStatement)
	if !ok {
		return
	}
	// if generator.Generate(ifStmt.Test) == "-330 < -670 || 840" {
	// 	panic(fmt.Sprint(isTrue(ifStmt.Test)))
	// }
	// fmt.Println(generator.Generate(ifStmt))
	t, err := isTrue(ifStmt.Test)
	if err != nil {
		return
	}
	if t {
		node.Stmt = ifStmt.Consequent.Stmt
	} else {
		if ifStmt.Alternate == nil {
			// fmt.Println("TESTING123", generator.Generate(ifStmt))
			// fmt.Println(err)
		}
		node.Stmt = ifStmt.Alternate.Stmt
	}
}

func isTrue(node *ast.Expression) (bool, error) {
	if _, ok := node.Expr.(*ast.BooleanLiteral); ok {
		return node.Expr.(*ast.BooleanLiteral).Value, nil
	}
	if _, ok := node.Expr.(*ast.NumberLiteral); ok {
		return node.Expr.(*ast.NumberLiteral).Value != 0, nil
	}
	if binaryExpr, ok := node.Expr.(*ast.BinaryExpression); ok {
		return complexIsTrue(binaryExpr)
	}
	if unaryExpr, ok := node.Expr.(*ast.UnaryExpression); ok {
		if binaryExpr, ok := unaryExpr.Operand.Expr.(*ast.BinaryExpression); ok {
			t, err := complexIsTrue(binaryExpr)
			if err != nil {
				return false, err
			}
			return !t, nil
		}
		if unaryExpr.Operator == token.Not {
			if numberLiteral, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
				return numberLiteral.Value == 0, nil
			}
			if booleanLiteral, ok := unaryExpr.Operand.Expr.(*ast.BooleanLiteral); ok {
				return !booleanLiteral.Value, nil
			}
		}
		if numberLiteral, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
			return numberLiteral.Value != 0, nil
		}
		if booleanLiteral, ok := unaryExpr.Operand.Expr.(*ast.BooleanLiteral); ok {
			return !booleanLiteral.Value, nil
		}
		return false, errors.New("not a boolean or number literal or binary expression")

	}
	return false, errors.New("not a boolean or number literal or binary expression")
}

func memberExprIsTrue(num1, num2, num3, num4 int, t token.Token, d *Row) bool {
	// fmt.Println(d.cells[num1].cells[num2] == d.cells[num3].cells[num4])
	switch t {
	case token.Equal, token.StrictEqual:
		return d.cells[num1].cells[num2] == d.cells[num3].cells[num4]
	case token.NotEqual, token.StrictNotEqual:
		return d.cells[num1].cells[num2] != d.cells[num3].cells[num4]
	}
	panic("???")
}

func getNums(node *ast.MemberExpression) (int, int) {
	// fmt.Println(generator.Generate(node))
	if _, ok := node.Object.Expr.(*ast.MemberExpression); !ok {
		return 0, 0
	}
	num1 := int(node.Object.Expr.(*ast.MemberExpression).Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.NumberLiteral).Value)
	num2 := int(node.Property.Prop.(*ast.ComputedProperty).Expr.Expr.(*ast.NumberLiteral).Value)
	return num1, num2
}

type Row struct {
	cells [512]*Row
}

func buildD(ints []int, expression string, identifiers []string) *Row {
	n := make([]*Row, 128)

	for e := 0; e < 128; e++ {
		n[e] = &Row{}
	}

	for o := 0; o < 512; o++ {
		for t := 0; t < 128; t++ {
			arr := []uint32{}
			varsAdded := 0
			for _, i := range ints {
				if i == 696969 {
					if varsAdded == 0 {
						arr = append(arr, uint32(o))
					}
					if varsAdded == 1 {
						arr = append(arr, uint32(t))
					}
					varsAdded++
					continue
				}
				arr = append(arr, uint32(i))
			}
			expr := expression
			for x, identifier := range identifiers {
				expr = strings.ReplaceAll(expr, identifier, fmt.Sprint(arr[x]))
			}
			j := matheval.Eval(expr)
			n[t].cells[o] = n[int(j)]
		}
	}

	return n[ints[7]]
}
