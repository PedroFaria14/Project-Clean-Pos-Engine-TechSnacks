package identity

import "strings"

type Role string

const (
	Operator Role = "operator"
	Admin    Role = "admin"
)

// Principal contains only public account attributes, never credentials.
type Principal struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

type User struct {
	Principal
	PasswordHash string `json:"-"`
	Active       bool   `json:"-"`
}

func NormalizeUsername(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) < 3 || len(value) > 64 {
		return "", false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return "", false
		}
	}
	return value, true
}
