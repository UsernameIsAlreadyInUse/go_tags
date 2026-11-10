package tags

import (
	"regexp"
	"strings"
)

// spaces находит любую последовательность пробельных символов.
// В Go \s означает [\t\n\f\r ] — ровно то же, что \s в ClickHouse (RE2).
var spaces = regexp.MustCompile(`\s+`)

// Normalize приводит запрос к виду, в котором он хранится в таблице query_tags.
// Правила те же, что в SQL: нижний регистр, схлопывание пробелов, обрезка по краям.
func Normalize(s string) string {
	s = strings.ToLower(s)
	s = spaces.ReplaceAllString(s, " ")
	return strings.Trim(s, " ")
}
