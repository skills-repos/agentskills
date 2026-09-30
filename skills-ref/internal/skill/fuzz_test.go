package skill

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func fixtureInputs(f *testing.F) [][]byte {
	f.Helper()
	root := filepath.Join("..", "..", "conformance", "cases")
	var inputs [][]byte
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (entry.Name() != "SKILL.md" && entry.Name() != "skill.md") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err == nil {
			inputs = append(inputs, content)
		}
		return err
	})
	if err != nil || len(inputs) == 0 {
		f.Fatalf("load conformance seeds: %v (%d files)", err, len(inputs))
	}
	return inputs
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"---\nname: example\ndescription: test\n---\n",
		"---\nname: [bad]\n---",
		"---\nname: example\ndescription: a---b\n---",
		"\xff",
	} {
		f.Add([]byte(seed))
	}
	for _, input := range fixtureInputs(f) {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, content []byte) {
		if len(content) > 1<<20 {
			t.Skip()
		}
		parse(content)
	})
}

func FuzzValidate(f *testing.F) {
	for _, seed := range []string{
		"---\nname: example\ndescription: test\n---",
		"---\nname: wrong\ndescription: test\nmetadata:\n  x: y\n---",
	} {
		f.Add(seed)
	}
	for _, input := range fixtureInputs(f) {
		f.Add(string(input))
	}
	f.Fuzz(func(t *testing.T, content string) {
		if len(content) > 1<<20 {
			t.Skip()
		}
		dir := filepath.Join(t.TempDir(), "example")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		Validate(dir)
	})
}
