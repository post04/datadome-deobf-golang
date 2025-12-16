package deobf

import (
	"github.com/post04/go-fast/ast"
)

type GetFunctionsVisitor struct {
	ast.NoopVisitor
	Functions []*ast.FunctionDeclaration
}

func (v *GetFunctionsVisitor) VisitFunctionDeclaration(node *ast.FunctionDeclaration) {
	v.Functions = append(v.Functions, node)
	node.VisitChildrenWith(v)
}
