package schemas_test

import "fmt"

type rejectLoader struct{}

func (rejectLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("network loading disabled for %s", url)
}
