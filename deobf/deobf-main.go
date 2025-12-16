package deobf

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/post04/go-fast/generator"
	"github.com/post04/go-fast/parser"
	"github.com/post04/go-fast/resolver"
	"github.com/post04/go-fast/transform/simplifier"
)

type scriptType int

var (
	prettyRegex = regexp.MustCompile(`[    ]+\n[    ]+\n`)
)

const (
	ScriptTypeTAGS scriptType = iota
	ScriptTypeCaptcha
	ScriptTypeInter
)

func Deobfuscate(code string, t scriptType) string {
	switch t {
	case ScriptTypeTAGS:
		return deobfTags(code)
	case ScriptTypeCaptcha:
		return deobfCaptcha(code)
	case ScriptTypeInter:
		return deobfInter(code)
	}
	return code
}

func deobfTags(code string) string {
	parsed, err := parser.ParseFile(code)
	if err != nil {
		return ""
	}
	simplifier.Simplify(parsed, true)
	resolver.Resolve(parsed)

	// get all needed functions
	start := time.Now()
	getFunctionsVisitor := &GetFunctionsVisitor{}
	getFunctionsVisitor.V = getFunctionsVisitor
	parsed.VisitChildrenWith(getFunctionsVisitor)
	fmt.Println("Time taken to get functions:", time.Since(start))

	// deobf ints
	start = time.Now()
	deobfIntsVisitor := &deobfInts{Functions: getFunctionsVisitor.Functions}
	deobfIntsVisitor.V = deobfIntsVisitor
	parsed.VisitChildrenWith(deobfIntsVisitor)
	fmt.Println("Time taken to deobf ints (first pass):", time.Since(start))

	// get all var decs
	start = time.Now()
	getVarDecVisitor1 := &getVarDecVisitor{}
	getVarDecVisitor1.V = getVarDecVisitor1
	parsed.VisitChildrenWith(getVarDecVisitor1)
	fmt.Println("Time taken to get var decs:", time.Since(start))

	start = time.Now()
	fmt.Println("num of var decs:", len(getVarDecVisitor1.VarDecs))
	replaceStringFromCharcodeVisitor := &replaceStringFromCharcode{VarDecs: getVarDecVisitor1.VarDecs}
	replaceStringFromCharcodeVisitor.V = replaceStringFromCharcodeVisitor
	parsed.VisitChildrenWith(replaceStringFromCharcodeVisitor)
	fmt.Println("Time taken to replace string from charcode:", time.Since(start))

	start = time.Now()
	stringConcatVisitor := &StringConcatVisitor{}
	stringConcatVisitor.V = stringConcatVisitor
	parsed.VisitChildrenWith(stringConcatVisitor)
	fmt.Println("Time taken to string concat:", time.Since(start))

	start = time.Now()
	getDeobfKeyVisitor := &getDeobfKeyVisitor{ScriptType: ScriptTypeTAGS, VarDecs: getVarDecVisitor1.VarDecs}
	getDeobfKeyVisitor.V = getDeobfKeyVisitor
	parsed.VisitChildrenWith(getDeobfKeyVisitor)
	fmt.Println("Time taken to get deobf key:", time.Since(start))
	if getDeobfKeyVisitor.Key == "" {
		fmt.Println("No key found, returning original code")
		return generator.Generate(parsed)
	}
	fmt.Println("Key found: " + getDeobfKeyVisitor.Key)

	start = time.Now()
	deobfStringsVisitor := &deobfStrings{DecodeFunctionId: getDeobfKeyVisitor.DecodeFunctionId, Base64FunctionId: getDeobfKeyVisitor.Base64FunctionId, Key: getDeobfKeyVisitor.Key, ArrayDecode: getDeobfKeyVisitor.ArrayDecode, ArrayBase64: getDeobfKeyVisitor.ArrayBase64}
	deobfStringsVisitor.V = deobfStringsVisitor
	parsed.VisitChildrenWith(deobfStringsVisitor)
	fmt.Println("Time taken to deobf strings:", time.Since(start))

	start = time.Now()
	cleanupArrayVisitor := &cleanupArrayVisitor{ArrayDecodeVarDecId: getDeobfKeyVisitor.ArrayDecodeVarDecId, ArrayBase64VarDecId: getDeobfKeyVisitor.ArrayBase64VarDecId}
	cleanupArrayVisitor.V = cleanupArrayVisitor
	parsed.VisitChildrenWith(cleanupArrayVisitor)
	fmt.Println("Time taken to cleanup array:", time.Since(start))

	// deobf ints again
	start = time.Now()
	deobfIntsVisitor = &deobfInts{SecondPass: true, Functions: getFunctionsVisitor.Functions, UsedFunctions: deobfIntsVisitor.UsedFunctions}
	deobfIntsVisitor.V = deobfIntsVisitor
	parsed.VisitChildrenWith(deobfIntsVisitor)
	fmt.Println("Time taken to deobf ints (second pass):", time.Since(start))

	// cleanup used functions
	start = time.Now()
	cleanupUsedFunctionsVisitor := &cleanupUsedFunctionsVisitor{UsedFunctions: deobfIntsVisitor.UsedFunctions}
	cleanupUsedFunctionsVisitor.V = cleanupUsedFunctionsVisitor
	parsed.VisitChildrenWith(cleanupUsedFunctionsVisitor)
	fmt.Println("Time taken to cleanup used functions:", time.Since(start))

	// replace window
	start = time.Now()
	replaceWindowVisitor := &replaceWindowVisitor{VarDecs: getVarDecVisitor1.VarDecs}
	replaceWindowVisitor.V = replaceWindowVisitor
	parsed.VisitChildrenWith(replaceWindowVisitor)
	fmt.Println("Time taken to replace window:", time.Since(start))

	// math
	start = time.Now()
	mathEvaluateVisitor := &mathEvaluate{}
	mathEvaluateVisitor.V = mathEvaluateVisitor
	parsed.VisitChildrenWith(mathEvaluateVisitor)
	fmt.Println("Time taken to evaluate math:", time.Since(start))

	// get var decs again
	start = time.Now()
	getVarDecVisitor2 := &getVarDecVisitor{}
	getVarDecVisitor2.V = getVarDecVisitor2
	parsed.VisitChildrenWith(getVarDecVisitor2)
	fmt.Println("Time taken to get var decs:", time.Since(start))

	start = time.Now()
	replaceMathVisitor := &replaceMathVisitor{ReplacedVarDecs: mathEvaluateVisitor.replacedVarDecs, VarDecs: getVarDecVisitor2.VarDecs}
	replaceMathVisitor.V = replaceMathVisitor
	parsed.VisitChildrenWith(replaceMathVisitor)
	fmt.Println("Time taken to replace math:", time.Since(start))

	start = time.Now()
	compressBinaryExpressionVisitor := &compressBinaryExpressionVisitor{}
	compressBinaryExpressionVisitor.V = compressBinaryExpressionVisitor
	parsed.VisitChildrenWith(compressBinaryExpressionVisitor)
	fmt.Println("Time taken to compress binary expression:", time.Since(start))

	start = time.Now()
	getVarDecVisitor3 := &getVarDecVisitor{Remove: mathEvaluateVisitor.replacedVarDecs}
	getVarDecVisitor3.V = getVarDecVisitor3
	parsed.VisitChildrenWith(getVarDecVisitor3)
	fmt.Println("Time taken to cleanup finished var decs:", time.Since(start))

	// get CFF ints
	start = time.Now()
	getCFFIntsVisitor := &getCFFInts{Functions: getFunctionsVisitor.Functions}
	getCFFIntsVisitor.V = getCFFIntsVisitor
	parsed.VisitChildrenWith(getCFFIntsVisitor)
	fmt.Println("Time taken to get CFF ints:", time.Since(start))
	if len(getCFFIntsVisitor.Ints) != 8 {
		fmt.Println("No CFF ints found, returning original code")
		return prettyRegex.ReplaceAllString(generator.Generate(parsed), "")
	}
	if getCFFIntsVisitor.ExpressionString == "" {
		fmt.Println("No expression string found, returning original code")
		return prettyRegex.ReplaceAllString(generator.Generate(parsed), "")
	}
	if len(getCFFIntsVisitor.ExpressionIdentifiersInOrder) != 8 {
		fmt.Println("No expression identifiers in order found, returning original code")
		return prettyRegex.ReplaceAllString(generator.Generate(parsed), "")
	}
	fmt.Println("CFF ints:", getCFFIntsVisitor.Ints)
	fmt.Println("Expression string:", getCFFIntsVisitor.ExpressionString)
	fmt.Println("Expression identifiers in order:", getCFFIntsVisitor.ExpressionIdentifiersInOrder)

	// control flow (final step!)
	start = time.Now()
	D := buildD(getCFFIntsVisitor.Ints, getCFFIntsVisitor.ExpressionString, getCFFIntsVisitor.ExpressionIdentifiersInOrder)
	deobfControlFlowVisitor := &deobfControlFlow{D: D, VarDecs: getVarDecVisitor3.VarDecs}
	deobfControlFlowVisitor.V = deobfControlFlowVisitor
	parsed.VisitChildrenWith(deobfControlFlowVisitor)
	fmt.Println("Time taken to deobf control flow:", time.Since(start))

	code1 := generator.Generate(parsed)
	code1 = strings.ReplaceAll(code1, "    ;", "")

	return prettyRegex.ReplaceAllString(code1, "")
}

func deobfInter(code string) string {
	return ""
}
func deobfCaptcha(code string) string {
	return ""
}
