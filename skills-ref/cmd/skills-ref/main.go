package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/skills-repos/agentskills/skills-ref/internal/skill"
)

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage: skills-ref <command> [options] <path>...

Commands:
  validate         Check one or more skills
  read-properties  Print a skill's properties as JSON
  to-prompt        Print an <available_skills> XML block
  version          Print the version
  help [command]   Print this help or a command's help

Each <path> is a skill directory or its SKILL.md file.
Run 'skills-ref <command> --help' for the command's options.

Exit codes: 0 success, 1 invalid or unreadable skill, 2 usage error or missing path.
`

const validateHelp = `Usage: skills-ref validate [--format text|json] [--recursive] <path>...

Check each skill against the Agent Skills specification.

Options:
  --format text|json  Output format (default text). json prints
                      {"results": [{"path", "valid", "problems": [{"code", "field", "message"}]}]}.
  --recursive         Check every skill found below each path.

Exit codes: 0 all skills valid, 1 a skill is invalid or none were found, 2 usage error or missing path.
`

const readPropertiesHelp = `Usage: skills-ref read-properties [--format json] <path>

Print the skill's frontmatter properties as JSON.

Options:
  --format json  Output format (json is the only format).

Exit codes: 0 success, 1 the skill cannot be read, 2 usage error or missing path.
`

const toPromptHelp = `Usage: skills-ref to-prompt [--skip-invalid] <path>...

Print an <available_skills> XML block for the skills.

Options:
  --skip-invalid  Omit skills that fail validation, with a warning on stderr.

Exit codes: 0 success, 1 a skill cannot be read or none are valid, 2 usage error or missing path.
`

var commandHelp = map[string]string{
	"validate":        validateHelp,
	"read-properties": readPropertiesHelp,
	"to-prompt":       toPromptHelp,
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	case "help":
		switch {
		case len(args) == 1:
			fmt.Fprint(stdout, usage)
			return 0
		case len(args) > 2:
			return usageError(stderr, "", "help takes at most one command")
		case commandHelp[args[1]] == "":
			return usageError(stderr, "", "unknown command %q", args[1])
		}
		fmt.Fprint(stdout, commandHelp[args[1]])
		return 0
	case "--version", "version":
		v := version
		if build, ok := debug.ReadBuildInfo(); ok && v == "dev" && build.Main.Version != "" && build.Main.Version != "(devel)" {
			v = build.Main.Version
		}
		fmt.Fprintf(stdout, "skills-ref, version %s\n", v)
		return 0
	case "validate":
		return validate(args[1:], stdout, stderr)
	case "read-properties":
		return readProperties(args[1:], stdout, stderr)
	case "to-prompt":
		return toPrompt(args[1:], stdout, stderr)
	}
	return usageError(stderr, "", "unknown command %q", args[0])
}

// usageError reports a command-line mistake and returns exit code 2.
func usageError(stderr io.Writer, command, format string, args ...any) int {
	fmt.Fprintf(stderr, "Error: "+format+"\n", args...)
	if command != "" {
		command += " "
	}
	fmt.Fprintf(stderr, "Run 'skills-ref %s--help' for usage.\n", command)
	return 2
}

// parse accepts options before or after paths; arguments after "--" are
// always paths. It returns the paths, or false and the exit code when the
// command should stop.
func parse(set *flag.FlagSet, args []string, help string, stdout, stderr io.Writer) ([]string, int, bool) {
	set.SetOutput(io.Discard)
	var paths []string
	for {
		if err := set.Parse(args); errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, help)
			return nil, 0, false
		} else if err != nil {
			return nil, usageError(stderr, set.Name(), "%v", err), false
		}
		rest := set.Args()
		if consumed := len(args) - len(rest); len(rest) == 0 || consumed > 0 && args[consumed-1] == "--" {
			paths = append(paths, rest...)
			break
		}
		paths = append(paths, rest[0])
		args = rest[1:]
	}
	if len(paths) == 0 {
		return nil, usageError(stderr, set.Name(), "missing skill path"), false
	}
	// Paths the CLI derives use native separators, so given paths must too;
	// otherwise Windows output mixes / and \ depending on the command.
	for i, path := range paths {
		paths[i] = filepath.FromSlash(path)
	}
	return paths, 0, true
}

// checkExist returns exit code 2 if a path cannot be stat'ed.
func checkExist(command string, paths []string, stderr io.Writer) int {
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return usageError(stderr, command, "Path '%s' does not exist.", path)
		}
	}
	return 0
}

type validationResult struct {
	Path     string          `json:"path"`
	Valid    bool            `json:"valid"`
	Problems []skill.Problem `json:"problems"`
}

func validate(args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet("validate", flag.ContinueOnError)
	format := set.String("format", "text", "")
	recursive := set.Bool("recursive", false, "")
	paths, code, ok := parse(set, args, validateHelp, stdout, stderr)
	if !ok {
		return code
	}
	if *format != "text" && *format != "json" {
		return usageError(stderr, "validate", "--format must be text or json, not %q", *format)
	}
	if code := checkExist("validate", paths, stderr); code != 0 {
		return code
	}
	if *recursive {
		var err error
		if paths, err = findSkills(paths); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
	}
	results := []validationResult{}
	exit := 0
	for _, path := range paths {
		dir := skill.Directory(path)
		problems := skill.Validate(dir)
		if problems == nil {
			problems = []skill.Problem{}
		}
		results = append(results, validationResult{path, len(problems) == 0, problems})
		if len(problems) > 0 {
			exit = 1
		}
		if *format == "json" {
			continue
		}
		if len(problems) == 0 {
			fmt.Fprintf(stdout, "Valid skill: %s\n", dir)
			continue
		}
		fmt.Fprintf(stderr, "Validation failed for %s:\n", dir)
		for _, p := range problems {
			fmt.Fprintf(stderr, "  - %s\n", p.Message)
		}
	}
	if *format == "json" {
		_ = json.NewEncoder(stdout).Encode(struct {
			Results []validationResult `json:"results"`
		}{results})
	}
	return exit
}

// findSkills returns every directory at or below roots that contains a skill
// file, once each. A root may be a symlink; symlinks below a root are not
// followed.
func findSkills(roots []string) ([]string, error) {
	var found []string
	seen := make(map[string]bool)
	for _, root := range roots {
		root = filepath.Clean(skill.Directory(root))
		walkRoot := root
		if info, err := os.Lstat(root); err == nil && info.Mode()&fs.ModeSymlink != 0 {
			// WalkDir does not descend into a symlinked root; a trailing
			// separator makes the operating system resolve it.
			walkRoot += string(filepath.Separator)
		}
		count := 0
		err := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, err error) error {
			path = filepath.Clean(path)
			switch {
			case err != nil:
				return err
			case !entry.IsDir():
				return nil
			case path != root && (entry.Name() == ".git" || entry.Name() == ".venv" || entry.Name() == "node_modules"):
				return filepath.SkipDir
			case skill.FindSkillMD(path) != "":
				count++
				if !seen[path] {
					seen[path] = true
					found = append(found, path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("no skills found under %s", root)
		}
	}
	return found, nil
}

func readProperties(args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet("read-properties", flag.ContinueOnError)
	format := set.String("format", "json", "")
	paths, code, ok := parse(set, args, readPropertiesHelp, stdout, stderr)
	switch {
	case !ok:
		return code
	case *format != "json":
		return usageError(stderr, "read-properties", "--format must be json, not %q", *format)
	case len(paths) != 1:
		return usageError(stderr, "read-properties", "expected one skill path, got %d", len(paths))
	}
	if code := checkExist("read-properties", paths, stderr); code != 0 {
		return code
	}
	props, err := skill.ReadProperties(skill.Directory(paths[0]))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(props)
	return 0
}

func toPrompt(args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet("to-prompt", flag.ContinueOnError)
	skipInvalid := set.Bool("skip-invalid", false, "")
	paths, code, ok := parse(set, args, toPromptHelp, stdout, stderr)
	if !ok {
		return code
	}
	if code := checkExist("to-prompt", paths, stderr); code != 0 {
		return code
	}
	var dirs []string
	for _, path := range paths {
		dir := skill.Directory(path)
		if *skipInvalid {
			// A skill that passes Validate always passes ReadProperties.
			if problems := skill.Validate(dir); len(problems) > 0 {
				fmt.Fprintf(stderr, "warning: %s: %s\n", path, problems[0].Message)
				continue
			}
		}
		dirs = append(dirs, dir)
	}
	if len(dirs) == 0 {
		fmt.Fprintln(stderr, "Error: no valid skills")
		return 1
	}
	output, err := skill.ToPrompt(dirs)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, output)
	return 0
}
