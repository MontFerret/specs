package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/MontFerret/specs/pkg/validation"
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
		Source     string                 `json:"source"`
		Status     validationStatus       `json:"status"`
		Violations []validation.Violation `json:"violations,omitempty"`
		Error      string                 `json:"error,omitempty"`
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

func (report *validationReport) addResult(result validationResult) {
	report.Results = append(report.Results, result)

	if report.Status == statusError || result.Status == statusError {
		report.Status = statusError
		return
	}

	if report.Status == statusInvalid || result.Status == statusInvalid {
		report.Status = statusInvalid
		return
	}

	report.Status = statusValid
}

func (report validationReport) exitCode() int {
	switch report.Status {
	case statusValid:
		return exitSuccess
	case statusInvalid:
		return exitInvalid
	default:
		return exitOperational
	}
}

func (report validationReport) writeText(stdout, stderr io.Writer) error {
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

func (report validationReport) writeJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)

	return encoder.Encode(report)
}
