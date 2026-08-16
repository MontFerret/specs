package catalog

import "fmt"

// UnsupportedVersionError reports a positive API Catalog schema version other than v1.
type UnsupportedVersionError struct {
	Version int
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("unsupported API Catalog schema version %d", e.Version)
}
