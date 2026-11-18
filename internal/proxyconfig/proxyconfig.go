// Package proxyconfig resolves proxy URLs from supported CLI inputs.
package proxyconfig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	// ErrConflict indicates that both supported proxy inputs were configured.
	ErrConflict = errors.New("-proxies and -proxies-file cannot be used together")
	// ErrEmptyFile indicates that a proxy file has no usable proxy URLs.
	ErrEmptyFile = errors.New("proxy file contains no proxy URLs")
	// ErrEmptyInline indicates that inline proxies contain no usable proxy URLs.
	ErrEmptyInline = errors.New("inline proxies contain no usable proxy URLs")
)

// Resolve returns proxy URLs from either an inline comma-separated value or a file.
func Resolve(inline, filePath string) ([]string, error) {
	inlineTrimmed := strings.TrimSpace(inline)
	filePathTrimmed := strings.TrimSpace(filePath)

	if inlineTrimmed != "" && filePathTrimmed != "" {
		return nil, ErrConflict
	}

	if inlineTrimmed != "" {
		parts := strings.Split(inlineTrimmed, ",")
		var proxies []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			proxies = append(proxies, trimmed)
		}
		if len(proxies) == 0 {
			return nil, ErrEmptyInline
		}
		return proxies, nil
	}

	if filePathTrimmed == "" {
		return nil, nil
	}

	file, err := os.Open(filePathTrimmed)
	if err != nil {
		return nil, fmt.Errorf("open proxy file %q: %w", filePathTrimmed, err)
	}
	defer file.Close()

	var proxies []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		proxies = append(proxies, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read proxy file %q: %w", filePathTrimmed, err)
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrEmptyFile, filePathTrimmed)
	}

	return proxies, nil
}
