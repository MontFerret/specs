package api

import (
	"fmt"
	"strings"
)

// ParseType parses one documentation type expression into its normalized
// recursive representation.
func ParseType(input string) (Type, error) {
	parsed, err := parseType(strings.TrimSpace(input))
	if err != nil {
		return Type{}, err
	}

	return *normalizeType(&parsed), nil
}

func parseType(input string) (Type, error) {
	if input == "" {
		return Type{}, fmt.Errorf("type expression must not be blank")
	}

	if strings.ContainsAny(input, "\r\n") {
		return Type{}, fmt.Errorf("type expression must be single-line")
	}

	parts, listClosing, err := splitTypeUnion(input)
	if err != nil {
		return Type{}, err
	}

	if len(parts) > 1 {
		members := make([]Type, 0, len(parts))
		for _, part := range parts {
			member, err := parseType(strings.TrimSpace(part))
			if err != nil {
				return Type{}, err
			}

			members = append(members, member)
		}

		return Type{Kind: TypeKindUnion, Types: members}, nil
	}

	if input[0] == '[' {
		if listClosing != len(input)-1 {
			return Type{}, fmt.Errorf("list type must end after its closing bracket")
		}

		element, err := parseType(strings.TrimSpace(input[1:listClosing]))
		if err != nil {
			return Type{}, fmt.Errorf("parse list element: %w", err)
		}

		return Type{Kind: TypeKindList, Element: &element}, nil
	}

	return Type{Kind: TypeKindNamed, Name: input}, nil
}

func splitTypeUnion(input string) ([]string, int, error) {
	parts := make([]string, 0, 1)
	start := 0
	stack := make([]rune, 0, 4)
	listClosing := -1

	for index, character := range input {
		switch character {
		case '(', '[', '{', '<':
			stack = append(stack, character)
		case ')', ']', '}', '>':
			if len(stack) == 0 || !matchingTypeGroup(stack[len(stack)-1], character) {
				return nil, -1, fmt.Errorf("unexpected closing delimiter %q", character)
			}

			stack = stack[:len(stack)-1]
			if index > 0 && input[0] == '[' && len(stack) == 0 && character == ']' {
				listClosing = index
			}
		case '|':
			if len(stack) == 0 {
				part := strings.TrimSpace(input[start:index])
				if part == "" {
					return nil, -1, fmt.Errorf("union member must not be blank")
				}

				parts = append(parts, part)
				start = index + 1
			}
		}
	}

	if len(stack) > 0 {
		return nil, -1, fmt.Errorf("unclosed delimiter %q", stack[len(stack)-1])
	}

	part := strings.TrimSpace(input[start:])
	if part == "" {
		return nil, -1, fmt.Errorf("union member must not be blank")
	}

	parts = append(parts, part)

	return parts, listClosing, nil
}

func matchingTypeGroup(opening, closing rune) bool {
	return opening == '(' && closing == ')' ||
		opening == '[' && closing == ']' ||
		opening == '{' && closing == '}' ||
		opening == '<' && closing == '>'
}

func normalizeReferenceTypes(reference *Reference) {
	for namespaceIndex := range reference.Namespaces {
		for functionIndex := range reference.Namespaces[namespaceIndex].Functions {
			function := &reference.Namespaces[namespaceIndex].Functions[functionIndex]
			for signatureIndex := range function.Signatures {
				signature := &function.Signatures[signatureIndex]
				for parameterIndex := range signature.Parameters {
					parameter := &signature.Parameters[parameterIndex]
					parameter.Type = normalizeType(parameter.Type)
				}

				if signature.Return != nil {
					signature.Return.Type = normalizeType(signature.Return.Type)
				}
			}
		}
	}
}

func normalizeType(value *Type) *Type {
	if value == nil {
		return nil
	}

	switch value.Kind {
	case TypeKindUnion:
		members := make([]Type, 0, len(value.Types))
		for index := range value.Types {
			member := normalizeType(&value.Types[index])
			if containsType(members, *member) {
				continue
			}

			members = append(members, *member)
		}

		if len(members) == 1 {
			return &members[0]
		}

		value.Types = members
	case TypeKindList:
		value.Element = normalizeType(value.Element)
	}

	return value
}

func containsType(values []Type, candidate Type) bool {
	for index := range values {
		if equalType(&values[index], &candidate) {
			return true
		}
	}

	return false
}

func equalType(left, right *Type) bool {
	if left == nil || right == nil {
		return left == right
	}

	if left.Kind != right.Kind || left.Name != right.Name || len(left.Types) != len(right.Types) {
		return false
	}

	if !equalType(left.Element, right.Element) {
		return false
	}

	for index := range left.Types {
		if !equalType(&left.Types[index], &right.Types[index]) {
			return false
		}
	}

	return true
}
