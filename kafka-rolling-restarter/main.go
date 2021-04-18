// Kafka rolling restarter that reads for URP and ISR metrics from Prometheus
// and then based on that, decides to restart kafka or wait further.
// Must be run on the node that has kafka running as systemd service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	// Number of active controllers sum total - one of the value of metric to trigger kafka service restart
	controllersSum = 1.0
	// Other key metrics sum total - one of the value of metric to trigger kafka service restart
	keyMetricsSum = 0.0
)

var (
	// URP metric defined as: sum of the rate of URP over one minute
	underReplicatedPartitionQuery = "sum(rate(prometheus_http_requests_total{code=\"200\",handler=\"/rules\"}[1m]))> 0"
	// ISR metric defined as: sum of the rate of ISR change over one minute
	inSyncReplicatedQuery = "sum(rate(prometheus_http_requests_total{code=\"200\",handler=\"/rules\"}[1m]))> 0"
	// broker traffic query as: sum of the rate of byte in + sum of the rate of byte out over 30 minutes
	//brokerTrafficQuery               = "sum(rate(prometheus_http_requests_total{code=\"200\",handler=\"/rules\"}[1m]))> 0"
	// sum of active controller count *should* be 1.0 in any stable kafka cluster.
	sumActiveControllerCountQuery = "sum(rate(prometheus_http_requests_total{code=\"200\",handler=\"/rules\"}[1m]))> 0"
	// poll interval from Prometheus
	prometheusQueryRangePollInterval = 5 * time.Second
	// How far to go to calculate the aggregated value of metrics
	prometheusQueryRangeStartAt = -time.Minute * 2
	// For zero downtime rolling restarts, we need to wait (metric aggregation interval) so they are available to fetch from Prometheus
	preStartSleepDuration = time.Second * 5 // 240
)

func main() {
	prometheusUrl := flag.String("p", "http://127.0.0.1:9090", "prometheus url: -p [prometheus_url]")
	restartKafkaService := flag.Bool("r", false, "restart kafka service (default false)")
	flag.Parse()

	// Give ample time for metrics to be available in Prometheus.
	fmt.Printf("Give ample time for metrics to be available in Prometheus. Waiting %s\n", preStartSleepDuration)
	time.Sleep(preStartSleepDuration)

	queryClient := newPromQueryClient(*prometheusUrl)
	for {
		var activeControllerSum, keyMetricsSum float64
		println()
		urp, err := queryClient.promQueryRange(context.Background(), underReplicatedPartitionQuery)
		if err != nil {
			fmt.Printf("error querying Prometheus: %v\n", err)
			os.Exit(1)
		}
		isr, err := queryClient.promQueryRange(context.Background(), inSyncReplicatedQuery)
		if err != nil {
			fmt.Printf("error querying Prometheus: %v\n", err)
			os.Exit(1)
		}
		acc, err := queryClient.promQueryRange(context.Background(), sumActiveControllerCountQuery)
		if err != nil {
			fmt.Printf("error querying Prometheus: %v\n", err)
			os.Exit(1)
		}
		activeControllerSum += queryClient.getActiveControllerSum(acc)
		keyMetricsSum += queryClient.getKeyMetricsSum(urp, isr)
		if isRestartRequired(activeControllerSum, keyMetricsSum) == true {
			fmt.Println("===============================")
			if *restartKafkaService == true {
				// local restart
				err := restart("kafka-server.service")
				if err != nil {
					log.Printf("restart of kafka service failed: %s", err)
					os.Exit(1)
				}
				break // we are done restarting this node's kafka service.
			}
			fmt.Println("sleeping a minute")
			time.Sleep(1 * time.Minute)
			fmt.Println("we are done")
			fmt.Println("===============================")
		} else {
			fmt.Printf("Kafka service is still replicating data so not restarting.. waiting for %s ..\n", prometheusQueryRangePollInterval)
		}
		fmt.Printf("key metrics sum = %.2f\n", keyMetricsSum)
		fmt.Printf("active controller metric sum = %.2f\n", activeControllerSum)
		time.Sleep(prometheusQueryRangePollInterval)

		// update time range
		queryClient.setTimeRange(
			time.Now().Add(prometheusQueryRangeStartAt),
			time.Now(),
			time.Minute,
		)
		println()
	}
}

type promQueryClient struct {
	apiClient v1.API
	timeRange v1.Range // The time range over which to aggregate the metric value
}

func newPromQueryClient(prometheusUrl string) *promQueryClient {
	client, err := api.NewClient(api.Config{
		Address: prometheusUrl,
	})
	if err != nil {
		fmt.Printf("error creating client: %v\n", err)
		os.Exit(1)
	}

	return &promQueryClient{
		apiClient: v1.NewAPI(client),
		timeRange: v1.Range{
			Start: time.Now().Add(prometheusQueryRangeStartAt),
			End:   time.Now(),
			Step:  time.Minute,
		},
	}
}

// setTimeRange sets the time range as we need sliding window interval for Prometheus to fetch
// the metrics for.
func (p *promQueryClient) setTimeRange(s, e time.Time, step time.Duration) {
	p.timeRange = v1.Range{
		Start: s,
		End:   e,
		Step:  step,
	}
}

func (p *promQueryClient) promQueryRange(ctx context.Context, q string) (model.Matrix, error) {
	fmt.Printf("running Prometheus query: %s\n", q)
	result, warnings, err := p.apiClient.QueryRange(ctx, q, p.timeRange)
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		return nil, err
	}
	m, ok := result.(model.Matrix)
	if !ok {
		return nil, errors.New("could not type assert to model.Matrix")
	}
	return m, nil
}

func (p *promQueryClient) getKeyMetricsSum(
	urp model.Matrix,
	isr model.Matrix,
) float64 {
	var sum float64
	for _, x := range urp {
		for _, v := range x.Values {
			fmt.Printf("%d,%.2f\n", v.Timestamp.UnixNano(), v.Value)
			sum += float64(v.Value)
		}
	}

	for _, x := range isr {
		for _, v := range x.Values {
			fmt.Printf("%d,%.2f\n", v.Timestamp.UnixNano(), v.Value)
			sum += float64(v.Value)
		}
	}
	return sum
}

func (p *promQueryClient) getActiveControllerSum(acc model.Matrix) float64 {
	var sum float64
	for _, x := range acc {
		for _, v := range x.Values {
			fmt.Printf("%d,%.2f\n", v.Timestamp.UnixNano(), v.Value)
			sum += float64(v.Value)
		}
	}
	return sum
}

func isRestartRequired(a, k float64) bool {
	return k == keyMetricsSum && a == controllersSum
}

//  restart via systemd
func restart(serviceName string) error {
	arg := "restart " + serviceName
	cmd := exec.Command("systemctl", arg)
	return cmd.Run()
	//fmt.Println("faking restart")
	//return nil
}
