package http

import "github.com/kochabonline/kit/core/reflect"

type Options struct {
	Swag    SwagOptions
	Metrics MetricsOptions
	Health  HealthOptions
}

type SwagOptions struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path" default:"/swagger/*any"`
}

func (s *SwagOptions) init() error {
	return reflect.SetDefaultTag(s)
}

type MetricsOptions struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path" default:"/metrics"`
}

func (m *MetricsOptions) init() error {
	return reflect.SetDefaultTag(m)
}

type HealthOptions struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path" default:"/health"`
}

func (h *HealthOptions) init() error {
	return reflect.SetDefaultTag(h)
}
