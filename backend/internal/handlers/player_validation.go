package handlers

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

var disposableEmailDomains = loadDisposableEmailDomains()

// loadDisposableEmailDomains reads data/disposable_email_domains.txt
// (one domain per line, blank lines and # comments ignored).
// A missing file yields an empty set so registration keeps working.
func loadDisposableEmailDomains() map[string]bool {
	set := make(map[string]bool)
	f, err := os.Open(csvPath("disposable_email_domains.txt"))
	if err != nil {
		logger.S.Warnf("disposable email list not found: %v", err)
		return set
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.ToLower(strings.TrimSpace(sc.Text()))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[line] = true
	}
	return set
}

func isDisposableEmail(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])
	return disposableEmailDomains[domain]
}

func isValidUsername(username string) bool {
	for _, r := range username {
		if !unicode.IsLetter(r) && r != '-' && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
