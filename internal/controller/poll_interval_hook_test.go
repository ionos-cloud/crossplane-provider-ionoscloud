package controller

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mocktest is an envtest harness rather than a production controller. It runs with a
// poll interval of its own and must stay deterministic, so it does not use the hook.
const pollIntervalHookExemptDir = "compute/mocktest"

// TestEveryReconcilerUsesThePollIntervalHook guards against a controller being added
// without the provider's poll interval strategy, which would leave that resource kind
// reconciling in lockstep with every other resource of its kind.
func TestEveryReconcilerUsesThePollIntervalHook(t *testing.T) {
	reconcilers := 0

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(path), pollIntervalHookExemptDir+"/") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		require.NoError(t, err, "cannot parse %s", path)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isSelectorCall(call.Fun, "managed", "NewReconciler") {
				return true
			}
			reconcilers++
			assert.True(t, hasPollIntervalHook(call.Args),
				"%s: managed.NewReconciler must be passed managed.WithPollIntervalHook(opts.PollIntervalHook())", path)
			return true
		})
		return nil
	})
	require.NoError(t, err)
	assert.NotZero(t, reconcilers, "found no managed.NewReconciler call to check")
}

func hasPollIntervalHook(args []ast.Expr) bool {
	for _, arg := range args {
		if call, ok := arg.(*ast.CallExpr); ok && isSelectorCall(call.Fun, "managed", "WithPollIntervalHook") {
			return true
		}
	}
	return false
}

func isSelectorCall(fun ast.Expr, pkg, name string) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == pkg
}
