package pkg

import (
	"strings"

	"github.com/Qianlitp/crawlergo/pkg/model"
	mapset "github.com/deckarep/golang-set"
	"golang.org/x/net/publicsuffix"
)

func SubDomainCollect(reqList []*model.Request, HostLimit string) []string {
	return SubDomainCollectForRoots(reqList, []string{HostLimit})
}

// SubDomainCollectForRoots collects subdomains under any of the task's registrable roots.
func SubDomainCollectForRoots(reqList []*model.Request, roots []string) []string {
	var subDomainList []string
	uniqueSet := mapset.NewSet()
	rootSet := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		root = normalizeDomainHost(root)
		if root != "" {
			rootSet[root] = struct{}{}
		}
	}
	for _, req := range reqList {
		if req == nil || req.URL == nil {
			continue
		}
		domain := normalizeDomainHost(req.URL.Hostname())
		if uniqueSet.Contains(domain) {
			continue
		}
		uniqueSet.Add(domain)
		for root := range rootSet {
			if domain != root && strings.HasSuffix(domain, "."+root) {
				subDomainList = append(subDomainList, domain)
				break
			}
		}
	}
	return subDomainList
}

// RootDomainsForTargets returns unique registrable roots for all valid target requests.
func RootDomainsForTargets(targets []*model.Request) []string {
	seen := make(map[string]struct{}, len(targets))
	roots := make([]string, 0, len(targets))
	for _, target := range targets {
		if target == nil || target.URL == nil {
			continue
		}
		host := normalizeDomainHost(target.URL.Hostname())
		root, err := publicsuffix.EffectiveTLDPlusOne(host)
		if err != nil {
			continue
		}
		root = normalizeDomainHost(root)
		if _, exists := seen[root]; exists {
			continue
		}
		seen[root] = struct{}{}
		roots = append(roots, root)
	}
	return roots
}

func normalizeDomainHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func AllDomainCollect(reqList []*model.Request) []string {
	uniqueSet := mapset.NewSet()
	var allDomainList []string
	for _, req := range reqList {
		if req == nil || req.URL == nil {
			continue
		}
		domain := req.URL.Hostname()
		if uniqueSet.Contains(domain) {
			continue
		}
		uniqueSet.Add(domain)
		allDomainList = append(allDomainList, req.URL.Hostname())
	}
	return allDomainList
}
