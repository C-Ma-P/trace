package phoneintake

import (
	_ "embed"
	"strings"
)

//go:embed page.html
var pageHTML string

func phonePage(token string) string {
	return strings.Replace(pageHTML, "{{TOKEN}}", token, 1)
}
