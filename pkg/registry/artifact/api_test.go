package artifact_test

import (
	"errors"
	"testing"

	"github.com/MontFerret/specs/pkg/api"
	"github.com/MontFerret/specs/pkg/registry/artifact"
)

func TestAPICompatibilityAliasesAndForwards(t *testing.T) {
	var canonical *api.Reference = validAPIReference()
	var compatibility *artifact.APIReference = canonical

	if artifact.APISchemaV1 != api.SchemaV1 {
		t.Fatalf("compatibility schema ID = %q, want %q", artifact.APISchemaV1, api.SchemaV1)
	}

	if err := artifact.ValidateAPIReference(compatibility); err != nil {
		t.Fatalf("validate through compatibility forward: %v", err)
	}

	data := []byte(`{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[]}`)
	parsed, err := artifact.ParseAPIReference(data)
	if err != nil {
		t.Fatalf("parse through compatibility forward: %v", err)
	}

	if parsed.ID != "acme/archive" {
		t.Fatalf("parsed ID = %q", parsed.ID)
	}
}

func TestAPICompatibilityForwardsTranslateVersionError(t *testing.T) {
	data := []byte(`{"schemaVersion":2,"id":"acme/archive","version":"1.0.0","namespaces":[]}`)
	_, err := artifact.ParseAPIReference(data)
	requireArtifactVersionError(t, err)

	reference := validAPIReference()
	reference.SchemaVersion = 2
	requireArtifactVersionError(t, artifact.ValidateAPIReference(reference))
}

func requireArtifactVersionError(t *testing.T, err error) {
	t.Helper()

	var artifactErr *artifact.UnsupportedVersionError
	if !errors.As(err, &artifactErr) || artifactErr.Version != 2 {
		t.Fatalf("error = %T %v, want artifact UnsupportedVersionError for version 2", err, err)
	}

	var apiErr *api.UnsupportedVersionError
	if errors.As(err, &apiErr) {
		t.Fatalf("compatibility forward leaked canonical version error: %v", err)
	}
}
