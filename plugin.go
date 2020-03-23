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

type MetricInfo struct {
	Label    string
	Unit     string
	StatType statType
}

// Plugin プラグインの型
type Plugin struct {
	Namespace        string
	DimensionName    string
	DimensionMetrics string
	MetricInfos      map[string]MetricInfo

	Dimensions      []*string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	CloudWatch      *cloudwatch.CloudWatch
}

// GraphDefinition グラフ定義
func (p Plugin) GraphDefinition() map[string]mp.Graphs {
	var graphdef = map[string]mp.Graphs{}

	for grp, info := range p.MetricInfos {
		var metrics []mp.Metrics
		for _, dimension := range p.Dimensions {
			metrics = append(metrics, mp.Metrics{Name: info.Label + "_" + *dimension, Label: *dimension})
		}
		graphdef[grp] = mp.Graphs{
			Label:   info.Label,
			Unit:    info.Unit,
			Metrics: metrics,
		}
	}
	return graphdef
}

// FetchMetrics metricsの取得
func (p Plugin) FetchMetrics() (map[string]float64, error) {
	stat := make(map[string]float64)

	// Metrics per ID
	for _, dimension := range p.Dimensions {
		d := []*cloudwatch.Dimension{
			{
				Name:  aws.String(p.DimensionName),
				Value: dimension,
			},
		}

		for _, info := range p.MetricInfos {
			v, err := p.getLastPoint(d, info.Label, info.StatType)
			if err == nil {
				stat[info.Label+"_"+*dimension] = v
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
		Namespace:  aws.String(p.Namespace),
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
		Namespace: aws.String(p.Namespace),
		Dimensions: []*cloudwatch.DimensionFilter{
			{
				Name: aws.String(p.DimensionName),
			},
		},
		MetricName: aws.String(p.DimensionMetrics),
	})

	if err != nil {
		return err
	}

	p.Dimensions = make([]*string, 0, len(ret.Metrics))
	for _, met := range ret.Metrics {
		if len(met.Dimensions) > 1 {
			continue
		} else if *met.Dimensions[0].Name != p.DimensionName {
			continue
		}

		p.Dimensions = append(p.Dimensions, met.Dimensions[0].Value)
	}

	return nil
}

func (p Plugin) do() {
	optRegion := flag.String("region", "", "AWS Region")
	optAccessKeyID := flag.String("access-key-id", "", "AWS Access Key ID")
	optSecretAccessKey := flag.String("secret-access-key", "", "AWS Secret Access Key")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	if *optRegion == "" {
		ec2metadata := ec2metadata.New(session.New())
		if ec2metadata.Available() {
			p.Region, _ = ec2metadata.Region()
		}
	} else {
		p.Region = *optRegion
	}
	p.AccessKeyID = *optAccessKeyID
	p.SecretAccessKey = *optSecretAccessKey

	err := p.prepare()
	if err != nil {
		log.Fatalln(err)
	}

	helper := mp.NewMackerelPlugin(p)
	helper.Tempfile = *optTempfile

	helper.Run()
}
