package main

import (
	"io"
	"os"

	"github.com/MontFerret/specs/pkg/module"
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
