package parser

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type Golang struct{}

func NewGolang() *Golang { return &Golang{} }

func (g *Golang) Language() string { return "go" }

func (g *Golang) Available() bool { return true }

func (g *Golang) AnalyzeFile(path string) (model.FileAnalysis, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return model.FileAnalysis{}, err
	}

	result := model.FileAnalysis{
		File: path,
		Ncloc: countGoNcloc(string(source)),
	}

	fset := token.NewFileSet()
	node, err := goparser.ParseFile(fset, path, source, goparser.AllErrors)
	if err != nil {
		result.ParseError = &model.ParseError{
			Message: err.Error(),
			Line:    1,
			Column:  1,
		}
	}

	if node != nil {
		g.walkFile(node, fset, &result)
	}
	return result, nil
}

func countGoNcloc(source string) int {
	count := 0
	for _, line := range strings.Split(source, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}
		count++
	}
	return count
}

func (g *Golang) walkFile(node *ast.File, fset *token.FileSet, result *model.FileAnalysis) {
	for _, imp := range node.Imports {
		name := strings.Trim(imp.Path.Value, `"`)
		result.Imports = append(result.Imports, model.ImportInfo{
			Name: name,
			Line: fset.Position(imp.Pos()).Line,
			Used: true,
		})
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			end := fset.Position(x.End()).Line
			if end < fset.Position(x.Pos()).Line {
				end = fset.Position(x.Pos()).Line
			}
			result.Functions = append(result.Functions, model.FunctionInfo{
				Name:       x.Name.Name,
				Line:       fset.Position(x.Pos()).Line,
				EndLine:    end,
				Complexity: 1,
				ParamCount: x.Type.Params.NumFields(),
			})
		case *ast.CallExpr:
			funcName := goCallName(x.Fun)
			line := fset.Position(x.Pos()).Line
			result.Calls = append(result.Calls, model.CallInfo{
				Func:         funcName,
				Line:         line,
				HasUserInput: goCallHasVariable(x),
				DynamicSQL:   strings.Contains(strings.ToLower(funcName), "query") || strings.Contains(strings.ToLower(funcName), "exec"),
				VariableArg:  goCallHasVariable(x),
			})
		case *ast.AssignStmt, *ast.ValueSpec:
			g.collectSecrets(x, fset, result)
		}
		return true
	})
}

func goCallName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return goCallName(x.X) + "." + x.Sel.Name
	default:
		return ""
	}
}

func goCallHasVariable(call *ast.CallExpr) bool {
	found := false
	ast.Inspect(call, func(n ast.Node) bool {
		if _, ok := n.(*ast.Ident); ok {
			if id, ok := n.(*ast.Ident); ok && id.Name != "nil" && id.Name != "true" && id.Name != "false" {
				found = true
			}
		}
		return true
	})
	return found
}

func (g *Golang) collectSecrets(node ast.Node, fset *token.FileSet, result *model.FileAnalysis) {
	switch x := node.(type) {
	case *ast.AssignStmt:
		for i, lhs := range x.Rhs {
			if i >= len(x.Lhs) {
				break
			}
			id, ok := x.Lhs[i].(*ast.Ident)
			if !ok || !looksGoSecret(id.Name) {
				continue
			}
			if lit, ok := lhs.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				result.Secrets = append(result.Secrets, model.SecretInfo{Name: id.Name, Line: fset.Position(id.Pos()).Line})
			}
		}
	case *ast.ValueSpec:
		for i, name := range x.Names {
			if !looksGoSecret(name.Name) || i >= len(x.Values) {
				continue
			}
			if lit, ok := x.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				result.Secrets = append(result.Secrets, model.SecretInfo{Name: name.Name, Line: fset.Position(name.Pos()).Line})
			}
		}
	}
}

func looksGoSecret(name string) bool {
	lower := strings.ToLower(name)
	for _, hint := range []string{"password", "secret", "token", "apikey", "api_key"} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

func LanguageForExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".go":
		return "go"
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return "javascript"
	case ".py", ".pyw":
		return "python"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	default:
		return ""
	}
}

func LanguageForFilename(filename string) string {
	return LanguageForExtension(filepath.Ext(filename))
}
