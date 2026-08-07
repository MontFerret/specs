package module

import "fmt"

type offlineLoader struct{}

func (offlineLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", url)
}
