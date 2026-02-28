package gitops

import (
	"log"
	"net/url"
	"strings"
)

// domainMapping stores the domain replacement rules: source → target.
var domainMapping map[string]string

// DomainMappingItem represents a single domain rewrite rule.
type DomainMappingItem struct {
	From string
	To   string
}

// InitDomainMapping initializes git URL domain rewrite rules.
func InitDomainMapping(items []DomainMappingItem) {
	if len(items) == 0 {
		return
	}
	domainMapping = make(map[string]string, len(items))
	for _, item := range items {
		domainMapping[item.From] = item.To
	}
	log.Printf("[gitops] domain mapping initialized: %v", domainMapping)
}

// rewriteGitURL replaces the host in a git URL according to domainMapping.
// If no mapping matches, the original URL is returned unchanged.
func rewriteGitURL(gitURL string) string {
	if len(domainMapping) == 0 {
		return gitURL
	}

	u, err := url.Parse(gitURL)
	if err != nil {
		return gitURL
	}

	for src, dst := range domainMapping {
		if strings.EqualFold(u.Host, src) {
			original := gitURL
			u.Host = dst
			log.Printf("[gitops] rewrite git URL: %s → %s", original, u.String())
			return u.String()
		}
	}
	return gitURL
}
