package skill

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func problemCode(err error) string {
	var p *Problem
	if errors.As(err, &p) {
		return p.Code
	}
	return ""
}

func TestSplitFrontmatter(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		code              string
	}{
		{"valid", "---\r\nname: test\r\n---\r\nbody", "name: test\r", ""},
		{"bom", "\xef\xbb\xbf---\nname: test\n---\nbody", "name: test", ""},
		{"embedded", "---\ndescription: a---b\n---\nbody", "description: a---b", ""},
		{"missing", "# heading", "", "frontmatter.missing"},
		{"unclosed", "---\nname: test", "", "frontmatter.unclosed"},
		{"invalid utf8", string([]byte{0xff}), "", "frontmatter.invalid_utf8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, issue := splitFrontmatter([]byte(tc.input))
			if tc.code != "" {
				if problemCode(issue) != tc.code {
					t.Fatalf("got %v; want code %q", issue, tc.code)
				}
			} else if issue != nil || string(raw) != tc.want {
				t.Fatalf("got %q, %v; want %q", raw, issue, tc.want)
			}
		})
	}
}

func TestStrictYAML(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"flow-map", "---\nname: {a: b}\n---"},
		{"anchor", "---\nname: &alias test\n---"},
		{"duplicate-key", "---\nname: one\nname: two\n---"},
		{"tag", "---\nname: !!str test\n---"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, issue := parse([]byte(tc.input)); problemCode(issue) != "frontmatter.invalid_yaml" {
				t.Errorf("should reject %q, got %v", tc.input, issue)
			}
		})
	}
	for _, value := range []string{"yes", "~", "123"} {
		fields, issue := parse([]byte("---\nname: " + value + "\n---"))
		if issue != nil || fields["name"].Value != value {
			t.Errorf("scalar %q: %v, %v", value, fields, issue)
		}
	}
}

func TestValidateAndPrompt(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "café")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: cafe\u0301\ndescription: café & <test>\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if problems := Validate(dir); len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	props, issue := ReadProperties(dir)
	if issue != nil || props.Name != "café" {
		t.Fatalf("properties: %v, %v", props, issue)
	}
	prompt, issue := ToPrompt([]string{dir})
	if issue != nil || !strings.Contains(prompt, "café &amp; &lt;test&gt;") {
		t.Fatalf("prompt: %q, %v", prompt, issue)
	}
	if err := os.WriteFile(path, []byte("---\nname: cafe\u0301\ndescription: test\ncompatibility:\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	problems := Validate(dir)
	if len(problems) != 1 || problems[0].Code != "compatibility.empty" {
		t.Fatalf("expected compatibility.empty, got %v", problems)
	}
}

func TestValidationRules(t *testing.T) {
	for _, tc := range []struct {
		id, dir, yaml string
		codes         []string
	}{
		{"valid", "my-skill", "name: my-skill\ndescription: test", nil},
		{"allowed-tools", "my-skill", "name: my-skill\ndescription: test\nallowed-tools: Read", nil},
		{"missing-name", "my-skill", "description: test", []string{"name.missing"}},
		{"empty-name", "my-skill", "name:\ndescription: test", []string{"name.empty"}},
		{"missing-desc", "my-skill", "name: my-skill", []string{"description.missing"}},
		{"empty-desc", "my-skill", "name: my-skill\ndescription:", []string{"description.empty"}},
		{"uppercase", "MySkill", "name: MySkill\ndescription: test", []string{"name.not_lowercase"}},
		{"hyphens", "-x--", "name: -x--\ndescription: test", []string{"name.edge_hyphen", "name.double_hyphen"}},
		{"underscore", "my_skill", "name: my_skill\ndescription: test", []string{"name.invalid_chars"}},
		{"mismatch", "wrong", "name: right\ndescription: test", []string{"name.dir_mismatch"}},
		{"unexpected", "my-skill", "name: my-skill\ndescription: test\nother: value", []string{"field.unexpected"}},
		{"long-name", strings.Repeat("a", 65), "name: " + strings.Repeat("a", 65) + "\ndescription: test", []string{"name.too_long"}},
		{"long-desc", "my-skill", "name: my-skill\ndescription: " + strings.Repeat("a", 1025), []string{"description.too_long"}},
		{"compatibility", "my-skill", "name: my-skill\ndescription: test\ncompatibility: " + strings.Repeat("a", 501), []string{"compatibility.too_long"}},
		{"license", "my-skill", "name: my-skill\ndescription: test\nlicense:\n  nested: value", []string{"license.not_string"}},
		{"allowed", "my-skill", "name: my-skill\ndescription: test\nallowed-tools:\n  nested: value", []string{"allowed_tools.not_string"}},
		{"metadata", "my-skill", "name: my-skill\ndescription: test\nmetadata:\n  nested:\n    key: value", []string{"metadata.invalid"}},
		{"not-map", "my-skill", "- a\n- b", []string{"frontmatter.not_mapping"}},
	} {
		t.Run(tc.id, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), tc.dir)
			if err := os.Mkdir(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\n"+tc.yaml+"\n---\n"), 0644); err != nil {
				t.Fatal(err)
			}
			var codes []string
			for _, p := range Validate(dir) {
				if p.Code == "" {
					t.Errorf("problem without code: %v", p)
				}
				codes = append(codes, p.Code)
			}
			if !slices.Equal(codes, tc.codes) {
				t.Errorf("codes = %v, want %v", codes, tc.codes)
			}
		})
	}
}

func TestPathErrors(t *testing.T) {
	dir := t.TempDir()
	t.Run("not-found", func(t *testing.T) {
		if got := Validate(filepath.Join(dir, "absent")); len(got) != 1 || got[0].Code != "path.not_found" {
			t.Errorf("not found: %v", got)
		}
	})
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Run("not-directory", func(t *testing.T) {
		if got := Validate(file); len(got) != 1 || got[0].Code != "path.not_directory" {
			t.Errorf("not dir: %v", got)
		}
	})
	t.Run("below-file", func(t *testing.T) {
		path := filepath.Join(file, "sub")
		if got := Validate(path); len(got) != 1 || got[0].Code != "path.not_found" || got[0].Message != "Path does not exist: "+path {
			t.Errorf("below file: %v", got)
		}
	})
	t.Run("missing-file", func(t *testing.T) {
		if got := Validate(dir); len(got) != 1 || got[0].Code != "skill_md.missing" {
			t.Errorf("no skill: %v", got)
		}
	})
	t.Run("read-missing-file", func(t *testing.T) {
		if _, issue := ReadProperties(dir); problemCode(issue) != "skill_md.missing" {
			t.Errorf("read missing: %v", issue)
		}
	})
	skillFile := filepath.Join(dir, "SKILL.md")
	if err := os.Mkdir(skillFile, 0755); err != nil {
		t.Fatal(err)
	}
	if got := FindSkillMD(dir); got != "" {
		t.Errorf("directory accepted as skill file: %q", got)
	}
	if got := Directory(file); got != file {
		t.Errorf("ordinary file changed: %q", got)
	}
}

func TestReadProperties(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-skill")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("---\nname: my-skill\ndescription: test\nlicense: MIT\ncompatibility: Go\nallowed-tools: Read\nmetadata:\n  z: Z\n  a: A\n---\n")
	if problems := Validate(dir); len(problems) != 0 {
		t.Fatalf("all allowed fields should validate: %v", problems)
	}
	props, issue := ReadProperties(dir)
	if issue != nil || props.Location != filepath.Join(dir, "SKILL.md") {
		t.Fatalf("properties: %v, %q", issue, props.Location)
	}
	if got := Directory(props.Location); got != dir {
		t.Errorf("file arg: %q", got)
	}
	if *props.AllowedTools != "Read" || props.Metadata["z"] != "Z" || props.Metadata["a"] != "A" {
		t.Errorf("properties: %+v", props)
	}
	for _, tc := range []struct{ text, code string }{
		{"---\ndescription: test\n---", "name.missing"},
		{"---\nname: my-skill\n---", "description.missing"},
		{"---\nname: \ndescription: test\n---", "name.empty"},
		{"---\nname: my-skill\ndescription:\n---", "description.empty"},
		{"---\nname: my-skill\ndescription: test\nname: duplicate\n---", "frontmatter.invalid_yaml"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			write(tc.text)
			if _, issue := ReadProperties(dir); problemCode(issue) != tc.code {
				t.Errorf("input %q: issue = %v, want %s", tc.text, issue, tc.code)
			}
		})
	}
}

func TestFindSkillMD(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		if got := FindSkillMD(t.TempDir()); got != "" {
			t.Errorf("found %q in empty directory", got)
		}
	})
	t.Run("lowercase", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "skill.md")
		if err := os.WriteFile(path, []byte("---\nname: test\ndescription: test\n---"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := FindSkillMD(dir); got == "" || filepath.Base(got) != "SKILL.md" && filepath.Base(got) != "skill.md" {
			t.Errorf("lowercase file not found: %q", got)
		}
		if props, issue := ReadProperties(dir); issue != nil || props.Name != "test" {
			t.Errorf("lowercase file cannot be read: %v, %v", props, issue)
		}
	})
	t.Run("prefer uppercase", func(t *testing.T) {
		dir := t.TempDir()
		upper := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(upper, []byte("---\nname: upper\ndescription: test\n---"), 0644); err != nil {
			t.Fatal(err)
		}
		// On a case-insensitive filesystem the second path aliases the first.
		lower := filepath.Join(dir, "skill.md")
		if err := os.WriteFile(lower, []byte("---\nname: lower\ndescription: test\n---"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := FindSkillMD(dir); got != upper {
			t.Errorf("found %q, want %q", got, upper)
		}
		upperInfo, err := os.Stat(upper)
		if err != nil {
			t.Fatal(err)
		}
		lowerInfo, err := os.Stat(lower)
		if err != nil {
			t.Fatal(err)
		}
		want := "upper"
		if os.SameFile(upperInfo, lowerInfo) {
			want = "lower"
		}
		if props, issue := ReadProperties(dir); issue != nil || props.Name != want {
			t.Errorf("preferred file on this filesystem: %v, %v; want name %q", props, issue, want)
		}
	})
}

func TestToPromptEmpty(t *testing.T) {
	got, issue := ToPrompt(nil)
	if issue != nil || got != "<available_skills>\n</available_skills>" {
		t.Errorf("got %q, %v", got, issue)
	}
}
