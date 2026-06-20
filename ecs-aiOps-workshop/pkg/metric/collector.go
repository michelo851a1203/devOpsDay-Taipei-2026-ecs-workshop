// Package metric for collecting cloudwatch data
package metric

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type ServiceMetric struct {
	ServiceName string
	TimeStamp   time.Time
	WindowMin   int

	// service connect
	RequestCount      float64
	ErrorCount        float64
	ErrorRate         float64
	P99Latency        float64
	P50Latency        float64
	AvgLatency        float64
	ActiveConnections float64
	NewConnections    float64

	// container insight
	CPUUtilized      float64
	MemoryUtilized   float64
	RunningTaskCount float64

	// data
	SCDataAvailable bool
	CIDataAvailable bool
}

func (m *ServiceMetric) String() string {
	sources := []string{}
	if m.SCDataAvailable {
		sources = append(sources, "SC")
	}
	if m.CIDataAvailable {
		sources = append(sources, "CI")
	}
	source := strings.Join(sources, "+")

	return fmt.Sprintf(
		"[%s][%s]: reqs=%0.f, errs=%0.f, errRate=%.2f%% cpu=%.3f mem=%.3f",
		source,
		m.ServiceName,
		m.RequestCount,
		m.ErrorCount,
		m.ErrorRate,
		m.CPUUtilized,
		m.MemoryUtilized,
	)
}

type ResponseDTO struct {
	name  string
	value float64
	err   error
}

type basic struct {
	key, metric, stat string
}

type Collector struct {
	cw           *cloudwatch.Client
	cluster      string
	namespace    string
	serviceNames map[string]string

	// ALB (AWS/ApplicationELB) 維度值
	albLoadBalancer string // 例如 app/demo-alb/0df4f07d47294b76
	albTargetGroup  string // 例如 targetgroup/demo-tg/f8719e5c5da979b0
}

func NewCollector(cfg aws.Config, cluster, namespace string, serviceNames map[string]string, albLoadBalancer, albTargetGroup string) *Collector {
	return &Collector{
		cw:              cloudwatch.NewFromConfig(cfg),
		cluster:         cluster,
		namespace:       namespace,
		serviceNames:    serviceNames,
		albLoadBalancer: albLoadBalancer,
		albTargetGroup:  albTargetGroup,
	}
}

// ALBMetric 是整個 target group 的彙總指標 (blue+green 混在一起,無法分版)。
type ALBMetric struct {
	RequestCount  float64 // 整體請求數
	Error5XXCount float64 // HTTPCode_Target_5XX_Count:target(應用)回的 5XX
	ELB5XXCount   float64 // HTTPCode_ELB_5XX_Count:ALB 自己產生的 5XX (target 全掛/timeout)
	ErrorRate     float64 // 整體錯誤率 = 5XX / RequestCount * 100
	P50Latency    float64 // TargetResponseTime p50 (秒)
	P99Latency    float64 // TargetResponseTime p99 (秒)
	Available     bool
}

func (m *ALBMetric) String() string {
	return fmt.Sprintf(
		"[ALB整體]: reqs=%.0f, 5xx=%.0f, elb5xx=%.0f, errRate=%.2f%%, p50=%.1fms, p99=%.1fms",
		m.RequestCount, m.Error5XXCount, m.ELB5XXCount, m.ErrorRate,
		m.P50Latency*1000, m.P99Latency*1000,
	)
}

// CollectALB 抓 ALB (AWS/ApplicationELB) 的整體流量指標,作為 rollback/proceed 的主要判斷依據。
func (c *Collector) CollectALB(ctx context.Context, windowMin int) (*ALBMetric, error) {
	now := time.Now()
	start := now.Add(-time.Duration(windowMin) * time.Minute)
	ns := "AWS/ApplicationELB"
	dims := []types.Dimension{
		{Name: aws.String("LoadBalancer"), Value: aws.String(c.albLoadBalancer)},
		{Name: aws.String("TargetGroup"), Value: aws.String(c.albTargetGroup)},
	}

	result := &ALBMetric{}

	req, err := c.getStaticsMetrics(ctx, ns, "RequestCount", "Sum", dims, start, now)
	if err != nil {
		return nil, fmt.Errorf("[ALB] RequestCount error: %w", err)
	}
	result.RequestCount = req

	if v, err := c.getStaticsMetrics(ctx, ns, "HTTPCode_Target_5XX_Count", "Sum", dims, start, now); err == nil {
		result.Error5XXCount = v
	}
	if v, err := c.getStaticsMetrics(ctx, ns, "HTTPCode_ELB_5XX_Count", "Sum", dims, start, now); err == nil {
		result.ELB5XXCount = v
	}
	if v, err := c.getPercentile(ctx, ns, "TargetResponseTime", "p50", dims, start, now); err == nil {
		result.P50Latency = v
	}
	if v, err := c.getPercentile(ctx, ns, "TargetResponseTime", "p99", dims, start, now); err == nil {
		result.P99Latency = v
	}

	if result.RequestCount > 0 {
		result.ErrorRate = result.Error5XXCount / result.RequestCount * 100
	}
	result.Available = true
	return result, nil
}

func (c *Collector) Collect(ctx context.Context, discoveryName string, windowMin int) (*ServiceMetric, error) {
	now := time.Now()
	start := now.Add(-time.Duration(windowMin) * time.Minute)
	result := &ServiceMetric{
		ServiceName: discoveryName,
		TimeStamp:   now,
		WindowMin:   windowMin,
	}
	ecsServiceName, ok := c.serviceNames[discoveryName]
	if !ok {
		return nil, fmt.Errorf("no ECS Service name for discovery name : %s", discoveryName)
	}

	scDims := []types.Dimension{
		{Name: aws.String("ClusterName"), Value: aws.String(c.cluster)},
		{Name: aws.String("ServiceName"), Value: aws.String(ecsServiceName)},
		{Name: aws.String("DiscoveryName"), Value: aws.String(discoveryName)},
	}

	// HTTP 狀態碼類指標 (HTTPCode_Target_*XX_Count) 只發佈在 TargetDiscoveryName 維度。
	// 注意:它的 {TargetDiscoveryName, ServiceName, ClusterName} 組合裡的 ServiceName 是
	// 「發出請求的 client 服務」,不是目標服務,所以用目標的 ecsServiceName 查不到。要拿到打到此
	// discovery name 的所有錯誤,只能用單一維度 TargetDiscoveryName。
	scTargetDims := []types.Dimension{
		{Name: aws.String("TargetDiscoveryName"), Value: aws.String(discoveryName)},
	}

	ciDims := []types.Dimension{
		{Name: aws.String("ClusterName"), Value: aws.String(c.cluster)},
		{Name: aws.String("ServiceName"), Value: aws.String(ecsServiceName)},
	}

	scErr := c.CollectServiceConnect(ctx, result, scDims, scTargetDims, start, now)
	if scErr != nil {
		log.Printf("[Collector] Service Connect 無數據 (ns: %s),(service: %s): %v\n", c.namespace, discoveryName, scErr)
	} else {
		result.SCDataAvailable = true
	}

	ciErr := c.CollectContainerInsight(ctx, result, ciDims, start, now)
	if ciErr != nil {
		log.Printf("[Collector] Container Insights 無數據 (%s) : %v\n", discoveryName, ciErr)
	} else {
		result.CIDataAvailable = true
	}

	if !result.SCDataAvailable && !result.CIDataAvailable {
		return result, fmt.Errorf("所有來源都沒有資料 (serviceName: %s)(cluster: %s)", discoveryName, c.cluster)
	}

	return result, nil
}

func (c *Collector) CollectServiceConnect(ctx context.Context, result *ServiceMetric, dims, targetDims []types.Dimension, start, end time.Time) error {
	ns := "AWS/ECS"

	ch := make(chan ResponseDTO, 4)
	basics := []basic{
		{"request_count", "RequestCount", "Sum"},
		{"error_count", "HTTPCode_Target_5XX_Count", "Sum"},
		{"active_connections", "ActiveConnectionCount", "Sum"},
		{"new_connections", "NewConnectionCount", "Sum"},
	}

	for _, q := range basics {
		// HTTPCode_Target_5XX_Count 用 TargetDiscoveryName 維度,其餘指標用 DiscoveryName
		qDims := dims
		if q.key == "error_count" {
			qDims = targetDims
		}
		go func(key, metric, stat string, d []types.Dimension) {
			res, err := c.getStaticsMetrics(ctx, ns, metric, stat, d, start, end)
			ch <- ResponseDTO{name: key, value: res, err: err}
		}(q.key, q.metric, q.stat, qDims)
	}

	hasData := false
	for range 4 {
		r := <-ch
		if r.err != nil {
			continue
		}
		hasData = true
		switch r.name {
		case "request_count":
			result.RequestCount = r.value
		case "error_count":
			result.ErrorCount = r.value
		case "active_connections":
			result.ActiveConnections = r.value
		case "new_connections":
			result.NewConnections = r.value
		}
	}
	if !hasData {
		return fmt.Errorf("service connect (AWS/ECS) 無DataPoints")
	}
	if result.RequestCount > 0 {
		result.ErrorRate = result.ErrorCount / result.RequestCount * 100
	}
	return nil
}

func (c *Collector) CollectContainerInsight(ctx context.Context, result *ServiceMetric, dims []types.Dimension, start, end time.Time) error {
	ns := "ECS/ContainerInsights"

	ch := make(chan ResponseDTO, 3)
	basics := []basic{
		{"cpu", "CpuUtilized", "Average"},
		{"mem", "MemoryUtilized", "Average"},
		{"tasks", "RunningTaskCount", "Average"},
	}

	for _, q := range basics {
		go func(key, metric, stat string) {
			res, err := c.getStaticsMetrics(
				ctx,
				ns,
				metric,
				stat,
				dims,
				start,
				end,
			)
			ch <- ResponseDTO{
				name:  key,
				value: res,
				err:   err,
			}
		}(q.key, q.metric, q.stat)
	}

	hasData := false
	for range 3 {
		r := <-ch
		if r.err != nil {
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
		return fmt.Errorf("container insights namespace (%s) 無DataPoints", ns)
	}

	return nil
}

// base method
func (c *Collector) getStaticsMetrics(
	ctx context.Context,
	namespace,
	metricName,
	stat string,
	dims []types.Dimension,
	start,
	end time.Time,
) (float64, error) {
	staticType := types.StatisticSum
	switch stat {
	case "Maximium":
		staticType = types.StatisticMaximum
	case "Minimium":
		staticType = types.StatisticMinimum
	case "Average":
		staticType = types.StatisticAverage
	case "Sum":
		staticType = types.StatisticSum
	}
	resp, err := c.cw.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Dimensions: dims,
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(60),
		Statistics: []types.Statistic{types.Statistic(staticType)},
	})
	if err != nil {
		return 0, err
	}
	if len(resp.Datapoints) == 0 {
		return 0, nil
	}
	// 跨整個 window 彙總所有 datapoint,而不是只取最新一分鐘,
	// 否則 Sum 類計數(請求數/5XX)會只反映一分鐘,且分子分母可能落在不同分鐘導致錯誤率失真。
	switch stat {
	case "Maximium":
		max := aws.ToFloat64(resp.Datapoints[0].Maximum)
		for _, dp := range resp.Datapoints {
			if v := aws.ToFloat64(dp.Maximum); v > max {
				max = v
			}
		}
		return max, nil
	case "Minimium":
		min := aws.ToFloat64(resp.Datapoints[0].Minimum)
		for _, dp := range resp.Datapoints {
			if v := aws.ToFloat64(dp.Minimum); v < min {
				min = v
			}
		}
		return min, nil
	case "Average":
		sum := 0.0
		for _, dp := range resp.Datapoints {
			sum += aws.ToFloat64(dp.Average)
		}
		return sum / float64(len(resp.Datapoints)), nil
	case "Sum":
		fallthrough
	default:
		sum := 0.0
		for _, dp := range resp.Datapoints {
			sum += aws.ToFloat64(dp.Sum)
		}
		return sum, nil
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

	res, err := c.cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("pct"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String(metricName),
						Dimensions: metricDims,
					},
					Period: aws.Int32(60),
					Stat:   aws.String(percentile),
				},
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("[getpercentile] error : %w", err)
	}
	if len(res.MetricDataResults) == 0 || len(res.MetricDataResults[0].Values) == 0 {
		return 0, nil
	}
	return res.MetricDataResults[0].Values[0], nil
}
