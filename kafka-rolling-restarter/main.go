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
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	// URP metric defined as: sum of the rate of URP over one minute
	underReplicatedPartitionQuery    = "sum(rate(prometheus_http_requests_total{code=\"200\",handler=\"/rules\"}[1m]))> 0"
	// ISR metric defined as: sum of the rate of ISR change over one minute
	inSyncReplicatedQuery            = "sum(rate(prometheus_http_requests_total{code=\"200\",handler=\"/rules\"}[1m]))> 0"
	prometheusQueryRangePollInterval = 5 * time.Second
	// How far to go to calculate the aggregated value of URP and ISR.
	prometheusQueryRangeStartAt      = -time.Minute * 2
	// For zero downtime rolling restarts, we need to wait so metrics are available in Prometheus
	preStartSleepDuration            = time.Second * 240
)

func main() {
	prometheusUrl := flag.String("p", "http://127.0.0.1:9090", "prometheus url: -p [prometheus_url]")
	restartKafkaService := flag.Bool("r", false, "restart kafka service (default false)")
	flag.Parse()

	// Give ample time for metrics to be available in Prometheus.
	fmt.Printf("Giving ample time for metrics to be available in Prometheus. Waiting %s\n", preStartSleepDuration)
	time.Sleep(preStartSleepDuration)

	queryClient := newPromQueryClient(*prometheusUrl)
	var sum float64
	for {
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
		sum += queryClient.getAggregatedMetricValue(urp, isr)
		if sum == 0.0 {
			fmt.Println("===============================")
			if *restartKafkaService == true {
				// local restart
				err := restart("kafka-server.service")
				if err != nil {
					log.Printf("restart of kafka service failed: %s", err)
				}
				break // we are done restarting this node's kafka service.
			}
			fmt.Println("mock sleep for 360s")
			//time.Sleep(6 * time.Minute)
			fmt.Println("===============================")
		} else {
			fmt.Printf("Kafka service is still replicating data so not restarting.. waiting for %s ..\n", prometheusQueryRangePollInterval)
		}
		fmt.Printf("sum = %.2f\n", sum)
		time.Sleep(prometheusQueryRangePollInterval)

		// update time range
		queryClient.updateTimeRange(
			time.Now().Add(prometheusQueryRangeStartAt),
			time.Now(),
			time.Minute,
		)
		// reset
		sum = 0.0
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
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}
	v1api := v1.NewAPI(client)
	return &promQueryClient{
		apiClient: v1api,
		timeRange: v1.Range{
			Start: time.Now().Add(prometheusQueryRangeStartAt),
			End:   time.Now(),
			Step:  time.Minute,
		},
	}
}

func (p *promQueryClient) updateTimeRange(s, e time.Time, step time.Duration) {
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

func (p *promQueryClient) getAggregatedMetricValue(urp model.Matrix, isr model.Matrix) float64 {
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

//  restart via systemd
func restart(serviceName string) error {
	fmt.Printf("systemctl restart %s\n", serviceName)
	return nil
}
