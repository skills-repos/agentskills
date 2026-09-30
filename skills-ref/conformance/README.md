# CLI conformance cases

Each `cases/<id>/case.json` describes one `skills-ref` invocation, and its
`skill/` directory holds the input files. `go test ./internal/conformance`
builds the binary and runs every case.

```json
{
  "id": "valid",
  "command": "validate",
  "args": ["{skill}/my-skill"],
  "expect": {"exit_code": 0, "stdout": "Valid skill: {skill}/my-skill\n"}
}
```

- `{skill}` in `args` or `expect` expands to the absolute path of `skill/`.
- `expect.exit_code` is required.
- `expect.stdout` and `expect.stderr` must match exactly; both default to `""`.
- `expect.stdout_json` replaces `stdout` with a comparison of decoded JSON.
  The stdout of `validate` with `stdout_json` must also match
  `schema/validate.json`.
- `expect.problem_codes` lists the problem codes `validate --format json`
  reports, in order.

Behavior that a portable fixture cannot express, such as a directory holding
both `SKILL.md` and `skill.md` on a case-insensitive filesystem, is tested in
`internal/skill/skill_test.go` instead.

The fuzz targets seed from every fixture `SKILL.md`:

```sh
go test ./internal/skill -run '^$' -fuzz '^FuzzParse$' -fuzztime=10m
go test ./internal/skill -run '^$' -fuzz '^FuzzValidate$' -fuzztime=10m
```
