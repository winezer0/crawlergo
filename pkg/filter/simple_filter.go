package filter

import (
	"fmt"
	"strings"

	"github.com/Qianlitp/crawlergo/pkg/config"
	"github.com/Qianlitp/crawlergo/pkg/model"
	mapset "github.com/deckarep/golang-set"
	"golang.org/x/net/publicsuffix"
)

const (
	ScopeFilterCurrDomain = "curr_domain"
	ScopeFilterWildDomain = "wild_domain"
	ScopeFilterUnlimit    = "unlimit"
)

type SimpleFilter struct {
	UniqueSet       mapset.Set
	HostLimit       string
	ScopeMode       string
	allowedHosts    map[string]struct{}
	allowedRoots    map[string]struct{}
	staticSuffixSet mapset.Set
}

func NewSimpleFilter(host string) *SimpleFilter {
	staticSuffixSet := config.StaticSuffixSet.Clone()

	for _, suffix := range []string{"js", "css", "json"} {
		staticSuffixSet.Add(suffix)
	}
	s := &SimpleFilter{UniqueSet: mapset.NewSet(), staticSuffixSet: staticSuffixSet, HostLimit: host}
	return s
}

// NewScopeFilter creates a task filter using all supplied target hosts as its scope roots.
func NewScopeFilter(targets []*model.Request, scope string) (*SimpleFilter, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = ScopeFilterCurrDomain
	}
	if scope != ScopeFilterCurrDomain && scope != ScopeFilterWildDomain && scope != ScopeFilterUnlimit {
		return nil, fmt.Errorf("unsupported scope filter %q", scope)
	}

	filter := NewSimpleFilter("")
	filter.ScopeMode = scope
	filter.allowedHosts = make(map[string]struct{}, len(targets))
	filter.allowedRoots = make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if target == nil || target.URL == nil {
			continue
		}
		host := normalizeScopeHost(target.URL.Hostname())
		if host == "" {
			continue
		}
		filter.allowedHosts[host] = struct{}{}
		if scope == ScopeFilterWildDomain {
			root, err := publicsuffix.EffectiveTLDPlusOne(host)
			if err == nil {
				filter.allowedRoots[normalizeScopeHost(root)] = struct{}{}
			}
		}
	}
	if len(filter.allowedHosts) == 0 {
		return nil, fmt.Errorf("scope filter requires at least one valid target host")
	}
	return filter, nil
}

// InScope reports whether a request belongs to this filter's configured scope.
func (s *SimpleFilter) InScope(req *model.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	if s.ScopeMode == "" {
		return s.HostLimit == "" || !s.DomainFilter(req)
	}
	scheme := strings.ToLower(strings.TrimSpace(req.URL.Scheme))
	if scheme != "http" && scheme != "https" {
		return false
	}
	host := normalizeScopeHost(req.URL.Hostname())
	if host == "" || s.ScopeMode == ScopeFilterUnlimit {
		return host != ""
	}
	if _, ok := s.allowedHosts[host]; ok {
		return true
	}
	if s.ScopeMode != ScopeFilterWildDomain {
		return false
	}
	for root := range s.allowedRoots {
		if host == root || strings.HasSuffix(host, "."+root) {
			return true
		}
	}
	return false
}

func normalizeScopeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

/*
*
需要过滤则返回 true
*/
func (s *SimpleFilter) DoFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if !s.InScope(req) {
		return true
	}
	// 首先判断是否需要过滤域名
	// 去重
	if s.UniqueFilter(req) {
		return true
	}
	// 过滤静态资源
	if s.StaticFilter(req) {
		return true
	}
	return false
}

/*
*
请求去重
*/
func (s *SimpleFilter) UniqueFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if s.UniqueSet.Contains(req.UniqueId()) {
		return true
	} else {
		s.UniqueSet.Add(req.UniqueId())
		return false
	}
}

/*
*
静态资源过滤
*/
func (s *SimpleFilter) StaticFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	// 首先将slice转换成map

	if req.URL.FileExt() == "" {
		return false
	}
	if s.staticSuffixSet.Contains(req.URL.FileExt()) {
		return true
	}
	return false
}

/*
*
只保留指定域名的链接
*/
func (s *SimpleFilter) DomainFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if req.URL.Host == s.HostLimit || req.URL.Hostname() == s.HostLimit {
		return false
	}
	if strings.HasSuffix(s.HostLimit, ":80") && req.URL.Port() == "" && req.URL.Scheme == "http" {
		if req.URL.Hostname()+":80" == s.HostLimit {
			return false
		}
	}
	if strings.HasSuffix(s.HostLimit, ":443") && req.URL.Port() == "" && req.URL.Scheme == "https" {
		if req.URL.Hostname()+":443" == s.HostLimit {
			return false
		}
	}
	return true
}
