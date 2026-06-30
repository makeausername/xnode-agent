package logparser

import (
	"reflect"
	"testing"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestParseStableUserEmailAcceptsStableFormat(t *testing.T) {
	userID, ok := ParseStableUserEmail("user-10001@panel.local")
	if !ok {
		t.Fatal("ParseStableUserEmail() ok = false, want true")
	}
	if userID != 10001 {
		t.Fatalf("ParseStableUserEmail() userID = %d, want 10001", userID)
	}
}

func TestParseStableUserEmailRejectsRandomEmail(t *testing.T) {
	if userID, ok := ParseStableUserEmail("alice@example.com"); ok {
		t.Fatalf("ParseStableUserEmail() = %d, true; want false", userID)
	}
}

func TestExtractSourceIPHandlesIPv4WithTCPPrefix(t *testing.T) {
	ip, ok := ExtractSourceIP("tcp:203.0.113.10:54321")
	if !ok {
		t.Fatal("ExtractSourceIP() ok = false, want true")
	}
	if ip != "203.0.113.10" {
		t.Fatalf("ExtractSourceIP() = %q, want %q", ip, "203.0.113.10")
	}
}

func TestExtractSourceIPHandlesIPv6WithBrackets(t *testing.T) {
	ip, ok := ExtractSourceIP("tcp:[2001:db8::1]:54321")
	if !ok {
		t.Fatal("ExtractSourceIP() ok = false, want true")
	}
	if ip != "2001:db8::1" {
		t.Fatalf("ExtractSourceIP() = %q, want %q", ip, "2001:db8::1")
	}
}

func TestExtractSourceIPHandlesSupportedEndpointShapes(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "udp ipv4", line: "udp:203.0.113.10:54321", want: "203.0.113.10"},
		{name: "bare ipv4", line: "203.0.113.10:54321", want: "203.0.113.10"},
		{name: "bare bracketed ipv6", line: "[2001:db8::1]:54321", want: "2001:db8::1"},
		{name: "arrow suffix", line: "tcp:203.0.113.10:54321->tcp:example.com:443", want: "203.0.113.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, ok := ExtractSourceIP(tt.line)
			if !ok {
				t.Fatal("ExtractSourceIP() ok = false, want true")
			}
			if ip != tt.want {
				t.Fatalf("ExtractSourceIP() = %q, want %q", ip, tt.want)
			}
		})
	}
}

func TestParseLineExtractsUserIDAndIPv4(t *testing.T) {
	line := "2026/06/30 10:00:00 tcp:203.0.113.10:54321 accepted tcp:example.com:443 email: user-10001@panel.local"

	entry, ok := ParseLine(line)
	if !ok {
		t.Fatal("ParseLine() ok = false, want true")
	}

	want := AccessEntry{
		UserID:   10001,
		Email:    "user-10001@panel.local",
		SourceIP: "203.0.113.10",
		Raw:      line,
	}
	if entry != want {
		t.Fatalf("ParseLine() = %#v, want %#v", entry, want)
	}
}

func TestParseLineExtractsUserIDAndIPv6(t *testing.T) {
	line := "2026/06/30 10:00:00 from tcp:[2001:db8::1]:54321 accepted tcp:example.com:443 email=user-10002@panel.local"

	entry, ok := ParseLine(line)
	if !ok {
		t.Fatal("ParseLine() ok = false, want true")
	}

	want := AccessEntry{
		UserID:   10002,
		Email:    "user-10002@panel.local",
		SourceIP: "2001:db8::1",
		Raw:      line,
	}
	if entry != want {
		t.Fatalf("ParseLine() = %#v, want %#v", entry, want)
	}
}

func TestParseLineReturnsFalseForInvalidLine(t *testing.T) {
	if entry, ok := ParseLine("not an xray access line"); ok {
		t.Fatalf("ParseLine() = %#v, true; want false", entry)
	}
}

func TestParseLinesCountsSkippedLines(t *testing.T) {
	lines := []string{
		"2026/06/30 10:00:00 tcp:203.0.113.10:54321 accepted tcp:example.com:443 email: user-10001@panel.local",
		"missing email tcp:203.0.113.11:54321",
		"",
	}

	result := ParseLines(lines)
	if len(result.Entries) != 1 {
		t.Fatalf("len(ParseLines().Entries) = %d, want 1", len(result.Entries))
	}
	if result.Skipped != 2 {
		t.Fatalf("ParseLines().Skipped = %d, want 2", result.Skipped)
	}
}

func TestBuildOnlineIPsDeduplicatesRepeatedUserIPPairs(t *testing.T) {
	entries := []AccessEntry{
		{UserID: 10001, SourceIP: "203.0.113.10"},
		{UserID: 10001, SourceIP: "203.0.113.10"},
		{UserID: 10002, SourceIP: "203.0.113.10"},
	}

	got := BuildOnlineIPs(entries)
	want := []nodeapi.OnlineIP{
		{UserID: 10001, IP: "203.0.113.10"},
		{UserID: 10002, IP: "203.0.113.10"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildOnlineIPs() = %#v, want %#v", got, want)
	}
}

func TestBuildOnlineIPsOutputIsSorted(t *testing.T) {
	entries := []AccessEntry{
		{UserID: 10002, SourceIP: "203.0.113.10"},
		{UserID: 10001, SourceIP: "203.0.113.20"},
		{UserID: 10001, SourceIP: "198.51.100.1"},
	}

	got := BuildOnlineIPs(entries)
	want := []nodeapi.OnlineIP{
		{UserID: 10001, IP: "198.51.100.1"},
		{UserID: 10001, IP: "203.0.113.20"},
		{UserID: 10002, IP: "203.0.113.10"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildOnlineIPs() = %#v, want %#v", got, want)
	}
}
