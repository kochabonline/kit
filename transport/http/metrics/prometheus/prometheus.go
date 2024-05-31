package prometheus

import (
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var Registry = new(prometheus.Registry)

func init() {
	Registry = prometheus.NewRegistry()
	Registry.MustRegister(collectors.NewBuildInfoCollector())
	Registry.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")}),
	))
}
