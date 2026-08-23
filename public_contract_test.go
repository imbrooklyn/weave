package weave

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPublicGoDocAndEnglishSourceText(t *testing.T) {
	fileSet := token.NewFileSet()
	concurrencyDocs := make(map[string]string)
	processMarker := regexp.MustCompile(`(?i)\b(?:S[0-9]{2}(?:-[A-Z][0-9]{2})?|Codex|TODO|FIXME)\b`)
	packageDocs := make(map[string]bool)

	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.IsDir() {
			if path != "." && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		source, readError := os.ReadFile(path)
		if readError != nil {
			return readError
		}
		for offset, value := range source {
			if value >= 0x80 {
				t.Errorf("%s contains non-ASCII source text at byte %d", path, offset)
				break
			}
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if marker := processMarker.FindString(string(source)); marker != "" {
			t.Errorf("%s exposes internal process marker %q", path, marker)
		}

		parsed, parseError := parser.ParseFile(
			fileSet,
			path,
			source,
			parser.ParseComments,
		)
		if parseError != nil {
			return parseError
		}
		packagePath := filepath.Dir(path)
		if parsed.Doc != nil && strings.TrimSpace(parsed.Doc.Text()) != "" {
			packageDocs[packagePath] = true
		}
		auditExportedDeclarations(t, fileSet, path, parsed, concurrencyDocs)
		return nil
	})
	if err != nil {
		t.Fatalf("source audit failed: %v", err)
	}

	for _, packagePath := range []string{".", "when"} {
		if !packageDocs[packagePath] {
			t.Errorf("package %q has no package GoDoc", packagePath)
		}
	}
	wantPhrases := map[string]string{
		"Builder":   "not safe for concurrent use",
		"Compiler":  "safe for concurrent use",
		"Factory":   "safe for concurrent use",
		"Group":     "not safe for concurrent use",
		"Predicate": "safe for concurrent reads",
		"NodeView":  "safe for concurrent reads",
	}
	for typeName, phrase := range wantPhrases {
		doc := strings.Join(strings.Fields(concurrencyDocs[typeName]), " ")
		if !strings.Contains(doc, phrase) {
			t.Errorf("%s GoDoc does not contain %q: %q", typeName, phrase, doc)
		}
	}
}

func auditExportedDeclarations(
	t *testing.T,
	fileSet *token.FileSet,
	path string,
	file *ast.File,
	concurrencyDocs map[string]string,
) {
	t.Helper()
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if value.Name.IsExported() && !hasGoDoc(value.Doc) {
				t.Errorf("%s:%d exported function or method %s has no GoDoc", path, fileSet.Position(value.Pos()).Line, value.Name.Name)
			}
		case *ast.GenDecl:
			for _, specification := range value.Specs {
				switch spec := specification.(type) {
				case *ast.TypeSpec:
					if !spec.Name.IsExported() {
						continue
					}
					doc := selectGoDoc(spec.Doc, value.Doc)
					if strings.TrimSpace(doc) == "" {
						t.Errorf("%s:%d exported type %s has no GoDoc", path, fileSet.Position(spec.Pos()).Line, spec.Name.Name)
					}
					if filepath.Dir(path) == "." {
						concurrencyDocs[spec.Name.Name] = doc
					}
					auditExportedFields(t, fileSet, path, spec)
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						if name.IsExported() && strings.TrimSpace(selectGoDoc(spec.Doc, value.Doc)) == "" {
							t.Errorf("%s:%d exported value %s has no GoDoc", path, fileSet.Position(spec.Pos()).Line, name.Name)
						}
					}
				}
			}
		}
	}
}

func auditExportedFields(
	t *testing.T,
	fileSet *token.FileSet,
	path string,
	spec *ast.TypeSpec,
) {
	t.Helper()
	var fields *ast.FieldList
	switch value := spec.Type.(type) {
	case *ast.StructType:
		fields = value.Fields
	case *ast.InterfaceType:
		fields = value.Methods
	default:
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			if name.IsExported() && !hasGoDoc(field.Doc) && !hasGoDoc(field.Comment) {
				t.Errorf(
					"%s:%d exported field or interface method %s.%s has no GoDoc",
					path,
					fileSet.Position(field.Pos()).Line,
					spec.Name.Name,
					name.Name,
				)
			}
		}
	}
}

func hasGoDoc(group *ast.CommentGroup) bool {
	return group != nil && strings.TrimSpace(group.Text()) != ""
}

func selectGoDoc(primary, fallback *ast.CommentGroup) string {
	if hasGoDoc(primary) {
		return primary.Text()
	}
	if hasGoDoc(fallback) {
		return fallback.Text()
	}
	return ""
}
