package conformance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var root = filepath.Join("..", "..", "conformance")

// skillPath matches {skill} and the slash-separated path that follows it.
var skillPath = regexp.MustCompile(`\{skill\}(/[^/\s:'"<]+)*`)

// expand replaces each {skill} path in text with the native path, with the
// directory and separators passed through quote.
func expand(text, skill string, quote func(string) string) string {
	sep := quote(string(filepath.Separator))
	return skillPath.ReplaceAllStringFunc(text, func(match string) string {
		return quote(skill) + strings.ReplaceAll(strings.TrimPrefix(match, "{skill}"), "/", sep)
	})
}

func jsonQuote(s string) string {
	quoted, _ := json.Marshal(s)
	return strings.Trim(string(quoted), `"`)
}

type testCase struct {
	ID      string   `json:"id"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Expect  struct {
		ExitCode   int      `json:"exit_code"`
		Stdout     string   `json:"stdout"`
		Stderr     string   `json:"stderr"`
		StdoutJSON any      `json:"stdout_json"`
		Codes      []string `json:"problem_codes"`
	} `json:"expect"`
}

type validateOutput struct {
	Results []struct {
		Problems []struct {
			Code string `json:"code"`
		} `json:"problems"`
	} `json:"results"`
}

func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	schema, err := jsonschema.NewCompiler().Compile(filepath.Join(root, "schema", "validate.json"))
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestSchemaRejects(t *testing.T) {
	schema := compileSchema(t)
	for _, doc := range []string{
		`{"results": [{"path": "a", "valid": true, "problems": [{"code": "name.too_long", "message": "m"}]}]}`,
		`{"results": [{"path": "a", "valid": false, "problems": []}]}`,
		`{"results": [{"path": "a", "valid": false, "problems": [{"code": "Bad Code", "message": "m"}]}]}`,
		`{"results": [{"path": "a", "valid": true}]}`,
	} {
		value, err := jsonschema.UnmarshalJSON(strings.NewReader(doc))
		if err != nil {
			t.Fatal(err)
		}
		if schema.Validate(value) == nil {
			t.Errorf("schema accepted %s", doc)
		}
	}
}

func TestCases(t *testing.T) {
	schema := compileSchema(t)
	binary := filepath.Join(t.TempDir(), "skills-ref")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, "../../cmd/skills-ref").CombinedOutput(); err != nil {
		t.Fatalf("build: %s: %v", output, err)
	}
	files, err := filepath.Glob(filepath.Join(root, "cases", "*", "case.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("cases missing: %v", err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var tc testCase
		if err := json.Unmarshal(data, &tc); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		t.Run(tc.ID, func(t *testing.T) {
			skill, err := filepath.Abs(filepath.Join(filepath.Dir(file), "skill"))
			if err != nil {
				t.Fatal(err)
			}
			native := func(text string) string {
				return expand(text, skill, func(s string) string { return s })
			}
			args := []string{tc.Command}
			for _, arg := range tc.Args {
				args = append(args, native(arg))
			}
			cmd := exec.Command(binary, args...)
			var stdout, stderr strings.Builder
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			code := 0
			if err := cmd.Run(); err != nil {
				exit, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != tc.Expect.ExitCode {
				t.Errorf("exit code %d, want %d", code, tc.Expect.ExitCode)
			}
			if want := native(tc.Expect.Stderr); stderr.String() != want {
				t.Errorf("stderr:\n%q\nwant:\n%q", stderr.String(), want)
			}
			if tc.Expect.StdoutJSON == nil {
				if want := native(tc.Expect.Stdout); stdout.String() != want {
					t.Errorf("stdout:\n%q\nwant:\n%q", stdout.String(), want)
				}
				return
			}
			var got, want any
			if err := json.Unmarshal([]byte(stdout.String()), &got); err != nil {
				t.Fatalf("stdout is not JSON: %s: %v", stdout.String(), err)
			}
			var buf strings.Builder
			encoder := json.NewEncoder(&buf)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(tc.Expect.StdoutJSON); err != nil {
				t.Fatal(err)
			}
			// Substitute in the JSON text, escaped so Windows paths stay valid.
			wantText := expand(buf.String(), skill, jsonQuote)
			if err := json.Unmarshal([]byte(wantText), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("stdout JSON:\n%s\nwant:\n%s", stdout.String(), wantText)
			}
			if tc.Command != "validate" {
				return
			}
			value, err := jsonschema.UnmarshalJSON(strings.NewReader(stdout.String()))
			if err == nil {
				err = schema.Validate(value)
			}
			if err != nil {
				t.Errorf("stdout does not match schema/validate.json: %v", err)
			}
			if tc.Expect.Codes != nil {
				var output validateOutput
				if err := json.Unmarshal([]byte(stdout.String()), &output); err != nil {
					t.Fatal(err)
				}
				var codes []string
				for _, result := range output.Results {
					for _, p := range result.Problems {
						codes = append(codes, p.Code)
					}
				}
				if !slices.Equal(codes, tc.Expect.Codes) {
					t.Errorf("problem codes %v, want %v", codes, tc.Expect.Codes)
				}
			}
		})
	}
}
