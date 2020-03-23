package main

func main() {

	var plugin Plugin
	plugin.Namespace = "AWS/DX"
	plugin.DimensionName = "ConnectionId"
	plugin.DimensionMetrics = "ConnectionState"
	plugin.MetricInfos = map[string]MetricInfo{
		"dconnect.connection.egress": MetricInfo{
			Label: "ConnectionBpsEgress",
			Unit:  "float",
		},
		"dconnect.connection.ingress": MetricInfo{
			Label: "ConnectionBpsIngress",
			Unit:  "float",
		},
		"dconnect.connection.crcerror": MetricInfo{
			Label: "ConnectionCRCErrorCount",
			Unit:  "integer",
		},
	}
	plugin.do()
}
