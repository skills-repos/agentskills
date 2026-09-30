// Package skill reads and validates Agent Skills metadata.
package skill

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"
)

// Problem is one validation or parse failure. Code is stable across releases;
// Message is meant for people.
type Problem struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (p *Problem) Error() string { return p.Message }

// Properties is a skill's frontmatter. Optional fields are nil when absent.
type Properties struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       *string           `json:"license,omitempty"`
	Compatibility *string           `json:"compatibility,omitempty"`
	AllowedTools  *string           `json:"allowed-tools,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	// Location is the path of the SKILL.md file that was read.
	Location string `json:"-"`
}

type fields map[string]*yaml.Node

var allowedFields = []string{"allowed-tools", "compatibility", "description", "license", "metadata", "name"}

// FindSkillMD prefers SKILL.md but accepts skill.md as a fallback.
func FindSkillMD(dir string) string {
	for _, name := range []string{"SKILL.md", "skill.md"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

// Directory accepts a skill directory or an explicit SKILL.md file argument.
func Directory(path string) string {
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() &&
		strings.EqualFold(filepath.Base(path), "SKILL.md") {
		return filepath.Dir(path)
	}
	return path
}

func splitFrontmatter(content []byte) ([]byte, error) {
	if !utf8.Valid(content) {
		return nil, &Problem{"frontmatter.invalid_utf8", "", "SKILL.md must be UTF-8 encoded"}
	}
	lines := bytes.Split(bytes.TrimPrefix(content, []byte("\ufeff")), []byte("\n"))
	if strings.TrimSpace(string(lines[0])) != "---" {
		return nil, &Problem{"frontmatter.missing", "", "SKILL.md must start with YAML frontmatter (---)"}
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(string(lines[i])) == "---" {
			return bytes.Join(lines[1:i], []byte("\n")), nil
		}
	}
	return nil, &Problem{"frontmatter.unclosed", "", "SKILL.md frontmatter not properly closed with ---"}
}

// checkStrictYAML allows only plain block-style YAML: no flow style, tags,
// anchors, aliases or duplicate keys.
func checkStrictYAML(n *yaml.Node) error {
	if n.Style&(yaml.FlowStyle|yaml.TaggedStyle) != 0 || n.Kind == yaml.AliasNode || n.Anchor != "" {
		return fmt.Errorf("flow style, tags, anchors, and aliases are not supported")
	}
	if n.Kind == yaml.MappingNode {
		seen := make(map[string]bool)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i].Value
			if seen[key] {
				return fmt.Errorf("duplicate key %q", key)
			}
			seen[key] = true
		}
	}
	for _, child := range n.Content {
		if err := checkStrictYAML(child); err != nil {
			return err
		}
	}
	return nil
}

func parse(content []byte) (fields, error) {
	raw, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err = yaml.Unmarshal(raw, &doc); err == nil {
		err = checkStrictYAML(&doc)
	}
	if err != nil {
		return nil, &Problem{"frontmatter.invalid_yaml", "", "Invalid YAML in frontmatter: " + err.Error()}
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, &Problem{"frontmatter.not_mapping", "", "SKILL.md frontmatter must be a YAML mapping"}
	}
	mapping := doc.Content[0].Content
	result := make(fields, len(mapping)/2)
	for i := 0; i < len(mapping); i += 2 {
		if mapping[i].Kind != yaml.ScalarNode {
			return nil, &Problem{"frontmatter.invalid_yaml", "", "Invalid YAML in frontmatter: mapping keys must be strings"}
		}
		result[mapping[i].Value] = mapping[i+1]
	}
	return result, nil
}

func readFields(dir string) (fields, string, error) {
	path := FindSkillMD(dir)
	if path == "" {
		return nil, "", &Problem{"skill_md.missing", "", "SKILL.md not found in " + dir}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", &Problem{"skill_md.unreadable", "", err.Error()}
	}
	f, err := parse(data)
	return f, path, err
}

func (f fields) scalar(key string) (string, bool) {
	if n := f[key]; n != nil && n.Kind == yaml.ScalarNode {
		return n.Value, true
	}
	return "", false
}

// required returns the untrimmed value of a required non-empty string field.
func (f fields) required(key string) (string, *Problem) {
	if f[key] == nil {
		return "", &Problem{key + ".missing", key, "Missing required field in frontmatter: " + key}
	}
	value, ok := f.scalar(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", &Problem{key + ".empty", key, fmt.Sprintf("Field '%s' must be a non-empty string", key)}
	}
	return value, nil
}

// typeProblems reports optional fields that are present with the wrong YAML type.
func (f fields) typeProblems(keys ...string) []Problem {
	var problems []Problem
	for _, key := range keys {
		n := f[key]
		switch {
		case n == nil:
		case key == "metadata":
			valid := n.Kind == yaml.MappingNode
			for _, child := range n.Content {
				valid = valid && child.Kind == yaml.ScalarNode
			}
			if !valid {
				problems = append(problems, Problem{"metadata.invalid", key, "Field 'metadata' must be a mapping of string keys to string values"})
			}
		case n.Kind != yaml.ScalarNode:
			code := strings.ReplaceAll(key, "-", "_") + ".not_string"
			problems = append(problems, Problem{code, key, fmt.Sprintf("Field '%s' must be a string", key)})
		}
	}
	return problems
}

// ReadProperties reads a skill's frontmatter. It checks only that the fields
// it returns are present and well typed; use Validate for the full rules. The
// returned error is a *Problem.
func ReadProperties(dir string) (Properties, error) {
	f, path, err := readFields(dir)
	if err != nil {
		return Properties{}, err
	}
	name, p := f.required("name")
	if p == nil {
		_, p = f.required("description")
	}
	if p != nil {
		return Properties{}, p
	}
	if problems := f.typeProblems("license", "compatibility", "allowed-tools", "metadata"); len(problems) > 0 {
		return Properties{}, &problems[0]
	}
	optional := func(key string) *string {
		if value, ok := f.scalar(key); ok {
			return &value
		}
		return nil
	}
	desc, _ := f.scalar("description")
	props := Properties{
		Name:          norm.NFKC.String(strings.TrimSpace(name)),
		Description:   strings.TrimSpace(desc),
		License:       optional("license"),
		Compatibility: optional("compatibility"),
		AllowedTools:  optional("allowed-tools"),
		Location:      path,
	}
	if meta := f["metadata"]; meta != nil {
		props.Metadata = make(map[string]string)
		for i := 0; i < len(meta.Content); i += 2 {
			props.Metadata[meta.Content[i].Value] = meta.Content[i+1].Value
		}
	}
	return props, nil
}

// Validate checks a skill directory against the specification. It returns nil
// when the skill is valid.
func Validate(dir string) []Problem {
	info, err := os.Stat(dir)
	switch {
	// ENOTDIR means a parent of dir is a regular file, so dir does not exist.
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		return []Problem{{"path.not_found", "", "Path does not exist: " + dir}}
	case err != nil:
		return []Problem{{"path.unreadable", "", err.Error()}}
	case !info.IsDir():
		return []Problem{{"path.not_directory", "", "Not a directory: " + dir}}
	}
	if FindSkillMD(dir) == "" {
		return []Problem{{"skill_md.missing", "", "Missing required file: SKILL.md"}}
	}
	f, _, err := readFields(dir)
	if err != nil {
		return []Problem{*err.(*Problem)}
	}

	var problems []Problem
	var extras []string
	for key := range f {
		if !slices.Contains(allowedFields, key) {
			extras = append(extras, key)
		}
	}
	if len(extras) > 0 {
		slices.Sort(extras)
		problems = append(problems, Problem{"field.unexpected", "", fmt.Sprintf(
			"Unexpected fields in frontmatter: %s. Only ['%s'] are allowed.",
			strings.Join(extras, ", "), strings.Join(allowedFields, "', '"))})
	}
	if name, p := f.required("name"); p != nil {
		problems = append(problems, *p)
	} else {
		problems = append(problems, nameProblems(norm.NFKC.String(strings.TrimSpace(name)), filepath.Base(dir))...)
	}
	if desc, p := f.required("description"); p != nil {
		problems = append(problems, *p)
	} else if n := utf8.RuneCountInString(desc); n > 1024 {
		problems = append(problems, Problem{"description.too_long", "description", fmt.Sprintf("Description exceeds 1024 character limit (%d chars)", n)})
	}
	problems = append(problems, f.typeProblems("compatibility")...)
	if value, ok := f.scalar("compatibility"); ok {
		if n := utf8.RuneCountInString(value); n == 0 {
			problems = append(problems, Problem{"compatibility.empty", "compatibility", "Compatibility must be 1-500 characters"})
		} else if n > 500 {
			problems = append(problems, Problem{"compatibility.too_long", "compatibility", fmt.Sprintf("Compatibility exceeds 500 character limit (%d chars)", n)})
		}
	}
	return append(problems, f.typeProblems("license", "allowed-tools", "metadata")...)
}

func nameProblems(name, dirName string) []Problem {
	var problems []Problem
	add := func(code, message string) { problems = append(problems, Problem{code, "name", message}) }
	if n := utf8.RuneCountInString(name); n > 64 {
		add("name.too_long", fmt.Sprintf("Skill name '%s' exceeds 64 character limit (%d chars)", name, n))
	}
	if name != strings.ToLower(name) {
		add("name.not_lowercase", fmt.Sprintf("Skill name '%s' must be lowercase", name))
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		add("name.edge_hyphen", "Skill name cannot start or end with a hyphen")
	}
	if strings.Contains(name, "--") {
		add("name.double_hyphen", "Skill name cannot contain consecutive hyphens")
	}
	if strings.ContainsFunc(name, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-' }) {
		add("name.invalid_chars", fmt.Sprintf("Skill name '%s' contains invalid characters. Only letters, digits, and hyphens are allowed.", name))
	}
	if norm.NFKC.String(dirName) != name {
		add("name.dir_mismatch", fmt.Sprintf("Directory name '%s' must match skill name '%s'", dirName, name))
	}
	return problems
}

// ToPrompt returns an <available_skills> XML block for the skill directories.
// The returned error is a *Problem when a skill cannot be read.
func ToPrompt(dirs []string) (string, error) {
	lines := []string{"<available_skills>"}
	for _, dir := range dirs {
		// Resolve the directory rather than SKILL.md itself, so a symlinked
		// SKILL.md is reported inside the skill directory it belongs to.
		dir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
		props, err := ReadProperties(dir)
		if err != nil {
			return "", err
		}
		lines = append(lines, "<skill>",
			"<name>", html.EscapeString(props.Name), "</name>",
			"<description>", html.EscapeString(props.Description), "</description>",
			"<location>", html.EscapeString(props.Location), "</location>",
			"</skill>")
	}
	lines = append(lines, "</available_skills>")
	return strings.Join(lines, "\n"), nil
}
