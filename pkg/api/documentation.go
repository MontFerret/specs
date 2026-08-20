package api

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"
)

const (
	parameterAnnotation  = "@param"
	returnAnnotation     = "@return"
	throwsAnnotation     = "@throws"
	deprecatedAnnotation = "@deprecated"
)

// ParseDocumentation parses normalized documentation-body text into ordinary
// prose and structured Ferret-facing metadata. Supported annotations must begin
// at the first character of their line. The returned value is fully validated.
func ParseDocumentation(text string) (Documentation, error) {
	metadata := Documentation{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	prose := make([]string, 0, len(lines))
	parameterNames := make(map[string]struct{})

	for index, line := range lines {
		tag, exists := supportedAnnotation(line)
		if !exists {
			prose = append(prose, line)

			continue
		}

		lineNumber := index + 1

		switch tag {
		case parameterAnnotation:
			parameter, err := parseParameterAnnotation(lineNumber, line)
			if err != nil {
				return Documentation{}, err
			}

			if _, exists := parameterNames[parameter.Name]; exists {
				return Documentation{}, documentationError(
					DocumentationErrorDuplicateParameter,
					lineNumber,
					line,
					fmt.Sprintf("parameter %q is declared more than once", parameter.Name),
				)
			}

			parameterNames[parameter.Name] = struct{}{}
			metadata.Parameters = append(metadata.Parameters, parameter)
		case returnAnnotation:
			if metadata.Return != nil {
				return Documentation{}, documentationError(
					DocumentationErrorMultipleReturns,
					lineNumber,
					line,
					"a declaration may contain at most one @return",
				)
			}

			value, description, err := parseTypedAnnotation(lineNumber, tag, line)
			if err != nil {
				return Documentation{}, err
			}

			parsedType, err := ParseType(value)
			if err != nil {
				return Documentation{}, invalidDocumentationType(lineNumber, line, err)
			}

			metadata.Return = &Return{Type: &parsedType, Description: description}
		case throwsAnnotation:
			value, description, err := parseTypedAnnotation(lineNumber, tag, line)
			if err != nil {
				return Documentation{}, err
			}

			metadata.Throws = append(metadata.Throws, Throw{Error: value, Description: description})
		case deprecatedAnnotation:
			if metadata.Deprecated != "" {
				return Documentation{}, documentationError(
					DocumentationErrorMultipleDeprecations,
					lineNumber,
					line,
					"a declaration may contain at most one @deprecated",
				)
			}

			description := strings.TrimSpace(strings.TrimPrefix(line, tag))
			if description == "" {
				return Documentation{}, documentationError(
					DocumentationErrorMalformedAnnotation,
					lineNumber,
					line,
					`expected "@deprecated <description>"`,
				)
			}

			metadata.Deprecated = description
		}
	}

	metadata.Description = documentationProse(prose)

	if metadata.Deprecated != "" {
		metadata.Description = removeStandardDeprecation(metadata.Description)
	}

	return metadata, nil
}

func parseParameterAnnotation(lineNumber int, annotation string) (Parameter, error) {
	rest := strings.TrimSpace(strings.TrimPrefix(annotation, parameterAnnotation))

	name, rest, ok := cutAnnotationToken(rest)
	if !ok || !validParameterName(name) || name == "_" {
		return Parameter{}, documentationError(
			DocumentationErrorMalformedAnnotation,
			lineNumber,
			annotation,
			`expected "@param <name> {<type>} <description>"`,
		)
	}

	value, description, reason := parseAnnotationValue(rest)
	if reason != "" {
		return Parameter{}, documentationError(
			DocumentationErrorMalformedAnnotation,
			lineNumber,
			annotation,
			fmt.Sprintf(`expected "@param <name> {<type>} <description>": %s`, reason),
		)
	}

	parsedType, err := ParseType(value)
	if err != nil {
		return Parameter{}, invalidDocumentationType(lineNumber, annotation, err)
	}

	return Parameter{Name: name, Type: &parsedType, Description: description}, nil
}

func parseTypedAnnotation(lineNumber int, tag, annotation string) (string, string, error) {
	rest := strings.TrimSpace(strings.TrimPrefix(annotation, tag))

	value, description, reason := parseAnnotationValue(rest)
	if reason != "" {
		expected := fmt.Sprintf(`expected "%s {<type>} <description>"`, tag)
		if tag == throwsAnnotation {
			expected = `expected "@throws {<error>} <description>"`
		}

		return "", "", documentationError(
			DocumentationErrorMalformedAnnotation,
			lineNumber,
			annotation,
			fmt.Sprintf("%s: %s", expected, reason),
		)
	}

	return value, description, nil
}

func parseAnnotationValue(value string) (string, string, string) {
	if value == "" || value[0] != '{' {
		return "", "", "annotation value must begin with an opening brace"
	}

	depth := 0
	closing := -1

	for index, character := range value {
		switch character {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				closing = index

				break
			}
		}

		if closing >= 0 {
			break
		}
	}

	if closing < 0 {
		return "", "", "annotation value is missing a closing brace"
	}

	typeExpression := value[1:closing]
	if strings.TrimSpace(typeExpression) == "" {
		return "", "", "annotation value must not be blank"
	}

	rest := value[closing+1:]
	if rest == "" || !unicode.IsSpace(rune(rest[0])) {
		return "", "", "annotation value must be followed by a description"
	}

	description := strings.TrimSpace(rest)
	if description == "" {
		return "", "", "annotation description must not be blank"
	}

	if description == "-" || strings.HasPrefix(description, "- ") {
		return "", "", "annotation description must not use a JSDoc '-' separator"
	}

	return typeExpression, description, ""
}

func supportedAnnotation(line string) (string, bool) {
	for _, tag := range []string{parameterAnnotation, returnAnnotation, throwsAnnotation, deprecatedAnnotation} {
		if line == tag {
			return tag, true
		}

		if strings.HasPrefix(line, tag) {
			rest := line[len(tag):]
			if rest != "" && unicode.IsSpace(rune(rest[0])) {
				return tag, true
			}
		}
	}

	return "", false
}

func cutAnnotationToken(value string) (string, string, bool) {
	index := strings.IndexFunc(value, unicode.IsSpace)
	if index <= 0 {
		return "", "", false
	}

	return value[:index], strings.TrimSpace(value[index:]), true
}

func validParameterName(value string) bool {
	if value == "" || !asciiLetter(value[0]) && value[0] != '_' {
		return false
	}

	for index := 1; index < len(value); index++ {
		if !asciiLetter(value[index]) && (value[index] < '0' || value[index] > '9') && value[index] != '_' {
			return false
		}
	}

	return true
}

func asciiLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func documentationProse(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	comments := make([]*ast.Comment, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			comments = append(comments, &ast.Comment{Text: "//"})

			continue
		}

		comments = append(comments, &ast.Comment{Text: "// " + line})
	}

	return strings.TrimSpace((&ast.CommentGroup{List: comments}).Text())
}

func removeStandardDeprecation(description string) string {
	paragraphs := strings.Split(description, "\n\n")
	kept := paragraphs[:0]

	for _, paragraph := range paragraphs {
		if strings.HasPrefix(strings.TrimSpace(paragraph), "Deprecated:") {
			continue
		}

		kept = append(kept, paragraph)
	}

	return strings.TrimSpace(strings.Join(kept, "\n\n"))
}

func documentationError(kind DocumentationErrorKind, line int, annotation, detail string) error {
	return &DocumentationError{
		Kind:       kind,
		Line:       line,
		Annotation: annotation,
		Detail:     detail,
	}
}

func invalidDocumentationType(line int, annotation string, err error) error {
	return documentationError(
		DocumentationErrorMalformedAnnotation,
		line,
		annotation,
		fmt.Sprintf("invalid type expression: %s", err),
	)
}
