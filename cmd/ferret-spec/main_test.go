package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MontFerret/specs/pkg/validation"
)

const fixtureRoot = "../../testdata/module-manifest"

func TestRunValidFilesAndStdinInArgumentOrder(t *testing.T) {
	jsonPath := fixturePath("valid", "minimal.json")
	yamlPath := fixturePath("valid", "minimal.yaml")
	stdin := mustReadFile(t, yamlPath)

	code, stdout, stderr := runCLI(
		t,
		[]string{"validate", "module", jsonPath, "-", yamlPath},
		stdin,
	)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d", exitSuccess, code)
	}
	wantStdout := jsonPath + ": valid\n<stdin>: valid\n" + yamlPath + ": valid\n"
	if stdout != wantStdout {
		t.Fatalf("unexpected stdout:\n%s\nwant:\n%s", stdout, wantStdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestRunReportsEverySemanticViolation(t *testing.T) {
	path := fixturePath("invalid", "multiple-violations.yaml")

	code, stdout, stderr := runCLI(t, []string{"validate", "module", path}, nil)

	if code != exitInvalid {
		t.Fatalf("expected exit code %d, got %d", exitInvalid, code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	want := strings.Join([]string{
		path + ":/compatibility/ferret [version-range] version must be a valid npm-compatible semantic version range",
		path + ":/dependencies/0/version [version-range] version must be a valid npm-compatible semantic version range",
		path + ":/dependencies/1/module [duplicate] dependency \"montferret/http\" is declared more than once",
		path + ":/license [spdx] license must be a valid SPDX license expression",
		"",
	}, "\n")
	if stderr != want {
		t.Fatalf("unexpected stderr:\n%s\nwant:\n%s", stderr, want)
	}
}

func TestRunReportsCanonicalLowercaseIdentityRequirement(t *testing.T) {
	input := strings.Replace(
		string(mustReadFile(t, fixturePath("valid", "minimal.yaml"))),
		"name: montferret/sqlite",
		"name: MontFerret/Archive",
		1,
	)

	code, stdout, stderr := runCLI(t, []string{"validate", "module", "-"}, []byte(input))
	if code != exitInvalid {
		t.Fatalf("expected exit code %d, got %d", exitInvalid, code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	want := "<stdin>:/name [pattern] module identity must use canonical lowercase owner/name spelling; each segment must start and end with a lowercase letter or digit\n"
	if stderr != want {
		t.Fatalf("unexpected stderr:\n%s\nwant:\n%s", stderr, want)
	}
}

func TestRunUsesDollarForDocumentRoot(t *testing.T) {
	code, stdout, stderr := runCLI(
		t,
		[]string{"validate", "module", "-"},
		[]byte("[unterminated"),
	)

	if code != exitInvalid {
		t.Fatalf("expected exit code %d, got %d", exitInvalid, code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !strings.HasPrefix(stderr, "<stdin>:$ [decode] ") {
		t.Fatalf("expected root decode diagnostic, got %q", stderr)
	}
}

func TestRunWritesExactValidJSONEnvelope(t *testing.T) {
	stdin := mustReadFile(t, fixturePath("valid", "minimal.yaml"))

	code, stdout, stderr := runCLI(
		t,
		[]string{"validate", "module", "--format", "json", "-"},
		stdin,
	)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d", exitSuccess, code)
	}
	want := "{\"formatVersion\":1,\"kind\":\"module\",\"status\":\"valid\",\"results\":[{\"source\":\"<stdin>\",\"status\":\"valid\"}]}\n"
	if stdout != want {
		t.Fatalf("unexpected JSON:\n%s\nwant:\n%s", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestRunWritesStructuredInvalidJSON(t *testing.T) {
	path := fixturePath("invalid", "multiple-violations.yaml")

	code, stdout, stderr := runCLI(
		t,
		[]string{"validate", "module", "--format=json", path},
		nil,
	)

	if code != exitInvalid {
		t.Fatalf("expected exit code %d, got %d", exitInvalid, code)
	}
	if stderr != "" {
		t.Fatalf("JSON validation wrote to stderr: %s", stderr)
	}

	var report validationReport
	if err := jsonUnmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	want := validationReport{
		FormatVersion: 1,
		Kind:          "module",
		Status:        statusInvalid,
		Results: []validationResult{{
			Source: path,
			Status: statusInvalid,
			Violations: []validation.Violation{
				{
					Path:    "/compatibility/ferret",
					Rule:    validation.RuleVersionRange,
					Message: "version must be a valid npm-compatible semantic version range",
				},
				{
					Path:    "/dependencies/0/version",
					Rule:    validation.RuleVersionRange,
					Message: "version must be a valid npm-compatible semantic version range",
				},
				{
					Path:    "/dependencies/1/module",
					Rule:    validation.RuleDuplicate,
					Message: "dependency \"montferret/http\" is declared more than once",
				},
				{
					Path:    "/license",
					Rule:    validation.RuleSPDX,
					Message: "license must be a valid SPDX license expression",
				},
			},
		}},
	}
	if !reflect.DeepEqual(report, want) {
		t.Fatalf("unexpected report:\n%#v\nwant:\n%#v", report, want)
	}
}

func TestOperationalErrorsTakePrecedenceAndDoNotStopValidation(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validated := make([]string, 0, 3)
	app := application{
		stdin:  strings.NewReader(""),
		stdout: &stdout,
		stderr: &stderr,
		readFile: func(path string) ([]byte, error) {
			return []byte(path), nil
		},
		validateModule: func(data []byte) error {
			value := string(data)
			validated = append(validated, value)
			switch value {
			case "invalid":
				return validation.NewErrors(validation.ScopeManifest, []validation.Violation{{
					Path:    "/version",
					Rule:    validation.RuleSemVer,
					Message: "bad version",
				}})
			case "broken":
				return errors.New("validator exploded")
			default:
				return nil
			}
		},
	}

	code := app.run([]string{
		"validate", "module", "--format", "json", "invalid", "broken", "valid",
	})

	if code != exitOperational {
		t.Fatalf("expected exit code %d, got %d", exitOperational, code)
	}
	if !reflect.DeepEqual(validated, []string{"invalid", "broken", "valid"}) {
		t.Fatalf("inputs were not all validated in order: %#v", validated)
	}
	want := "{\"formatVersion\":1,\"kind\":\"module\",\"status\":\"error\",\"results\":[" +
		"{\"source\":\"invalid\",\"status\":\"invalid\",\"violations\":[{\"path\":\"/version\",\"rule\":\"semver\",\"message\":\"bad version\"}]}," +
		"{\"source\":\"broken\",\"status\":\"error\",\"error\":\"validator exploded\"}," +
		"{\"source\":\"valid\",\"status\":\"valid\"}]}\n"
	if stdout.String() != want {
		t.Fatalf("unexpected JSON:\n%s\nwant:\n%s", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Fatalf("JSON validation wrote to stderr: %s", stderr.String())
	}
}

func TestReadErrorsAreOperational(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := application{
		stdin:  strings.NewReader(""),
		stdout: &stdout,
		stderr: &stderr,
		readFile: func(string) ([]byte, error) {
			return nil, errors.New("permission denied")
		},
		validateModule: func([]byte) error {
			t.Fatal("validator called after read failure")
			return nil
		},
	}

	code := app.run([]string{"validate", "module", "manifest.yaml"})

	if code != exitOperational {
		t.Fatalf("expected exit code %d, got %d", exitOperational, code)
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if stderr.String() != "manifest.yaml: error: permission denied\n" {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestDirectoryInputIsNotDiscovered(t *testing.T) {
	code, stdout, stderr := runCLI(
		t,
		[]string{"validate", "module", t.TempDir()},
		nil,
	)

	if code != exitOperational {
		t.Fatalf("expected exit code %d, got %d", exitOperational, code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !strings.Contains(stderr, ": error: ") {
		t.Fatalf("expected an operational error, got %q", stderr)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{name: "missing command", message: "missing command"},
		{name: "unknown command", args: []string{"check"}, message: `unknown command "check"`},
		{name: "missing kind", args: []string{"validate"}, message: "missing manifest kind"},
		{name: "unsupported kind", args: []string{"validate", "plugin"}, message: `unsupported manifest kind "plugin"`},
		{name: "missing input", args: []string{"validate", "module"}, message: "at least one manifest file is required"},
		{name: "repeated stdin", args: []string{"validate", "module", "-", "-"}, message: "stdin may be specified at most once"},
		{name: "unsupported format", args: []string{"validate", "module", "--format", "xml", "manifest.yaml"}, message: `unsupported format "xml"`},
		{name: "unknown flag", args: []string{"validate", "module", "--quiet", "manifest.yaml"}, message: "flag provided but not defined"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, test.args, nil)
			if code != exitOperational {
				t.Fatalf("expected exit code %d, got %d", exitOperational, code)
			}
			if stdout != "" {
				t.Fatalf("unexpected stdout: %s", stdout)
			}
			if !strings.Contains(stderr, "error: "+test.message) {
				t.Fatalf("missing error %q in %q", test.message, stderr)
			}
			if !strings.Contains(stderr, "Usage:") {
				t.Fatalf("missing usage in %q", stderr)
			}
		})
	}
}

func TestHelpAtEveryCommandLevel(t *testing.T) {
	tests := [][]string{
		{"--help"},
		{"validate", "--help"},
		{"validate", "module", "--help"},
	}

	for _, args := range tests {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, args, nil)
			if code != exitSuccess {
				t.Fatalf("expected exit code %d, got %d", exitSuccess, code)
			}
			if !strings.Contains(stdout, "ferret-spec validate module") {
				t.Fatalf("missing usage in stdout: %s", stdout)
			}
			if stderr != "" {
				t.Fatalf("unexpected stderr: %s", stderr)
			}
		})
	}
}

func TestStdinReadFailureIsOperational(t *testing.T) {
	code, stdout, stderr := runCLIWithReader(
		[]string{"validate", "module", "-"},
		failingReader{},
	)

	if code != exitOperational {
		t.Fatalf("expected exit code %d, got %d", exitOperational, code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if stderr != "<stdin>: error: read failed\n" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func runCLI(t *testing.T, args []string, stdin []byte) (int, string, string) {
	t.Helper()

	return runCLIWithReader(args, bytes.NewReader(stdin))
}

func runCLIWithReader(args []string, stdin io.Reader) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(args, stdin, &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func fixturePath(parts ...string) string {
	values := append([]string{fixtureRoot}, parts...)
	return filepath.Join(values...)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func jsonUnmarshal(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(value); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("report contains more than one JSON value")
	}

	return nil
}
