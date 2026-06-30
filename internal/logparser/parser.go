package logparser

import (
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Parser struct{}

type AccessEntry struct {
	UserID   int64
	Email    string
	SourceIP string
	Raw      string
}

type ParseResult struct {
	Entries []AccessEntry
	Skipped int
}

var sourceEndpointPattern = regexp.MustCompile(`(?i)(?:^|[^0-9A-Za-z_.:\[\]])(((?:tcp|udp):)?(?:\[[0-9A-F:.]+\]|(?:[0-9]{1,3}\.){3}[0-9]{1,3}):[0-9]{1,5})(?:$|[^0-9A-Za-z_.:\[\]])`)

func ExtractSourceIP(line string) (string, bool) {
	for _, match := range sourceEndpointPattern.FindAllStringSubmatchIndex(line, -1) {
		if len(match) < 4 || match[2] < 0 || match[3] < 0 {
			continue
		}
		candidate := line[match[2]:match[3]]
		if ip, ok := parseEndpointCandidate(candidate); ok {
			return ip, true
		}
	}

	return "", false
}

func ParseLine(line string) (AccessEntry, bool) {
	email, userID, ok := findStableUserEmail(line)
	if !ok {
		return AccessEntry{}, false
	}

	sourceIP, ok := ExtractSourceIP(line)
	if !ok {
		return AccessEntry{}, false
	}

	return AccessEntry{
		UserID:   userID,
		Email:    email,
		SourceIP: sourceIP,
		Raw:      line,
	}, true
}

func ParseLines(lines []string) ParseResult {
	result := ParseResult{
		Entries: make([]AccessEntry, 0, len(lines)),
	}

	for _, line := range lines {
		entry, ok := ParseLine(line)
		if !ok {
			result.Skipped++
			continue
		}
		result.Entries = append(result.Entries, entry)
	}

	return result
}

func BuildOnlineIPs(entries []AccessEntry) []nodeapi.OnlineIP {
	seen := make(map[onlineIPKey]struct{}, len(entries))
	online := make([]nodeapi.OnlineIP, 0, len(entries))

	for _, entry := range entries {
		if entry.UserID <= 0 || strings.TrimSpace(entry.SourceIP) == "" {
			continue
		}

		key := onlineIPKey{
			userID: entry.UserID,
			ip:     strings.TrimSpace(entry.SourceIP),
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		online = append(online, nodeapi.OnlineIP{
			UserID: key.userID,
			IP:     key.ip,
		})
	}

	sort.Slice(online, func(i, j int) bool {
		if online[i].UserID != online[j].UserID {
			return online[i].UserID < online[j].UserID
		}
		return online[i].IP < online[j].IP
	})

	return online
}

type onlineIPKey struct {
	userID int64
	ip     string
}

func parseEndpointCandidate(candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	lower := strings.ToLower(candidate)
	if strings.HasPrefix(lower, "tcp:") || strings.HasPrefix(lower, "udp:") {
		candidate = candidate[4:]
	}

	host, port, err := net.SplitHostPort(candidate)
	if err != nil {
		return "", false
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber <= 0 || portNumber > 65535 {
		return "", false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}

	return ip.String(), true
}
