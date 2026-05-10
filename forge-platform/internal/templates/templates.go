// Package templates serves the curated catalogues used by the create-repo
// wizard: licenses (LICENSE file content) and .gitignore templates.
//
// Curated rather than mirrored from upstream so the binary stays
// self-contained and the catalogue is tiny. Add more entries as users ask.
package templates

import (
	"strconv"
	"strings"
	"time"
)

type License struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Body string `json:"body,omitempty"` // omitted from /licenses list, included on individual fetch
}

type Gitignore struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Body string `json:"body,omitempty"`
}

// Render substitutes [year] and [fullname] placeholders.
func (l License) Render(fullName string) string {
	body := strings.ReplaceAll(l.Body, "[year]", strconv.Itoa(time.Now().Year()))
	body = strings.ReplaceAll(body, "[fullname]", fullName)
	return body
}

// Licenses sorted alphabetically by Name for stable UI ordering.
var Licenses = []License{
	{Key: "apache-2.0", Name: "Apache License 2.0", Body: licenseApache20},
	{Key: "bsd-3-clause", Name: "BSD 3-Clause", Body: licenseBSD3},
	{Key: "gpl-3.0", Name: "GNU GPL v3.0", Body: licenseGPL3},
	{Key: "isc", Name: "ISC License", Body: licenseISC},
	{Key: "mit", Name: "MIT License", Body: licenseMIT},
	{Key: "unlicense", Name: "The Unlicense", Body: licenseUnlicense},
}

func LicenseByKey(k string) (License, bool) {
	for _, l := range Licenses {
		if l.Key == k {
			return l, true
		}
	}
	return License{}, false
}

var Gitignores = []Gitignore{
	{Key: "android", Name: "Android", Body: gitignoreAndroid},
	{Key: "c", Name: "C", Body: gitignoreC},
	{Key: "cpp", Name: "C++", Body: gitignoreCPP},
	{Key: "go", Name: "Go", Body: gitignoreGo},
	{Key: "java", Name: "Java", Body: gitignoreJava},
	{Key: "kotlin", Name: "Kotlin", Body: gitignoreKotlin},
	{Key: "node", Name: "Node", Body: gitignoreNode},
	{Key: "python", Name: "Python", Body: gitignorePython},
	{Key: "ruby", Name: "Ruby", Body: gitignoreRuby},
	{Key: "rust", Name: "Rust", Body: gitignoreRust},
	{Key: "swift", Name: "Swift", Body: gitignoreSwift},
}

func GitignoreByKey(k string) (Gitignore, bool) {
	for _, g := range Gitignores {
		if g.Key == k {
			return g, true
		}
	}
	return Gitignore{}, false
}
