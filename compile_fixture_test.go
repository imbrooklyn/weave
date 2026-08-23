package weave_test

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

const localGo127 = "/usr/local/golang/bin/go"

func TestGo127CompileFixtures(t *testing.T) {
	goTool := requireGo127(t)
	moduleRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}

	positive, err := filepath.Glob(filepath.Join(moduleRoot, "testdata", "compile", "positive", "*"))
	if err != nil {
		t.Fatalf("list positive compile fixtures: %v", err)
	}
	if len(positive) == 0 {
		t.Fatal("no positive compile fixtures found")
	}
	for _, fixture := range positive {
		fixture := fixture
		t.Run("positive/"+filepath.Base(fixture), func(t *testing.T) {
			output, compileErr := runCompileFixture(t, goTool, moduleRoot, fixture)
			if compileErr != nil {
				t.Fatalf("positive fixture did not compile:\n%s", output)
			}
		})
	}

	negative, err := filepath.Glob(filepath.Join(moduleRoot, "testdata", "compile", "negative", "*"))
	if err != nil {
		t.Fatalf("list negative compile fixtures: %v", err)
	}
	if len(negative) == 0 {
		t.Fatal("no negative compile fixtures found")
	}
	for _, fixture := range negative {
		fixture := fixture
		t.Run("negative/"+filepath.Base(fixture), func(t *testing.T) {
			output, compileErr := runCompileFixture(t, goTool, moduleRoot, fixture)
			if compileErr == nil {
				t.Fatalf("negative fixture compiled successfully:\n%s", output)
			}
			assertCompileDiagnostic(t, fixture, output)
		})
	}
}

func requireGo127(t *testing.T) string {
	t.Helper()
	candidates := make([]string, 0, 3)
	if configured := os.Getenv("WEAVE_GO127"); configured != "" {
		if !filepath.IsAbs(configured) {
			t.Fatalf("WEAVE_GO127 must be an absolute path, got %q", configured)
		}
		candidates = append(candidates, configured)
	}
	candidates = append(candidates, localGo127)
	if isGo127(runtime.Version()) {
		candidates = append(candidates, filepath.Join(runtime.GOROOT(), "bin", "go"))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, duplicate := seen[candidate]; duplicate {
			continue
		}
		seen[candidate] = struct{}{}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		command := exec.Command(candidate, "version")
		command.Env = compileEnvironment()
		output, err := command.CombinedOutput()
		if err != nil {
			continue
		}
		fields := strings.Fields(string(output))
		if len(fields) >= 3 && isGo127(fields[2]) {
			return candidate
		}
	}

	t.Fatalf(
		"Go 1.27 toolchain not found; checked explicit WEAVE_GO127, %s, and the Go 1.27 test runtime",
		localGo127,
	)
	return ""
}

func isGo127(version string) bool {
	return version == "go1.27" || strings.HasPrefix(version, "go1.27.")
}

func runCompileFixture(
	t *testing.T,
	goTool string,
	moduleRoot string,
	fixture string,
) (string, error) {
	t.Helper()
	directory := t.TempDir()
	goMod := fmt.Sprintf(`module example.com/weave-compile-fixture

go 1.27

require github.com/imbrooklyn/weave v0.0.0

replace github.com/imbrooklyn/weave => %s
`, strconv.Quote(moduleRoot))
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write fixture go.mod: %v", err)
	}

	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatalf("read fixture directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		source, err := os.ReadFile(filepath.Join(fixture, entry.Name()))
		if err != nil {
			t.Fatalf("read fixture source %s: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(directory, entry.Name()), source, 0o600); err != nil {
			t.Fatalf("stage fixture source %s: %v", entry.Name(), err)
		}
	}

	command := exec.Command(goTool, "test", "./...")
	command.Dir = directory
	command.Env = compileEnvironment()
	output, err := command.CombinedOutput()
	return string(output), err
}

func assertCompileDiagnostic(t *testing.T, fixture string, output string) {
	t.Helper()
	wantPath := filepath.Join(fixture, "diagnostic.want")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read expected diagnostic: %v", err)
	}
	wants := strings.Split(strings.TrimSpace(string(wantBytes)), "\n")
	if len(wants) < 2 || wants[0] == "" || wants[1] == "" {
		t.Fatalf("%s must contain an exact file:line:column locator and at least one message fragment", wantPath)
	}

	locator := "./" + wants[0] + ":"
	diagnostic := ""
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, locator) {
			diagnostic = line
			break
		}
	}
	if diagnostic == "" {
		t.Fatalf("compiler output does not contain expected location %q:\n%s", locator, output)
	}
	for _, fragment := range wants[1:] {
		if fragment != "" && !strings.Contains(diagnostic, fragment) {
			t.Fatalf("expected diagnostic does not contain %q:\n%s", fragment, diagnostic)
		}
	}
}

func compileEnvironment() []string {
	excluded := map[string]struct{}{
		"GOTOOLCHAIN": {},
		"GOWORK":      {},
	}
	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, skip := excluded[key]; !skip {
			environment = append(environment, entry)
		}
	}
	return append(environment, "GOTOOLCHAIN=local", "GOWORK=off")
}
