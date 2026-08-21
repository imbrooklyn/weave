package when_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestInvalidAPIShapesDoNotCompile(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantOutput []string
	}{
		{
			name: "Positive rejects a named string",
			source: `package compiletest

import "github.com/imbrooklyn/weave/when"

type Text string

var _ = when.Positive(Text("value"))
`,
			wantOutput: []string{"invalid.go:", "when.Number"},
		},
		{
			name: "ValidRange rejects time",
			source: `package compiletest

import (
	"time"

	"github.com/imbrooklyn/weave/when"
)

var _ = when.ValidRange(time.Time{}, time.Time{})
`,
			wantOutput: []string{"invalid.go:", "when.Number"},
		},
		{
			name: "ValidRange rejects mismatched bound types",
			source: `package compiletest

import "github.com/imbrooklyn/weave/when"

var lower int
var upper int64

var _ = when.ValidRange(lower, upper)
`,
			wantOutput: []string{"invalid.go:"},
		},
		{
			name: "NotZero rejects slices",
			source: `package compiletest

import "github.com/imbrooklyn/weave/when"

var _ = when.NotZero([]int{1})
`,
			wantOutput: []string{"invalid.go:", "comparable"},
		},
		{
			name: "NotEmpty rejects non-slices",
			source: `package compiletest

import "github.com/imbrooklyn/weave/when"

var _ = when.NotEmpty(1)
`,
			wantOutput: []string{"invalid.go:", "when.NotEmpty", "cannot infer E"},
		},
	}

	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, compileErr := compileFixture(t, moduleRoot, "invalid.go", test.source)
			if compileErr == nil {
				t.Fatalf("invalid API shape compiled successfully:\n%s", output)
			}
			for _, want := range test.wantOutput {
				if !strings.Contains(output, want) {
					t.Fatalf("compiler output does not contain %q:\n%s", want, output)
				}
			}
		})
	}
}

func compileFixture(t *testing.T, moduleRoot, filename, source string) (string, error) {
	t.Helper()
	directory := t.TempDir()
	goMod := fmt.Sprintf(`module example.com/weave-compile-test

go 1.27

require github.com/imbrooklyn/weave v0.0.0

replace github.com/imbrooklyn/weave => %s
`, strconv.Quote(moduleRoot))
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write fixture go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, filename), []byte(source), 0o600); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}

	command := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "test", "./...")
	command.Dir = directory
	command.Env = append(goEnvironmentWithout("GOTOOLCHAIN", "GOWORK"), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err := command.CombinedOutput()
	return string(output), err
}

func goEnvironmentWithout(keys ...string) []string {
	prefixes := make([]string, len(keys))
	for index, key := range keys {
		prefixes[index] = key + "="
	}

	environment := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		include := true
		for _, prefix := range prefixes {
			if strings.HasPrefix(entry, prefix) {
				include = false
				break
			}
		}
		if include {
			environment = append(environment, entry)
		}
	}
	return environment
}
