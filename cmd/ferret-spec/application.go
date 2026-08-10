package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/MontFerret/specs/pkg/validation"
)

type application struct {
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	readFile       func(string) ([]byte, error)
	validateModule func([]byte) error
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
		err = report.writeJSON(app.stdout)
	} else {
		err = report.writeText(app.stdout, app.stderr)
	}

	if err != nil {
		_, _ = fmt.Fprintf(app.stderr, "error: write output: %v\n", err)
		return exitOperational
	}

	return report.exitCode()
}

func (app application) validateSources(sources []string) validationReport {
	report := validationReport{
		FormatVersion: 1,
		Kind:          "module",
		Status:        statusValid,
		Results:       make([]validationResult, 0, len(sources)),
	}

	for _, source := range sources {
		report.addResult(app.validateSource(source))
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

	var validationErr *validation.Errors
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

func (app application) usageError(message string, usage func(io.Writer)) int {
	_, _ = fmt.Fprintf(app.stderr, "error: %s\n\n", message)
	usage(app.stderr)

	return exitOperational
}
