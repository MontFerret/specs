package artifact

import (
	"fmt"
)

// UnsupportedVersionError reports a positive artifact schema version other than v1.
type UnsupportedVersionError struct {
	Version int
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("unsupported registry artifact schema version %d", e.Version)
}
