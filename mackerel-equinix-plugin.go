package main

func main() {

	var plugin Plugin
	plugin.Namespace = "AWS/DX"
	plugin.DimensionName = "ConnectionId"
	plugin.DimensionMetrics = "ConnectionState"
	plugin.MetricInfos = map[string]MetricInfo{
		"dcon.bpsegress": MetricInfo{
			Label:    "ConnectionBpsEgress",
			Unit:     "float",
			StatType: stAve,
		},
		"dcon.bpsingress": MetricInfo{
			Label:    "ConnectionBpsIngress",
			Unit:     "float",
			StatType: stAve,
		},
		"dcon.crcerror": MetricInfo{
			Label:    "ConnectionCRCErrorCount",
			Unit:     "integer",
			StatType: stAve,
		},
	}
	plugin.do()
}
