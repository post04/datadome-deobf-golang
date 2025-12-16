package deobf

import (
	"fmt"
	"slices"
	"strings"

	matheval "github.com/post04/datadome-deobf-golang/mathEval"
	"github.com/post04/go-fast/ast"
	"github.com/post04/go-fast/generator"
	"github.com/post04/go-fast/token"
)

type deobfInts struct {
	ast.NoopVisitor
	Functions     []*ast.FunctionDeclaration
	UsedFunctions []ast.Id
	SecondPass    bool
}

func (v *deobfInts) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)

	// make sure the expr is a call expr
	callExpr, ok := node.Expr.(*ast.CallExpression)
	if !ok {
		return
	}
	// first check if the callExpression has 2 arguments
	if len(callExpr.ArgumentList) != 2 {
		return
	}

	// make sure both arguments are NumericLiterals or UnaryExpressions where right is a NumericLiteral
	valid := false
	numbers := [2]int64{}
	for i, arg := range callExpr.ArgumentList {
		if _, ok := arg.Expr.(*ast.NumberLiteral); ok {
			valid = true
			numbers[i] = int64(arg.Expr.(*ast.NumberLiteral).Value)
			continue
		}
		if unaryExpr, ok := arg.Expr.(*ast.UnaryExpression); ok {
			if _, ok := unaryExpr.Operand.Expr.(*ast.NumberLiteral); ok {
				valid = true
				numbers[i] = -int64(unaryExpr.Operand.Expr.(*ast.NumberLiteral).Value)
				continue
			}
		}
	}
	if !valid {
		return
	}
	// fmt.Printf("numbers: %+v\n", numbers)

	function := getFunctionByNode(v.Functions, callExpr)
	if function == nil {
		return
	}

	// validate the function being a math function
	// body is at most 2 long, with the last one being a return statement
	if len(function.Function.Body.List) > 2 {
		return
	}
	if _, ok := function.Function.Body.List[len(function.Function.Body.List)-1].Stmt.(*ast.ReturnStatement); !ok {
		return
	}
	// if we do function.Function.Body.List[len(function.Function.Body.List)-1].Stmt.(*ast.ReturnStatement).Argument
	// it will not do `()` correctly, for some reason?
	statement := generator.Generate(function.Function)
	statement = strings.Split(statement, "\n")[1]
	statement = strings.Split(statement, "return ")[1]

	statement = strings.ReplaceAll(statement, generator.Generate(&function.Function.ParameterList.List[0]), fmt.Sprint(numbers[0]))
	statement = strings.ReplaceAll(statement, generator.Generate(&function.Function.ParameterList.List[1]), fmt.Sprint(numbers[1]))

	replaceAssignExprVisitor := &replaceAssignExpr{MapToInt: map[string]int{function.Function.ParameterList.List[0].Target.Target.(*ast.Identifier).Name: int(numbers[0]), function.Function.ParameterList.List[1].Target.Target.(*ast.Identifier).Name: int(numbers[1])}}
	replaceAssignExprVisitor.V = replaceAssignExprVisitor
	function.VisitChildrenWith(replaceAssignExprVisitor)
	replaceIdentifierVisitor := &replaceIdentifierVisitor{Operator: replaceAssignExprVisitor.Token, ToReplace: replaceAssignExprVisitor.ReplacedName, ReplaceWith: replaceAssignExprVisitor.ReplacedWithName}
	replaceIdentifierVisitor.V = replaceIdentifierVisitor
	function.VisitChildrenWith(replaceIdentifierVisitor)
	// if strings.Contains(generator.Generate(node), `Jc(24, 94)`) || strings.Contains(generator.Generate(node), `Jc(21, 32)`) {
	// 	fmt.Println(generator.Generate(function))
	// 	fmt.Println(statement)
	// }
	if replaceAssignExprVisitor.ReplacedName != "" {
		// fmt.Println(replaceAssignExprVisitor.ReplacedName, replaceAssignExprVisitor.ReplacedValue)
		statement = strings.ReplaceAll(statement, replaceAssignExprVisitor.ReplacedName, fmt.Sprint(replaceAssignExprVisitor.ReplacedValue))
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%v = ", replaceAssignExprVisitor.ReplacedValue), "")
	}

	if len(function.Function.Body.List) == 2 {
		stmt := generator.Generate(function.Function.Body.List[0].Stmt.(*ast.VariableDeclaration).List[0].Initializer.Expr)
		stmt = strings.ReplaceAll(stmt, generator.Generate(&function.Function.ParameterList.List[0]), fmt.Sprint(numbers[0]))
		stmt = strings.ReplaceAll(stmt, generator.Generate(&function.Function.ParameterList.List[1]), fmt.Sprint(numbers[1]))
		if replaceAssignExprVisitor.ReplacedName != "" {
			// fmt.Println(replaceAssignExprVisitor.ReplacedName, replaceAssignExprVisitor.ReplacedValue)
			stmt = strings.ReplaceAll(stmt, replaceAssignExprVisitor.ReplacedName, fmt.Sprint(replaceAssignExprVisitor.ReplacedValue))
			stmt = strings.ReplaceAll(stmt, fmt.Sprintf("%v = ", replaceAssignExprVisitor.ReplacedValue), "")

		}
		err := matheval.ValidateExpression(stmt)
		if err != nil {
			return
		}
		eval := matheval.Eval(stmt)
		statement = strings.ReplaceAll(statement, function.Function.Body.List[0].Stmt.(*ast.VariableDeclaration).List[0].Target.Target.(*ast.Identifier).Name, fmt.Sprint(eval))
	}
	// if strings.Contains(generator.Generate(node), `Jc(24, 94)`) || strings.Contains(generator.Generate(node), `Jc(21, 32)`) {
	// 	fmt.Println(generator.Generate(function))
	// 	fmt.Println(statement)
	// }

	// fmt.Printf("statement: %s\n", statement)
	statement = strings.TrimSuffix(statement, ";")
	err := matheval.ValidateExpression(statement)
	if err != nil {
		if generator.Generate(node) == "f(378, 312)" {
			panic(statement)
		}
		return
	}
	eval := matheval.Eval(statement)
	// fmt.Printf("eval: %d\n", eval)
	// replace the call expression with the eval
	if eval < 0 {
		node.Expr = &ast.UnaryExpression{
			Operator: token.Minus,
			Operand:  &ast.Expression{Expr: &ast.NumberLiteral{Value: -float64(eval)}},
		}
	} else {
		node.Expr = &ast.NumberLiteral{Value: float64(eval)}
	}
	if !v.SecondPass {
		if !slices.Contains(v.UsedFunctions, function.Function.Name.ToId()) {
			v.UsedFunctions = append(v.UsedFunctions, function.Function.Name.ToId())
		}
	}
}

type replaceAssignExpr struct {
	ast.NoopVisitor

	MapToInt map[string]int

	ReplacedName     string
	ReplacedWithName string
	ReplacedValue    string

	Token token.Token
}

func (v *replaceAssignExpr) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Expr.(*ast.AssignExpression); !ok {
		return
	}
	assignExpr := node.Expr.(*ast.AssignExpression)
	v.ReplacedName = assignExpr.Left.Expr.(*ast.Identifier).Name
	unaryExpr, ok := assignExpr.Right.Expr.(*ast.UnaryExpression)
	if !ok {
		return
	}
	identifier, ok := unaryExpr.Operand.Expr.(*ast.Identifier)
	if !ok {
		return
	}
	toInt, ok := v.MapToInt[identifier.Name]
	if !ok {
		return
	}
	v.ReplacedValue = generator.Generate(assignExpr.Right.Expr)
	v.ReplacedValue = strings.ReplaceAll(v.ReplacedValue, identifier.Name, fmt.Sprint(toInt))
	v.ReplacedWithName = identifier.Name
	v.Token = unaryExpr.Operator
	node.Expr = &ast.UnaryExpression{
		Operator: unaryExpr.Operator,
		Operand:  &ast.Expression{Expr: &ast.Identifier{Name: identifier.Name}},
	}
}

type replaceIdentifierVisitor struct {
	ast.NoopVisitor

	ToReplace   string
	ReplaceWith string
	Operator    token.Token
}

func (v *replaceIdentifierVisitor) VisitExpression(node *ast.Expression) {
	if node == nil || node.Expr == nil {
		return
	}
	node.VisitChildrenWith(v)
	if _, ok := node.Expr.(*ast.Identifier); !ok {
		return
	}
	if node.Expr.(*ast.Identifier).Name == v.ToReplace {
		node.Expr = &ast.UnaryExpression{
			Operator: v.Operator,
			Operand:  &ast.Expression{Expr: &ast.Identifier{Name: v.ReplaceWith}},
		}
	}
}

type cleanupUsedFunctionsVisitor struct {
	ast.NoopVisitor
	UsedFunctions []ast.Id
}

func (v *cleanupUsedFunctionsVisitor) VisitFunctionDeclaration(node *ast.FunctionDeclaration) {
	if node == nil || node.Function == nil {
		return
	}
	node.VisitChildrenWith(v)
	if slices.Contains(v.UsedFunctions, node.Function.Name.ToId()) {
		node.Function = nil
		return
	}
}
