package logparser

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	stableUserEmailPattern       = regexp.MustCompile(`^user-([0-9]+)@panel\.local$`)
	stableUserEmailInLinePattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9._%+\-])(user-[0-9]+@panel\.local)(?:$|[^A-Za-z0-9._%+\-])`)
)

func ParseStableUserEmail(email string) (int64, bool) {
	email = strings.TrimSpace(email)
	matches := stableUserEmailPattern.FindStringSubmatch(email)
	if len(matches) != 2 {
		return 0, false
	}

	userID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}

	return userID, true
}

func findStableUserEmail(line string) (string, int64, bool) {
	matches := stableUserEmailInLinePattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return "", 0, false
	}

	userID, ok := ParseStableUserEmail(matches[1])
	if !ok {
		return "", 0, false
	}

	return matches[1], userID, true
}
