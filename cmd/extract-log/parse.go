package main

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

type Exchange struct {
	Method    string  `json:"method"`
	URL       string  `json:"url"`
	Timestamp string  `json:"timestamp"`
	Request   Message `json:"request"`
	Response  Message `json:"response"`
}

type Message struct {
	Status  string            `json:"status,omitempty"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

var (
	// ANSI color codes (ESC [ ... m)
	ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*m")

	// Patterns for finding request/response blocks
	reqStartRegex = regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2}\.\d+)\]\s*\[DEBUG\]\s+(GET|POST|PUT|DELETE|HEAD|PATCH|OPTIONS)\s+(/\S*)\s*$`)
	logLineRegex  = regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\.\d+\]`)
	responseRegex = regexp.MustCompile(`^\[Response:\s*(\d+)\]\s*$`)
	headerRegex   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*:\s`)

	// Patterns for finding sensitive information
	tokenFormRegex = regexp.MustCompile(`(steam_clienttoken=)[^&]*`)
	tokenJSONRegex = regexp.MustCompile(`("(?:access_token|refresh_token)"\s*:\s*")[^"]*"`)
	initTokenRegex = regexp.MustCompile(`("token"\s*:\s*")[^"]*"`)
)

// readLines reads the log, normalizes line endings and strips ANSI color codes
func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := strings.Split(string(data), "\n")

	lines := make([]string, len(raw))
	for i, l := range raw {
		lines[i] = ansiRegex.ReplaceAllString(strings.TrimSuffix(l, "\r"), "")
	}

	return lines, nil
}

// parseExchanges walks the log and builds one Exchange per captured block
func parseExchanges(lines []string) []Exchange {
	var starts []int
	for i, l := range lines {
		if reqStartRegex.MatchString(l) {
			starts = append(starts, i)
		}
	}

	var exchanges []Exchange

	for s, startIdx := range starts {
		blockEnd := len(lines)
		if s+1 < len(starts) {
			blockEnd = starts[s+1]
		}

		start := reqStartRegex.FindStringSubmatch(lines[startIdx])
		timestamp, method, path := start[1], start[2], start[3]

		// Find the response marker within this block
		respIdx := -1
		for k := startIdx + 1; k < blockEnd; k++ {
			if responseRegex.MatchString(lines[k]) {
				respIdx = k
				break
			}
		}

		// Malformed / truncated block
		if respIdx < 0 {
			continue
		}

		status := responseRegex.FindStringSubmatch(lines[respIdx])[1]

		// Request runs from after the request line up to the response marker
		reqHeaders, reqBody := splitSection(lines[startIdx+1 : respIdx])

		// Response runs from after the marker up to the next ordinary log line or the end of the block
		resEnd := respIdx + 1
		for resEnd < blockEnd && !logLineRegex.MatchString(lines[resEnd]) {
			resEnd++
		}

		resHeaders, resBody := splitSection(lines[respIdx+1 : resEnd])

		// Redact sensitive information
		redactHeaders(reqHeaders)
		reqBody = redactBody(path, reqBody)
		redactHeaders(resHeaders)
		resBody = redactBody(path, resBody)

		exchanges = append(exchanges, Exchange{
			Method:    method,
			URL:       path,
			Timestamp: timestamp,
			Request:   Message{Headers: reqHeaders, Body: parseBody(reqBody)},
			Response:  Message{Status: status, Headers: resHeaders, Body: parseBody(resBody)},
		})
	}

	return exchanges
}

// splitSection separates a block section into its headers and the remaining body text
func splitSection(section []string) (map[string]string, string) {
	headers := map[string]string{}

	i := 0
	for i < len(section) && headerRegex.MatchString(section[i]) {
		idx := strings.Index(section[i], ":")
		name := strings.TrimSpace(section[i][:idx])
		value := strings.TrimSpace(section[i][idx+1:])

		headers[name] = value
		i++
	}

	body := section[i:]

	end := len(body)
	for end > 0 && strings.TrimSpace(body[end-1]) == "" {
		end--
	}

	return headers, strings.TrimSpace(strings.Join(body[:end], "\n"))
}

// redactHeaders strips credentials from a collection of headers
func redactHeaders(headers map[string]string) {
	if headers["Authorization"] != "" {
		headers["Authorization"] = "[redacted]"
	}
}

// redactBody strips credentials from a request/response body based on the endpoint
func redactBody(path, body string) string {
	switch {
	case strings.HasSuffix(path, "/token"):
		body = tokenFormRegex.ReplaceAllString(body, "${1}[redacted]")
		body = tokenJSONRegex.ReplaceAllString(body, `${1}[redacted]"`)

	case strings.HasSuffix(path, "/Init"):
		body = initTokenRegex.ReplaceAllString(body, `${1}[redacted]"`)
	}

	return body
}

// parseBody returns the body verbatim when it is valid JSON, wrapped
// as a JSON string when it isn't, or nil for an empty "<empty>" body.
func parseBody(text string) json.RawMessage {
	t := strings.TrimSpace(text)
	if t == "" || t == "<empty>" {
		return nil
	}

	if json.Valid([]byte(t)) {
		return json.RawMessage(t)
	}

	return encodeString(t)
}

// encodeString returns a JSON string without escaping HTML characters
func encodeString(s string) json.RawMessage {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)

	return json.RawMessage(bytes.TrimRight(b.Bytes(), "\n"))
}
