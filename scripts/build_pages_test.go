package scripts_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildPagesPublishesAPIIndexSchema(t *testing.T) {
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

	source, err := os.ReadFile(filepath.Join(repositoryRoot, "schemas", "registry", "artifact", "api-index", "v1.json"))
	if err != nil {
		t.Fatal(err)
	}

	published, err := os.ReadFile(filepath.Join(destination, "registry", "artifact", "api-index", "v1.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(published, source) {
		t.Fatal("published API Index schema differs from its canonical source")
	}
}
