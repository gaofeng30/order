package migrate

import "strings"

// This is a command-boundary lexer, not a SQL grammar. The server validates SQL
// syntax. Only the transactional DDL/DML command families used by this chain are
// accepted; transaction/session control and procedural definitions cannot enter
// the runner. Strings and comments follow PostgreSQL's standard string mode.
func validPostgreSQLStatements(s string) bool {
	var head []string
	statements := 0
	escapedContinuation := false
	add := func(token string) {
		escapedContinuation = false
		if len(head) < 2 {
			head = append(head, token)
		}
	}
	for i := 0; i < len(s); {
		switch {
		case strings.ContainsRune(" \t\n\f\v", rune(s[i])):
			i++
		case strings.HasPrefix(s[i:], "--"):
			end := strings.IndexByte(s[i:], '\n')
			if end < 0 {
				i = len(s)
			} else {
				i += end + 1
			}
		case strings.HasPrefix(s[i:], "/*"):
			i += 2
			depth := 1
			for i < len(s) && depth > 0 {
				switch {
				case strings.HasPrefix(s[i:], "/*"):
					depth++
					i += 2
				case strings.HasPrefix(s[i:], "*/"):
					depth--
					i += 2
				default:
					i++
				}
			}
			if depth != 0 {
				return false
			}
		case s[i] == ';':
			if !allowedPostgreSQLCommand(head) {
				return false
			}
			statements++
			head = nil
			escapedContinuation = false
			i++
		case (s[i] == 'u' || s[i] == 'U') && strings.HasPrefix(s[i+1:], "&") && i+2 < len(s) && (s[i+2] == '\'' || s[i+2] == '"'):
			// Unicode escape quoting is not used by the migration chain. Do not
			// interpret escaped identifiers as ordinary quoted names.
			return false
		case s[i] == '\'' || s[i] == '"':
			quote := s[i]
			if quote == '\'' && escapedContinuation {
				// PostgreSQL carries E-string escaping into newline-separated
				// adjacent literals; that form is outside this bounded lexer.
				return false
			}
			escaped := quote == '\'' && i > 0 && (s[i-1] == 'e' || s[i-1] == 'E') && (i == 1 || !postgresIdentifierPart(s[i-2]))
			add("<quoted>")
			i++
			start := i
			closed := false
			for i < len(s) {
				if escaped && s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == quote {
					i++
					if i < len(s) && s[i] == quote {
						i++
						continue
					}
					closed = true
					break
				}
				i++
			}
			if !closed {
				return false
			}
			if quote == '"' && forbiddenPostgreSQLSessionIdentifier(strings.ReplaceAll(s[start:i-1], "\"\"", "\"")) {
				return false
			}
			escapedContinuation = escaped
		case s[i] == '$':
			end := i + 1
			if end < len(s) && postgresIdentifierStart(s[end]) {
				for end < len(s) && (postgresIdentifierStart(s[end]) || s[end] >= '0' && s[end] <= '9') {
					end++
				}
			}
			if end >= len(s) || s[end] != '$' {
				return false
			}
			delimiter := s[i : end+1]
			closeAt := strings.Index(s[end+1:], delimiter)
			if closeAt < 0 {
				return false
			}
			add("<quoted>")
			i = end + 1 + closeAt + len(delimiter)
		case postgresIdentifierStart(s[i]):
			start := i
			for i < len(s) && postgresIdentifierPart(s[i]) {
				i++
			}
			if forbiddenPostgreSQLSessionIdentifier(s[start:i]) {
				return false
			}
			add(strings.ToUpper(s[start:i]))
		default:
			add("<symbol>")
			i++
		}
	}
	return statements > 0 && len(head) == 0
}

func forbiddenPostgreSQLSessionIdentifier(name string) bool {
	name = strings.ToLower(name)
	return name == "set_config" || strings.HasPrefix(name, "pg_advisory_") || strings.HasPrefix(name, "pg_try_advisory_")
}

func postgresIdentifierStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= 128
}

func postgresIdentifierPart(c byte) bool {
	return postgresIdentifierStart(c) || c >= '0' && c <= '9' || c == '$'
}

func allowedPostgreSQLCommand(head []string) bool {
	if len(head) == 0 {
		return false
	}
	switch head[0] {
	case "SELECT", "INSERT", "UPDATE", "DELETE":
		return true
	case "CREATE":
		return len(head) == 2 && (head[1] == "TABLE" || head[1] == "SEQUENCE" || head[1] == "INDEX")
	case "ALTER":
		return len(head) == 2 && (head[1] == "TABLE" || head[1] == "SEQUENCE")
	case "DROP":
		return len(head) == 2 && head[1] == "INDEX"
	default:
		return false
	}
}
