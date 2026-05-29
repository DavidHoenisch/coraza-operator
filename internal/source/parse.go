package source

import (
	"bufio"
	"bytes"
	"strings"
)

// ParseBlocklist extracts IP/CIDR entries from a newline-delimited blocklist file.
// Empty lines and lines starting with # are ignored.
func ParseBlocklist(content []byte) []string {
	var ips []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ips = append(ips, line)
	}
	return ips
}
