package main

import (
	"errors"
	"flag"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

type statType int

const (
	stAve statType = iota
	stSum
)

func (s statType) String() string {
	switch s {
	case stAve:
		return "Average"
	case stSum:
		return "Sum"
	}
	return ""
}

// Plugin プラグインの型
type Plugin struct {
	IDs             []*string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	CloudWatch      *cloudwatch.CloudWatch
}

// GraphDefinition グラフ定義
func (p Plugin) GraphDefinition() map[string]mp.Graphs {
	var graphdef = map[string]mp.Graphs{}

	for _, id := range p.IDs {

		graphdef["dconnect.connection.egress"] = mp.Graphs{
			Label: "ConnectionBpsEgress",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: *id + ".ConnectionBpsEgress", Label: *id},
			},
		}
		graphdef["dconnect.connection.ingress"] = mp.Graphs{
			Label: "ConnectionBpsIngress",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: *id + ".ConnectionBpsIngress", Label: *id},
			},
		}
		graphdef["dconnect.connection.crcerror"] = mp.Graphs{
			Label: "ConnectionCRCErrorCount",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: *id + ".ConnectionCRCErrorCount", Label: *id, Stacked: true},
			},
		}
	}

	return graphdef
}

// FetchMetrics metricsの取得
func (p Plugin) FetchMetrics() (map[string]float64, error) {
	stat := make(map[string]float64)

	// Metrics per ID
	for _, id := range p.IDs {
		d := []*cloudwatch.Dimension{
			{
				Name:  aws.String("ConnectionId"),
				Value: id,
			},
		}
		for _, met := range []string{"ConnectionBpsEgress", "ConnectionBpsIngress", "ConnectionCRCErrorCount"} {
			v, err := p.getLastPoint(d, met, stAve)
			if err == nil {
				stat[*id+"."+met] = v
			}
		}
	}

	return stat, nil
}

func (p Plugin) getLastPoint(dimensions []*cloudwatch.Dimension, metricName string, sTyp statType) (float64, error) {
	now := time.Now()

	response, err := p.CloudWatch.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Dimensions: dimensions,
		StartTime:  aws.Time(now.Add(time.Duration(120) * time.Second * -1)), // 2 min (to fetch at least 1 data-point)
		EndTime:    aws.Time(now),
		MetricName: aws.String(metricName),
		Period:     aws.Int64(60),
		Statistics: []*string{aws.String(sTyp.String())},
		Namespace:  aws.String("AWS/DX"),
	})
	if err != nil {
		return 0, err
	}

	datapoints := response.Datapoints
	if len(datapoints) == 0 {
		return 0, errors.New("fetched no datapoints")
	}

	latest := new(time.Time)
	var latestVal float64
	for _, dp := range datapoints {
		if dp.Timestamp.Before(*latest) {
			continue
		}

		latest = dp.Timestamp
		switch sTyp {
		case stAve:
			latestVal = *dp.Average
		case stSum:
			latestVal = *dp.Sum
		}
	}

	return latestVal, nil
}

func (p *Plugin) prepare() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	config := aws.NewConfig()
	if p.AccessKeyID != "" && p.SecretAccessKey != "" {
		config = config.WithCredentials(credentials.NewStaticCredentials(p.AccessKeyID, p.SecretAccessKey, ""))
	}
	if p.Region != "" {
		config = config.WithRegion(p.Region)
	}

	p.CloudWatch = cloudwatch.New(sess, config)

	ret, err := p.CloudWatch.ListMetrics(&cloudwatch.ListMetricsInput{
		Namespace: aws.String("AWS/DX"),
		Dimensions: []*cloudwatch.DimensionFilter{
			{
				Name: aws.String("ConnectionId"),
			},
		},
		MetricName: aws.String("ConnectionState"),
	})

	if err != nil {
		return err
	}

	p.IDs = make([]*string, 0, len(ret.Metrics))
	for _, met := range ret.Metrics {
		if len(met.Dimensions) > 1 {
			continue
		} else if *met.Dimensions[0].Name != "ConnectionId" {
			continue
		}

		p.IDs = append(p.IDs, met.Dimensions[0].Value)
	}

	return nil
}

func main() {
	optRegion := flag.String("region", "", "AWS Region")
	optAccessKeyID := flag.String("access-key-id", "", "AWS Access Key ID")
	optSecretAccessKey := flag.String("secret-access-key", "", "AWS Secret Access Key")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var plugin Plugin

	if *optRegion == "" {
		ec2metadata := ec2metadata.New(session.New())
		if ec2metadata.Available() {
			plugin.Region, _ = ec2metadata.Region()
		}
	} else {
		plugin.Region = *optRegion
	}
	plugin.AccessKeyID = *optAccessKeyID
	plugin.SecretAccessKey = *optSecretAccessKey

	err := plugin.prepare()
	if err != nil {
		log.Fatalln(err)
	}

	helper := mp.NewMackerelPlugin(plugin)
	helper.Tempfile = *optTempfile

	helper.Run()
}
