package mon_service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Q struct {
	errors   chan error
	results  chan *DNSResult
	lg       zerolog.Logger
	intv     time.Duration
	cacheTTL time.Duration
	mu       sync.Mutex
}

type DNSResult struct {
	answers  []string
	question string
	qtype    string
}

var (
	cache = make(map[string]*DNSResult)
)

func New(lg zerolog.Logger, interval string) (*Q, error) {
	intv, err := time.ParseDuration(interval)
	if err != nil {
		return nil, err
	}

	return &Q{
		errors:   make(chan error),
		results:  make(chan *DNSResult),
		lg:       lg.With().Str("module", "mon_service").Timestamp().Logger(),
		intv:     intv,
		cacheTTL: 20 * time.Second,
	}, nil
}

func (q *Q) dnsQuery(queryType string, record string) *DNSResult {
	var result []string
	switch queryType {
	case "A":
		ips, err := net.LookupIP(record)
		if err != nil {
			q.errors <- fmt.Errorf("lookup ip failed: %w", err)
		}
		for _, ip := range ips {
			r := strings.TrimSpace(ip.String())
			r = strings.Trim(r, "\n")
			result = append(result, r)
		}
	case "CNAME":
		cname, err := net.LookupCNAME(record)
		if err != nil {
			q.errors <- fmt.Errorf("lookup cname failed: %w", err)
		}

		result = append(result, cname)
	case "TXT":
		txt, err := net.LookupTXT(record)
		if err != nil {
			q.errors <- fmt.Errorf("lookup txt failed: %w", err)
		}
		for _, t := range txt {
			result = append(result, t)
		}
	case "MX":
		mx, err := net.LookupMX(record)
		if err != nil {
			q.errors <- fmt.Errorf("lookup mx failed: %w", err)
		}
		for _, m := range mx {
			result = append(result, fmt.Sprintf("%d %s", m.Pref, m.Host))
		}
	case "NS":
		ns, err := net.LookupNS(record)
		if err != nil {
			q.errors <- fmt.Errorf("lookup ns failed: %w", err)
		}
		for _, n := range ns {
			result = append(result, n.Host)
		}
	}

	//
	return &DNSResult{
		question: record,
		answers:  result,
		qtype:    queryType,
	}
}

func (q *Q) Start(ctx context.Context, rec []string, qryType string) error {
	defer ctx.Done()

	var wg sync.WaitGroup
	errDone := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		q.lg.Info().Msg("monitoring errors..")
		// errors are now monitored. blocking and waiting on errors channel
		q.monitorErrors(ctx)
		errDone <- struct{}{}
	}()

	resultDone := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		q.lg.Info().Msg("monitoring DNS results..")
		// results are now monitored. blocking and waiting on results channel
		q.processQueryResults(ctx)
		resultDone <- struct{}{}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(q.cacheTTL)
		for {
			select {
			case <-t.C:
				q.mu.Lock()
				q.lg.Info().Msg("flushing entire cache..")
				cache = make(map[string]*DNSResult)
				q.mu.Unlock()
			}
		}
	}()

	// blocking
	q.lg.Info().Msg("running monitoring service..")
	if err := q.runDNSMonitoring(ctx, rec, qryType); err != nil {
		q.lg.Error().Err(err)
	}

	<-errDone
	<-resultDone
	close(q.errors)
	close(q.results)

	wg.Wait()
	return nil
}

// blocking
func (q *Q) monitorErrors(ctx context.Context) {
	defer ctx.Done()
	for e := range q.errors {
		q.lg.Debug().Msgf("error: %s", e.Error())
	}
}

// blocking
func (q *Q) processQueryResults(ctx context.Context) {
	defer ctx.Done()
	for a := range q.results {
		q.lg.Debug().Msgf("%s %s %s", a.question, a.qtype, a.answers)
	}
}

// blocking
func (q *Q) processDNSRecordChange(ctx context.Context, d *DNSResult) {
	defer ctx.Done()

	// check if d exists in cache
	// if it *does not* exist in the cache, then ADD it to the cache.
	// finally, return.
	val, ok := cache[d.question]
	if !ok {
		q.mu.Lock()
		cache[d.question] = d
		q.mu.Unlock()
		return
	}

	if val.question == d.question && val.qtype == d.qtype {
		if len(val.answers) != len(d.answers) {
			return
		}
		match := len(val.answers)
		for s := range val.answers {
			for j := range d.answers {
				if val.answers[s] == d.answers[j] {
					match -= 1
				}
			}
		}
		// dns record's answer has changed. log it.
		if match != 0 {
			q.lg.Info().Msgf(
				"detected change in record:"+
					"[%s %s %s] -> [%s %s %s]",
				val.question, val.qtype, val.answers,
				d.question, d.qtype, d.answers)
		}
	}
}

// blocking
func (q *Q) runDNSMonitoring(ctx context.Context, records []string, qry string) error {
	t := time.NewTicker(q.intv)
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
			for _, r := range records {
				if !strings.HasPrefix(r, "#") {
					res := q.dnsQuery(qry, r)
					q.processDNSRecordChange(ctx, res)
					q.results <- res
				}
			}
			q.lg.Info().Msgf("cache update finished..")
		}
	}
}
