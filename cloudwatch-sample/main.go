package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type ServiceMetric struct {
	ServiceName string
	TimeStamp   time.Time
	windowMin   int

	// Service Connect 來源的指標
	// Envoy sidecar 量測流量視角, canary 決策核心
	RequestCount      float64 `json:"request_count"`
	ErrorCount        float64 `json:"error_count"`
	ErrorRate         float64 `json:"error_rate"`
	P99Latency        float64 `json:"p99_latenecy"`
	P50Latency        float64 `json:"p50_latency"`
	AvgLatency        float64 `json:"avg_latency"`
	ActiveConnections float64 `json:"active_connections"`
	NewConnections    float64 `json:"new_connections"`

	// Container Insight 的指標
	CPUUtilized      float64 `json:"cpu_utilized"`    // vCPU 單位
	MemoryUtilized   float64 `json:"memory_utilized"` // MB
	RunningTaskCount float64 `json:"running_task_count"`

	// 資料來源標記
	SCDataAvailable bool `json:"sc_data_available"`
	CIDataAVailable bool `json:"ci_data_available"`
}

type responseDTO struct {
	name  string
	value float64
	err   error
}

type basic struct {
	key, metric, stat string
}

func (m *ServiceMetric) String() string {
	source := ""
	if m.SCDataAvailable {
		source += "SC"
	}
	if m.CIDataAVailable {
		if source != "" {
			source += "+"
		}
		source += "CI"
	}
	return fmt.Sprintf(
		"[%s][%s] reqs=%.0f errs=%.0f errRate=%.2f%% p50=%.1fms p99=%.1fms cpu=%.3f mem=%.0fMB",
		m.ServiceName,
		source,
		m.RequestCount,
		m.ErrorCount,
		m.ErrorRate,
		m.P50Latency,
		m.P99Latency,
		m.CPUUtilized,
		m.MemoryUtilized,
	)
}

type Collector struct {
	cw            *cloudwatch.Client
	cluster       string
	namespace     string
	serviceNammes map[string]string
}

func NewCollector(cfg aws.Config, cluster, namespace string, serviceNames map[string]string) *Collector {
	return &Collector{
		cw:            cloudwatch.NewFromConfig(cfg),
		cluster:       cluster,
		namespace:     namespace,
		serviceNammes: serviceNames,
	}
}

func (c *Collector) Collect(ctx context.Context, discoveryName string, windowMin int) (*ServiceMetric, error) {
	now := time.Now()
	start := now.Add(-time.Duration(windowMin) * time.Minute)

	result := &ServiceMetric{
		ServiceName: discoveryName,
		TimeStamp:   now,
		windowMin:   windowMin,
	}

	ecsServiceName, ok := c.serviceNammes[discoveryName]
	if !ok {
		return nil, fmt.Errorf("no ECS service name mapped for discovery name: %s", discoveryName)
	}

	// service connect
	scDims := []types.Dimension{
		{Name: aws.String("ClusterName"), Value: aws.String(c.cluster)},
		{Name: aws.String("ServiceName"), Value: aws.String(ecsServiceName)},
		{Name: aws.String("DiscoveryName"), Value: aws.String(discoveryName)},
	}

	// container insight
	ciDims := []types.Dimension{
		{Name: aws.String("ClusterName"), Value: aws.String(c.cluster)},
		{Name: aws.String("ServiceName"), Value: aws.String(ecsServiceName)},
	}

	scErr := c.collectServiceConnnect(ctx, result, scDims, start, now)
	if scErr != nil {
		log.Printf("[Collector] Service Connect 無數據 (ns: %s) (service:%s): %v\n", c.namespace, discoveryName, scErr)
	} else {
		result.SCDataAvailable = true
	}

	ciErr := c.collectContainerInsight(ctx, result, ciDims, start, now)
	if ciErr != nil {
		log.Printf("[Coolector] Container Insights 無數據 (%s): %v\n", discoveryName, ciErr)
	} else {
		result.CIDataAVailable = true
	}

	if !result.SCDataAvailable && !result.CIDataAVailable {
		return result, fmt.Errorf("兩個來源都沒數據 : %s (cluster=%s, namespace=%s)", discoveryName, c.cluster, c.namespace)
	}

	return result, nil
}

func (c *Collector) getMetricStatistic(
	ctx context.Context,
	namespace,
	metricName,
	stat string,
	dims []types.Dimension,
	start,
	end time.Time,
) (float64, error) {
	statType := types.StatisticSum
	switch stat {
	case "Maximium":
		statType = ""
		statType = types.StatisticMaximum
	case "Minimium":
		statType = types.StatisticMinimum
	case "Average":
		statType = types.StatisticAverage
	}
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Dimensions: dims,
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(60),
		Statistics: []types.Statistic{types.Statistic(statType)},
	}

	resp, err := c.cw.GetMetricStatistics(ctx, input)
	if err != nil {
		return 0, err
	}
	if len(resp.Datapoints) == 0 {
		return 0, nil
	}

	// 可能回傳多個 datapoints -> 取最新的
	sort.Slice(resp.Datapoints, func(i, j int) bool {
		return resp.Datapoints[i].Timestamp.After(*resp.Datapoints[j].Timestamp)
	})

	dp := resp.Datapoints[0]

	switch stat {
	case "Sum":
		return aws.ToFloat64(dp.Sum), nil
	case "Average":
		return aws.ToFloat64(dp.Average), nil
	case "Minimium":
		return aws.ToFloat64(dp.Minimum), nil
	case "Maximium":
		return aws.ToFloat64(dp.Maximum), nil
	default:
		return aws.ToFloat64(dp.Sum), nil
	}
}

func (c *Collector) getPercentile(
	ctx context.Context,
	namespace,
	metricName,
	percentile string,
	dims []types.Dimension,
	start, end time.Time,
) (float64, error) {
	metricDims := make([]types.Dimension, len(dims))
	copy(metricDims, dims)

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("pct"), // 查詢 ID
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String(metricName),
						Dimensions: metricDims,
					},
					Period: aws.Int32(60),
					Stat:   aws.String(percentile), // p50,p99 <-這個指的是這個
				},
			},
		},
	}

	resp, err := c.cw.GetMetricData(ctx, input)
	if err != nil {
		return 0, err
	}

	if len(resp.MetricDataResults) == 0 || len(resp.MetricDataResults[0].Values) == 0 {
		return 0, nil
	}

	return resp.MetricDataResults[0].Values[0], nil
}

// 收集 service connect 提供的資料
func (c *Collector) collectServiceConnnect(ctx context.Context, result *ServiceMetric, dims []types.Dimension, start, end time.Time) error {
	ns := "AWS/ECS"
	// ns := "AWS/ECS/ServiceConnect"

	type metricPercentile struct {
		key, pct string
	}

	ch := make(chan responseDTO, 7)

	basics := []basic{
		{"request_count", "RequestCount", "Sum"},
		{"error_count", "HTTPCode_Target_5XX_Count", "Sum"},
		{"avg_latency", "TargetResponseTime", "Average"},
		{"active_connections", "ActiveConnectionCount", "Sum"},
		{"new_connections", "NewConnectionCount", "Sum"},
	}

	others := []metricPercentile{
		{"p50_latency", "p50"},
		{"p99_latency", "p99"},
	}

	for _, q := range basics {
		go func(key, metric, stat string) {
			val, err := c.getMetricStatistic(
				ctx,
				ns,
				metric,
				stat,
				dims,
				start,
				end,
			)
			ch <- responseDTO{
				name:  key,
				value: val,
				err:   err,
			}
		}(q.key, q.metric, q.stat)
	}

	for _, p := range others {
		go func(key, pct string) {
			val, err := c.getPercentile(
				ctx,
				ns,
				"TargetResponseTime",
				pct,
				dims,
				start,
				end,
			)
			ch <- responseDTO{
				name:  key,
				value: val,
				err:   err,
			}
		}(p.key, p.pct)
	}
	hasData := false
	for range 7 {
		r := <-ch
		if r.err != nil {
			// 單一指標失敗不影響整體
			continue
		}
		if r.value > 0 {
			hasData = true
		}
		switch r.name {
		case "request_count":
			result.RequestCount = r.value
		case "error_count":
			result.ErrorCount = r.value
		case "avg_latency":
			result.AvgLatency = r.value
		case "active_connections":
			result.ActiveConnections = r.value
		case "new_connections":
			result.NewConnections = r.value
		case "p50_latency":
			result.P50Latency = r.value
		case "p99_latency":
			result.P99Latency = r.value
		}
	}

	if !hasData {
		return fmt.Errorf("service connect namespace (%s) 沒有 datapoints", ns)
	}
	if result.RequestCount > 0 {
		result.ErrorRate = result.ErrorCount / result.RequestCount * 100
	}
	return nil
}

// 收集 continaer insight 指標
func (c *Collector) collectContainerInsight(ctx context.Context, result *ServiceMetric, dims []types.Dimension, start, end time.Time) error {
	// ns := "AWS/ECS/ContainerInsights"
	ns := "ECS/ContainerInsights"

	ch := make(chan responseDTO, 3)

	queries := []basic{
		{"cpu", "CpuUtilized", "Average"},
		{"mem", "MemoryUtilized", "Average"},
		{"tasks", "RunningTaskCount", "Average"},
	}

	for _, q := range queries {
		go func(key, metric, stat string) {
			val, err := c.getMetricStatistic(
				ctx,
				ns,
				metric,
				stat,
				dims,
				start,
				end,
			)
			ch <- responseDTO{name: key, value: val, err: err}
		}(q.key, q.metric, q.stat)
	}

	hasData := false

	for range 3 {
		r := <-ch
		if r.err != nil {
			// 一樣單一指標失敗不會影響整體
			continue
		}
		if r.value > 0 {
			hasData = true
		}
		switch r.name {
		case "cpu":
			result.CPUUtilized = r.value
		case "mem":
			result.MemoryUtilized = r.value
		case "tasks":
			result.RunningTaskCount = r.value
		}
	}

	if !hasData {
		return fmt.Errorf("container insights namespace (%s) 沒有 datapoints", ns)
	}
	return nil
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("ap-east-2"), // 這個可以換成自己喜歡的 region
		config.WithSharedConfigProfile("<這個用自己的 sso profile>"),
	)
	if err != nil {
		log.Fatalf("load config error : %v\n", err)
		return
	}

	// ==============================
	cluster := "<cluster的名稱>比如supportive-gorilla-15sf6h" // ECS cluster
	namespace := "<namespace的名稱>比如workshop.local"        // cloud map cluster

	// Map: discoveryName -> actual ECS service name
	serviceNames := map[string]string{
		"api-blue":  "blue 的服務名稱",
		"api-green": "green 的服務名稱",
	}
	// ==============================

	collector := NewCollector(cfg, cluster, namespace, serviceNames)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		blueMetric, err := collector.Collect(ctx, "api-blue", 10)
		if err != nil {
			log.Printf("\033[031mcollect blueMetric error : %v\n\033[0m", err)
			continue
		}
		greenMetric, err := collector.Collect(ctx, "api-green", 10)
		if err != nil {
			log.Printf("\033[031mcollect greenMetric error : %v\n\033[0m", err)
			continue
		}
		log.Printf(
			"\n\033[094mblueService : %s, \n \033[032mgreenService : %s\033[0m\n",
			blueMetric.String(),
			greenMetric.String(),
		)
	}
}
