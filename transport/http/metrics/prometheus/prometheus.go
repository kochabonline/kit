package prometheus

import (
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Config struct {
	Path                      string
	EnabledGoCollector        bool
	EnabledBuildInfoCollector bool
}

type Prometheus struct {
	Config    Config
	Registry  *prometheus.Registry
	gauge     *prometheus.GaugeVec
	histogram *prometheus.HistogramVec
	summery   *prometheus.SummaryVec
	counter   *prometheus.CounterVec
}

func NewPrometheus(c Config) *Prometheus {
	if c.Path == "" {
		c.Path = "/metrics"
	}

	p := &Prometheus{
		Config:   c,
		Registry: prometheus.NewRegistry(),
	}

	if c.EnabledGoCollector {
		p.WithGoCollectorRuntimeMetrics()
	}
	if c.EnabledBuildInfoCollector {
		p.WithBuildInfoCollector()
	}

	return p
}

func (p *Prometheus) RegisterGauge(name, help string, labels []string) {
	p.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels)

	p.Registry.MustRegister(p.gauge)
}

func (p *Prometheus) RegisterHistogram(name, help string, labels []string, buckets []float64) {
	p.histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, labels)

	p.Registry.MustRegister(p.histogram)
}

func (p *Prometheus) RegisterSummary(name, help string, labels []string, objectives map[float64]float64) {
	p.summery = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       name,
		Help:       help,
		Objectives: objectives,
	}, labels)

	p.Registry.MustRegister(p.summery)
}

func (p *Prometheus) RegisterCounter(name, help string, labels []string) {
	p.counter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)

	p.Registry.MustRegister(p.counter)
}

func (p *Prometheus) WithGoCollectorRuntimeMetrics() {
	p.Registry.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")}),
	))
}

func (p *Prometheus) WithBuildInfoCollector() {
	p.Registry.MustRegister(collectors.NewBuildInfoCollector())
}
