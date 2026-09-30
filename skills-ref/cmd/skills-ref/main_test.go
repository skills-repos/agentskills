package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errOut strings.Builder
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{
		{"--help"}, {"-h"},
		{"validate", "--help"}, {"validate", "-h"},
		{"read-properties", "--help"},
		{"to-prompt", "--help"},
	} {
		code, stdout, stderr := runCLI(args...)
		if code != 0 || stderr != "" || !strings.HasPrefix(stdout, "Usage: skills-ref ") || !strings.Contains(stdout, "Exit codes: 0 ") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", args, code, stdout, stderr)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		args   []string
		stderr string
	}{
		{nil, "Usage: skills-ref "},
		{[]string{"lint"}, "Error: unknown command \"lint\"\nRun 'skills-ref --help' for usage.\n"},
		{[]string{"validate"}, "Error: missing skill path\nRun 'skills-ref validate --help' for usage.\n"},
		{[]string{"validate", "--nope", "x"}, "Error: flag provided but not defined: -nope\n"},
		{[]string{"validate", "--format", "xml", "x"}, "Error: --format must be text or json, not \"xml\"\n"},
		{[]string{"read-properties", "--format", "text", "x"}, "Error: --format must be json, not \"text\"\n"},
		{[]string{"read-properties", "a", "b"}, "Error: expected one skill path, got 2\n"},
		{[]string{"to-prompt"}, "Error: missing skill path\n"},
	} {
		code, stdout, stderr := runCLI(tc.args...)
		if code != 2 || stdout != "" || !strings.HasPrefix(stderr, tc.stderr) {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 2, stderr prefix %q", tc.args, code, stdout, stderr, tc.stderr)
		}
	}
}

func TestOptionsAfterPaths(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "demo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: demo\ndescription: Demo\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runCLI("validate", dir, "--format", "json")
	if code != 0 || stderr != "" || !strings.HasPrefix(stdout, `{"results":[{"path":`) {
		t.Errorf("option after path: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	code, stdout, _ = runCLI("validate", root, "--recursive", "--format=json")
	if code != 0 || strings.Count(stdout, `"valid":true`) != 1 {
		t.Errorf("options after path: exit %d, stdout %q", code, stdout)
	}
	// After "--", an argument that looks like an option is a path.
	code, _, stderr = runCLI("validate", dir, "--", "--format")
	if code != 2 || !strings.HasPrefix(stderr, "Error: Path '--format' does not exist.\n") {
		t.Errorf("after --: exit %d, stderr %q", code, stderr)
	}
}

func TestMissingPath(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "demo"))
	file := filepath.Join(root, "demo", "SKILL.md")
	for _, missing := range []string{filepath.Join(root, "absent"), filepath.Join(file, "below")} {
		for _, args := range [][]string{
			{"validate", missing},
			{"validate", "--format", "json", filepath.Join(root, "demo"), missing},
			{"validate", "--recursive", missing},
			{"read-properties", missing},
			{"to-prompt", filepath.Join(root, "demo"), missing},
			{"to-prompt", "--skip-invalid", missing},
		} {
			code, stdout, stderr := runCLI(args...)
			want := "Error: Path '" + missing + "' does not exist.\nRun 'skills-ref " + args[0] + " --help' for usage.\n"
			if code != 2 || stdout != "" || stderr != want {
				t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 2, stderr %q", args, code, stdout, stderr, want)
			}
		}
	}
}

func TestPromptSymlinkedSkillFile(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "store", "demo"))
	dir := filepath.Join(root, "demo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "store", "demo", "SKILL.md"), filepath.Join(dir, "SKILL.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	code, stdout, stderr := runCLI("to-prompt", dir)
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := "<location>\n" + filepath.Join(resolved, "SKILL.md") + "\n</location>"
	if code != 0 || stderr != "" || !strings.Contains(stdout, want) {
		t.Errorf("exit %d, stdout %q, stderr %q; want location %q", code, stdout, stderr, want)
	}
}

func TestHelpCommand(t *testing.T) {
	for args, want := range map[string]string{
		"help":                 usage,
		"help validate":        validateHelp,
		"help read-properties": readPropertiesHelp,
		"help to-prompt":       toPromptHelp,
	} {
		if code, stdout, stderr := runCLI(strings.Fields(args)...); code != 0 || stdout != want || stderr != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", args, code, stdout, stderr)
		}
	}
	for args, want := range map[string]string{
		"help lint":         "Error: unknown command \"lint\"\n",
		"help validate foo": "Error: help takes at most one command\n",
	} {
		if code, stdout, stderr := runCLI(strings.Fields(args)...); code != 2 || stdout != "" || !strings.HasPrefix(stderr, want) {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", args, code, stdout, stderr)
		}
	}
}

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + filepath.Base(dir) + "\ndescription: Demo\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "skills", "demo"))
	link := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "skills"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// A symlink below the root is not followed.
	if err := os.Symlink(filepath.Join(root, "skills", "demo"), filepath.Join(root, "skills", "alias")); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("validate", "--recursive", link)
	want := "Valid skill: " + filepath.Join(link, "demo") + "\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want %q", code, stdout, stderr, want)
	}
}

func TestRecursiveDeduplicates(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "demo"))
	writeSkill(t, filepath.Join(root, "demo", "nested"))
	code, stdout, stderr := runCLI("validate", "--recursive", root, filepath.Join(root, "demo")+string(filepath.Separator), filepath.Join(root, "demo", "SKILL.md"))
	want := "Valid skill: " + filepath.Join(root, "demo") + "\nValid skill: " + filepath.Join(root, "demo", "nested") + "\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want %q", code, stdout, stderr, want)
	}
}

func TestReadPropertiesJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: demo\ndescription: café & <b>\nlicense: MIT\nmetadata:\n  z: Z\n  a: A\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("read-properties", dir)
	want := "{\n  \"name\": \"demo\",\n  \"description\": \"café & <b>\",\n  \"license\": \"MIT\",\n  \"metadata\": {\n    \"a\": \"A\",\n    \"z\": \"Z\"\n  }\n}\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want %q", code, stdout, stderr, want)
	}
}

func TestVersion(t *testing.T) {
	for _, arg := range []string{"--version", "version"} {
		if code, stdout, _ := runCLI(arg); code != 0 || stdout != "skills-ref, version dev\n" {
			t.Errorf("%s: exit %d, stdout %q", arg, code, stdout)
		}
	}
}
