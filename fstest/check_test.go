//go:build !remote

package fstest

import (
	"testing"

	errname "github.com/Antonboom/errname/pkg/analyzer"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"

	"lesiw.io/checker"
	"lesiw.io/linelen"
)

func TestCheck(t *testing.T) {
	checker.Run(t,
		atomicalign.Analyzer,
		errname.New(),
		linelen.Analyzer,
		loopclosure.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		unusedwrite.Analyzer,
	)
}
