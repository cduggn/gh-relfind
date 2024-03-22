package main

type SemanticFilter interface {
	Filter(list *ListReleases, term string) string
}

type SemanticFilterService struct {
	filter SemanticFilter
}

func NewSemanticFilterService(filter SemanticFilter) SemanticFilterService {
	return SemanticFilterService{filter: filter}
}

func (s SemanticFilterService) Filter(list *ListReleases, term string) string {
	return s.filter.Filter(list, term)
}

type TextFilter struct{}

func (t TextFilter) Filter(list *ListReleases, term string) string {
	return term
}
