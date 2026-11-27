package coords

import "fmt"

// ValidatePaginationCursor enforces positive limit and non-negative offset.
func ValidatePaginationCursor(offset, limit int) error {
	if offset < 0 {
		return fmt.Errorf("contract violation: negative offset %d", offset)
	}
	if limit <= 0 || limit > 100 {
		return fmt.Errorf("contract violation: limit %d out of range [1, 100]", limit)
	}
	return nil
}
