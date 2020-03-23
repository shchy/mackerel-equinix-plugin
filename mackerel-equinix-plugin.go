package main

func main() {

	var plugin Plugin
	plugin.Namespace = "AWS/ELB"
	plugin.DimensionName = "AvailabilityZone"
	plugin.DimensionMetrics = "HealthyHostCount"
	plugin.MetricInfos = map[string]MetricInfo{
		"elb.latency": MetricInfo{
			Label:    "Latency",
			Unit:     "float",
			StatType: stAve,
		},
	}
	plugin.do()
}
