package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/MontFerret/specs/pkg/module"
)

type (
	validationStatus string

	validationReport struct {
		FormatVersion int                `json:"formatVersion"`
		Kind          string             `json:"kind"`
		Status        validationStatus   `json:"status"`
		Results       []validationResult `json:"results"`
	}

	validationResult struct {
		Source     string             `json:"source"`
		Status     validationStatus   `json:"status"`
		Violations []module.Violation `json:"violations,omitempty"`
		Error      string             `json:"error,omitempty"`
	}

	application struct {
		stdin          io.Reader
		stdout         io.Writer
		stderr         io.Writer
		readFile       func(string) ([]byte, error)
		validateModule func([]byte) error
	}
)

const (
	exitSuccess     = 0
	exitInvalid     = 1
	exitOperational = 2
)

const (
	statusValid   validationStatus = "valid"
	statusInvalid validationStatus = "invalid"
	statusError   validationStatus = "error"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	app := application{
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		readFile: os.ReadFile,
		validateModule: func(data []byte) error {
			_, err := module.Parse(data)
			return err
		},
	}

	return app.run(args)
}

func (app application) run(args []string) int {
	if len(args) == 0 {
		return app.usageError("missing command", writeRootUsage)
	}

	switch args[0] {
	case "-h", "--help", "help":
		writeRootUsage(app.stdout)
		return exitSuccess
	case "validate":
		return app.runValidate(args[1:])
	default:
		return app.usageError(fmt.Sprintf("unknown command %q", args[0]), writeRootUsage)
	}
}

func (app application) runValidate(args []string) int {
	if len(args) == 0 {
		return app.usageError("missing manifest kind", writeValidateUsage)
	}

	switch args[0] {
	case "-h", "--help", "help":
		writeValidateUsage(app.stdout)
		return exitSuccess
	case "module":
		return app.runValidateModule(args[1:])
	default:
		return app.usageError(fmt.Sprintf("unsupported manifest kind %q", args[0]), writeValidateUsage)
	}
}

func (app application) runValidateModule(args []string) int {
	flags := flag.NewFlagSet("validate module", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	format := flags.String("format", "text", "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeModuleUsage(app.stdout)
			return exitSuccess
		}

		return app.usageError(err.Error(), writeModuleUsage)
	}

	if *format != "text" && *format != "json" {
		return app.usageError(fmt.Sprintf("unsupported format %q", *format), writeModuleUsage)
	}

	sources := flags.Args()
	if len(sources) == 0 {
		return app.usageError("at least one manifest file is required", writeModuleUsage)
	}

	stdinCount := 0
	for _, source := range sources {
		if source == "-" {
			stdinCount++
		}
	}

	if stdinCount > 1 {
		return app.usageError("stdin may be specified at most once", writeModuleUsage)
	}

	report := app.validateSources(sources)
	var err error
	if *format == "json" {
		err = writeJSONReport(app.stdout, report)
	} else {
		err = writeTextReport(app.stdout, app.stderr, report)
	}

	if err != nil {
		_, _ = fmt.Fprintf(app.stderr, "error: write output: %v\n", err)
		return exitOperational
	}

	return report.Status.exitCode()
}

func (app application) validateSources(sources []string) validationReport {
	report := validationReport{
		FormatVersion: 1,
		Kind:          "module",
		Status:        statusValid,
		Results:       make([]validationResult, 0, len(sources)),
	}

	for _, source := range sources {
		result := app.validateSource(source)
		report.Results = append(report.Results, result)
		report.Status = aggregateStatus(report.Status, result.Status)
	}

	return report
}

func (app application) validateSource(source string) validationResult {
	label := source
	var (
		data []byte
		err  error
	)

	if source == "-" {
		label = "<stdin>"
		data, err = io.ReadAll(app.stdin)
	} else {
		data, err = app.readFile(source)
	}

	if err != nil {
		return validationResult{
			Source: label,
			Status: statusError,
			Error:  err.Error(),
		}
	}

	err = app.validateModule(data)
	if err == nil {
		return validationResult{
			Source: label,
			Status: statusValid,
		}
	}

	var validationErr *module.ValidationErrors
	if errors.As(err, &validationErr) {
		return validationResult{
			Source:     label,
			Status:     statusInvalid,
			Violations: validationErr.Violations,
		}
	}

	return validationResult{
		Source: label,
		Status: statusError,
		Error:  err.Error(),
	}
}

func aggregateStatus(current, next validationStatus) validationStatus {
	if current == statusError || next == statusError {
		return statusError
	}

	if current == statusInvalid || next == statusInvalid {
		return statusInvalid
	}

	return statusValid
}

func (status validationStatus) exitCode() int {
	switch status {
	case statusValid:
		return exitSuccess
	case statusInvalid:
		return exitInvalid
	default:
		return exitOperational
	}
}

func writeTextReport(stdout, stderr io.Writer, report validationReport) error {
	for _, result := range report.Results {
		switch result.Status {
		case statusValid:
			if _, err := fmt.Fprintf(stdout, "%s: valid\n", result.Source); err != nil {
				return err
			}
		case statusInvalid:
			for _, violation := range result.Violations {
				path := violation.Path
				if path == "" {
					path = "$"
				}
				if _, err := fmt.Fprintf(
					stderr,
					"%s:%s [%s] %s\n",
					result.Source,
					path,
					violation.Rule,
					violation.Message,
				); err != nil {
					return err
				}
			}
		case statusError:
			if _, err := fmt.Fprintf(stderr, "%s: error: %s\n", result.Source, result.Error); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeJSONReport(writer io.Writer, report validationReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)

	return encoder.Encode(report)
}

func (app application) usageError(message string, usage func(io.Writer)) int {
	_, _ = fmt.Fprintf(app.stderr, "error: %s\n\n", message)
	usage(app.stderr)

	return exitOperational
}

func writeRootUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `ferret-spec validates Ferret specification documents.

Usage:
  ferret-spec validate module [--format text|json] FILE...`)
}

func writeValidateUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `Usage:
  ferret-spec validate module [--format text|json] FILE...`)
}

func writeModuleUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `Usage:
  ferret-spec validate module [--format text|json] FILE...

Arguments:
  FILE    JSON or YAML module manifest; use - to read stdin

Options:
  --format text|json    Output format (default: text)`)
}
