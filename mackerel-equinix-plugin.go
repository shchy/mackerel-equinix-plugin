package main

import (
	"flag"
	"math/rand"
	"strings"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// Plugin プラグインの型
type Plugin struct {
	Prefix string
}

// GraphDefinition グラフ定義
func (p Plugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(p.MetricKeyPrefix())
	return map[string]mp.Graphs{
		"": {
			Label: labelPrefix,
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "seconds", Label: "Seconds"},
			},
		},
	}
}

// FetchMetrics metricsの取得
func (p Plugin) FetchMetrics() (map[string]float64, error) {
	rand.Seed(time.Now().UnixNano())
	return map[string]float64{"seconds": float64(rand.Intn(10))}, nil
}

// MetricKeyPrefix Prefixの取得
func (p Plugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "equinix"
	}
	return p.Prefix
}

func main() {
	optPrefix := flag.String("metric-key-prefix", "equinix", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	u := Plugin{
		Prefix: *optPrefix,
	}
	plugin := mp.NewMackerelPlugin(u)
	plugin.Tempfile = *optTempfile
	plugin.Run()
}
