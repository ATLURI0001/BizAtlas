package coords

import (
	"fmt"
	"strings"
)

// AssertHttpUrl rejects non-HTTP(S) schemes in scrape target URLs.
func AssertHttpUrl(raw string) error {
	if !strings.HasPrefix(raw, "https://") && !strings.HasPrefix(raw, "http://") {
		return fmt.Errorf("sentinel violation: unsupported URL scheme in %q", raw)
	}
	return nil
}
