package scripts_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildPagesPublishesPortableSchemas(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	repositoryRoot := filepath.Dir(workingDirectory)
	destination := filepath.Join(t.TempDir(), "site")
	command := exec.Command("./scripts/build-pages.sh", destination)
	command.Dir = repositoryRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build Pages site: %v\n%s", err, output)
	}

	for _, relative := range []string{
		filepath.Join("registry", "artifact", "api-index", "v1.json"),
		filepath.Join("registry", "artifact", "api-catalog", "v1.json"),
	} {
		source, err := os.ReadFile(filepath.Join(repositoryRoot, "schemas", relative))
		if err != nil {
			t.Fatal(err)
		}

		published, err := os.ReadFile(filepath.Join(destination, relative))
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(published, source) {
			t.Fatalf("published schema %s differs from its canonical source", relative)
		}
	}
}
