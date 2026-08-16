package schemas_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	ferretschemas "github.com/MontFerret/specs/schemas"
)

func TestEveryEmbeddedSchemaCompilesOffline(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(rejectLoader{})

	ids := make([]string, 0)
	err := fs.WalkDir(ferretschemas.FS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := ferretschemas.FS.ReadFile(path)
		if err != nil {
			return err
		}

		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()

		var document map[string]any
		if err := decoder.Decode(&document); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}

		if document["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
			return fmt.Errorf("%s does not declare Draft 2020-12", path)
		}
		id, ok := document["$id"].(string)
		if !ok || !strings.HasPrefix(id, "https://schemas.ferretlang.org/") {
			return fmt.Errorf("%s has invalid canonical ID", path)
		}

		if err := compiler.AddResource(id, document); err != nil {
			return fmt.Errorf("register %s: %w", path, err)
		}

		ids = append(ids, id)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range ids {
		if _, err := compiler.Compile(id); err != nil {
			t.Errorf("compile %s: %v", id, err)
		}
	}
}

func TestReservedSchemasRejectEveryDocument(t *testing.T) {
	for _, path := range []string{"plugin/v1.json", "registry/v1.json"} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			schema := compileSingle(t, path)
			for _, document := range []any{nil, map[string]any{}, "value", []any{}} {
				if err := schema.Validate(document); err == nil {
					t.Fatalf("placeholder accepted %#v", document)
				}
			}
		})
	}
}

func TestRegistryArtifactSchemasAreEmbedded(t *testing.T) {
	for _, path := range []string{
		"registry/artifact/api-index/v1.json",
		"registry/artifact/api/v1.json",
		"registry/artifact/root/v1.json",
		"registry/artifact/module-index/v1.json",
		"registry/artifact/module/v1.json",
		"registry/artifact/version/v1.json",
		"registry/artifact/category-index/v1.json",
		"registry/artifact/category/v1.json",
		"registry/artifact/plugin-index/v1.json",
	} {
		if _, err := ferretschemas.FS.ReadFile(path); err != nil {
			t.Errorf("artifact schema %s is not embedded: %v", path, err)
		}
	}
}

func TestAPICatalogSchemaIsEmbedded(t *testing.T) {
	if _, err := ferretschemas.FS.ReadFile("registry/artifact/api-catalog/v1.json"); err != nil {
		t.Fatalf("API Catalog schema is not embedded: %v", err)
	}
}

func compileSingle(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()
	data, err := ferretschemas.FS.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	id := document["$id"].(string)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(rejectLoader{})
	if err := compiler.AddResource(id, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		t.Fatal(err)
	}

	return schema
}
