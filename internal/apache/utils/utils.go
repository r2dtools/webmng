package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/unknwon/com"
)

// TranslateFnmatchToRegex converts a shell PATTERN to a regular expression string
func TranslateFnmatchToRegex(fnmatchStr string) string {
	i, n := 0, len(fnmatchStr)
	var res string

	for i < n {
		c := string(fnmatchStr[i])
		i++

		if c == "*" {
			res = res + ".*"
		} else if c == "?" {
			res = res + "."
		} else if c == "[" {
			j := i

			if j < n && string(fnmatchStr[j]) == "!" {
				j++
			}

			if j < n && string(fnmatchStr[j]) == "]" {
				j++
			}

			for j < n && string(fnmatchStr[j]) != "]" {
				j++
			}

			if j >= n {
				res = res + "//["
			} else {
				stuff := strings.Replace(fnmatchStr[i:j], "//", "////", -1)
				i = j + 1

				if string(stuff[0]) == "!" {
					stuff = "^" + stuff[1:]
				} else if string(stuff[0]) == "^" {
					stuff = "\\" + stuff
				}

				res = fmt.Sprintf("%s[%s]", res, stuff)
			}
		} else {
			res = res + regexp.QuoteMeta(c)
		}
	}

	return fmt.Sprintf("(?s:%s)$", res)
}

// IsPortListened checks if port in the list
func IsPortListened(listens []string, port string) bool {
	if com.IsSliceContainsStr(listens, port) {
		return true
	}

	for _, listen := range listens {
		// listen can be 1.1.1.1:443 https
		lParts := strings.Split(listen, ":")

		if len(lParts) > 1 {
			p := strings.Split(lParts[len(lParts)-1], " ")

			if p[0] == port {
				return true
			}
		}
	}

	return false
}

// GetIPFromListen returns IP address from Listen directive statement
func GetIPFromListen(listen string) string {
	rListen := com.Reverse(listen)
	rParts := strings.SplitN(rListen, ":", 2)

	if len(rParts) > 1 {
		return com.Reverse(rParts[1])
	}

	return ""
}

func Escape(filePath string) string {
	filePath = strings.Replace(filePath, ",", "\\,", -1)
	filePath = strings.Replace(filePath, "[", "\\[", -1)
	filePath = strings.Replace(filePath, "]", "\\]", -1)
	filePath = strings.Replace(filePath, "|", "\\|", -1)
	filePath = strings.Replace(filePath, "=", "\\=", -1)
	filePath = strings.Replace(filePath, "(", "\\(", -1)
	filePath = strings.Replace(filePath, ")", "\\)", -1)
	filePath = strings.Replace(filePath, "!", "\\!", -1)

	return filePath
}

func DisableDangerousForSslRewriteRules(content []string) ([]string, bool) {
	var result []string
	var skipped bool
	linesCount := len(content)

	for i := 0; i < linesCount; i++ {
		line := content[i]
		isRewriteCondition := strings.HasPrefix(strings.TrimSpace(strings.ToLower(line)), "rewritecond")
		isRewriteRule := strings.HasPrefix(strings.TrimSpace(strings.ToLower(line)), "rewriterule")

		if !isRewriteRule && !isRewriteCondition {
			result = append(result, line)
			continue
		}

		isRewriteRuleDangerous := IsRewriteRuleDangerousForSsl(line)

		if isRewriteRule && !isRewriteRuleDangerous {
			result = append(result, line)
			continue
		} else if isRewriteRule && isRewriteRuleDangerous {
			skipped = true

			result = append(result, "# "+line)
		}

		if isRewriteCondition {
			var chunk []string

			chunk = append(chunk, line)
			j := i + 1

			for ; j < linesCount; j++ {
				isRewriteRuleNextLine := strings.HasPrefix(strings.TrimSpace(strings.ToLower(content[j])), "rewriterule")

				if isRewriteRuleNextLine {
					break
				}

				chunk = append(chunk, content[j])
			}

			i = j
			chunk = append(chunk, content[j])

			if IsRewriteRuleDangerousForSsl(content[j]) {
				skipped = true

				for _, l := range chunk {
					result = append(result, "# "+l)
				}
			} else {
				result = append(result, strings.Join(chunk, "\n"))
			}
		}
	}

	return result, skipped
}

// IsRewriteRuleDangerousForSsl checks if provided rewrite rule potentially can not be used for the virtual host with ssl
// e.g:
// RewriteRule ^ https://%{SERVER_NAME}%{REQUEST_URI} [L,QSA,R=permanent]
// Copying the above line to the ssl vhost would cause a
// redirection loop.
func IsRewriteRuleDangerousForSsl(line string) bool {
	line = strings.TrimSpace(strings.ToLower(line))

	if !strings.HasPrefix(line, "rewriterule") {
		return false
	}

	// According to: https://httpd.apache.org/docs/2.4/rewrite/flags.html
	// The syntax of a RewriteRule is:
	// RewriteRule pattern target [Flag1,Flag2,Flag3]
	// i.e. target is required, so it must exist.
	parts := strings.Split(line, " ")

	if len(parts) < 3 {
		return false
	}

	target := strings.TrimSpace(parts[2])
	target = strings.Trim(target, "'\"")

	return strings.HasPrefix(target, "https://")
}

// RemoveClosingHostTag removes closing tag </virtualhost> for the virtualhost block
func RemoveClosingHostTag(lines []string) {
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		tagIndex := strings.Index(strings.ToLower(line), "</virtualhost>")

		if tagIndex != -1 {
			lines[i] = line[:tagIndex]
			break
		}
	}
}
