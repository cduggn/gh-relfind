package main

import (
	"github.com/samber/lo"
	"strings"
)

type SemanticFilter interface {
	Filter(list []Release, term string) []Release
}

func NewSemanticFilter(service SemanticFilter) SemanticFilter {
	return service
}

type SimpleFilter struct {
}

func (t SimpleFilter) Filter(list []Release, term string) []Release {
	subset := lo.Filter(list, func(item Release, _ int) bool {
		return strings.Contains(item.Body, term)
	})

	return subset
}
