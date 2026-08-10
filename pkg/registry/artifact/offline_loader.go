package artifact

import "fmt"

type offlineLoader struct{}

func (offlineLoader) Load(schemaURL string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", schemaURL)
}
