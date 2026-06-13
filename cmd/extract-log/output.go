package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var nonSafeRegex = regexp.MustCompile(`[^A-Za-z0-9_.\-]`)

// writeExchanges writes one JSON file per exchange into outputDir
func writeExchanges(exchanges []Exchange, outputDir string) (int, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return 0, err
	}

	padWidth := max(len(strconv.Itoa(len(exchanges))), 3)
	written := 0

	for n, ex := range exchanges {
		seq := fmt.Sprintf("%0*d", padWidth, n+1)
		name := seq + "_" + safeName(ex.URL) + ".json"

		data, err := marshalJSON(ex)
		if err != nil {
			return written, err
		}

		if err := os.WriteFile(filepath.Join(outputDir, name), data, 0o644); err != nil {
			return written, err
		}

		written++
	}

	return written, nil
}

// marshalJSON returns an indented JSON without escaping HTML characters
func marshalJSON(v any) ([]byte, error) {
	var compact bytes.Buffer

	enc := json.NewEncoder(&compact)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, bytes.TrimRight(compact.Bytes(), "\n"), "", "  "); err != nil {
		return nil, err
	}

	pretty.WriteByte('\n')

	return pretty.Bytes(), nil
}

// safeName turns a string into a filename-safe base name
func safeName(u string) string {
	path := u
	if parsed, err := url.Parse(u); err == nil && parsed.Path != "" {
		path = parsed.Path
	}

	path = strings.SplitN(path, "?", 2)[0]
	path = strings.Trim(path, "/")
	if path == "" {
		path = "root"
	}

	name := strings.ReplaceAll(path, "/", "_")

	return nonSafeRegex.ReplaceAllString(name, "_")
}
