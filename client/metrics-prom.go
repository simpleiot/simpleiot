package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

const (
	// promDefaultMaxSeries bounds a scrape when the node does not set a limit.
	// A client_golang default registry plus an application's own metrics runs
	// 50-150 series, so this leaves headroom for the case the client is meant
	// to serve while bounding an endpoint that turns out to expose far more.
	promDefaultMaxSeries = 200

	// promMaxBody bounds how much of a response is read. An endpoint that
	// serves more than this is not one this client can usefully collect.
	promMaxBody = 8 * 1024 * 1024

	// promMaxTimeout caps the scrape timeout. The timeout is otherwise half
	// the scrape period, so that a slow endpoint cannot leave scrapes
	// overlapping.
	promMaxTimeout = 10 * time.Second

	// promAccept asks for the text exposition format. client_golang serves
	// protobuf or OpenMetrics when asked for them, and asking for text keeps
	// the input shape stable.
	promAccept = "text/plain;version=0.0.4;q=1,*/*;q=0.1"
)

// promReservedTypes are point types a scrape may not publish.
//
// Run feeds every point arriving on the node through data.MergePoints into the
// Metrics config struct, which matches on point type, so a metric named period
// would rewrite the scrape interval and one named disabled would switch the
// client off. Prometheus naming convention makes a collision unlikely -- a real
// metric is namespaced and unit suffixed, as in myapp_requests_total -- but the
// failure is quiet enough to be worth closing rather than documenting.
var promReservedTypes = map[string]bool{
	// Metrics config fields
	data.PointTypeDescription:  true,
	data.PointTypeType:         true,
	data.PointTypeName:         true,
	data.PointTypePeriod:       true,
	data.PointTypeURI:          true,
	data.PointTypePrefix:       true,
	data.PointTypeCounterDelta: true,
	data.PointTypeMaxSeries:    true,

	// node level points the manager and the UI depend on
	data.PointTypeTag:             true,
	data.PointTypeDisabled:        true,
	data.PointTypeError:           true,
	data.PointTypeErrorCount:      true,
	data.PointTypeErrorCountReset: true,
	data.PointTypeConnected:       true,
	data.PointTypeDebug:           true,
	data.PointTypeLog:             true,
	data.PointTypeNodeType:        true,
}

// checkPromDefaults publishes the scrape defaults to the node so they are
// visible and editable in the UI rather than implicit, the way checkPeriod does
// for the sample period.
//
// A node that has never been configured has no maxSeries point, which reads
// back as zero. That is taken as the signal to publish both defaults, since
// counterDelta is a bool and an absent point is indistinguishable from one
// explicitly set false. Once maxSeries is set the defaults are left alone, so a
// later counterDelta of false stays off.
func (m *MetricsClient) checkPromDefaults() {
	if m.config.MaxSeries >= 1 {
		return
	}

	m.config.MaxSeries = promDefaultMaxSeries
	m.config.CounterDelta = true

	pts := data.Points{
		data.NewPointFloat(data.PointTypeMaxSeries, "", float64(m.config.MaxSeries)),
		data.NewPointFloat(data.PointTypeCounterDelta, "",
			data.BoolToFloat(m.config.CounterDelta)),
	}

	if err := SendNodePoints(m.nc, m.config.ID, pts, false); err != nil {
		log.Println("Metrics: error sending prometheus defaults:", err)
	}
}

// promReset drops what the client remembers about an endpoint. Counter values
// are only meaningful relative to the endpoint they came from.
func (m *MetricsClient) promReset() {
	m.promCounters = make(map[string]float64)
	m.promSkipped = make(map[string]bool)
}

// promClient is shared across metrics nodes. Its timeout is set per request,
// since each node has its own period.
var promClient = &http.Client{}

// promPeriodic scrapes the configured endpoint and publishes what it carried
func (m *MetricsClient) promPeriodic() {
	if m.config.URI == "" {
		m.promSetError("no URI configured")
		return
	}

	samples, badLines, err := m.promScrape()
	if err != nil {
		// stale readings are worse than absent ones, so nothing is
		// published. The counter state is left alone so that the first
		// delta after a recovery covers the whole gap rather than
		// reporting a false reset.
		log.Printf("Metrics: error scraping %v: %v", m.config.URI, err)
		m.promSetError(err.Error())

		return
	}

	if badLines > 0 {
		log.Printf("Metrics: %v unparsable lines from %v", badLines, m.config.URI)
	}

	pts, errMsg := m.promPoints(samples)

	if err := SendNodePoints(m.nc, m.config.ID, pts, false); err != nil {
		log.Println("Metrics: error sending points:", err)
	}

	m.promSetError(errMsg)
}

// promScrape fetches the endpoint and parses what it served
func (m *MetricsClient) promScrape() ([]sample, int, error) {
	timeout := time.Duration(m.config.Period) * time.Second / 2
	if timeout <= 0 || timeout > promMaxTimeout {
		timeout = promMaxTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.config.URI, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Accept", promAccept)

	// Accept-Encoding is deliberately not set: http.Transport adds gzip and
	// decompresses the response transparently.

	resp, err := promClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("scrape returned %v", resp.Status)
	}

	return parseExpositionLimited(resp.Body)
}

// parseExpositionLimited parses a response body, reading no more of it than
// promMaxBody
func parseExpositionLimited(r io.Reader) ([]sample, int, error) {
	return parseExposition(io.LimitReader(r, promMaxBody))
}

// promPoints turns samples into points, applying the prefix filter, the
// reserved names, the series cap, and the counter deltas. It returns the points
// to publish along with a message describing anything the operator should know
// about, which is empty when the scrape was clean.
func (m *MetricsClient) promPoints(samples []sample) (data.Points, string) {
	keep := make([]sample, 0, len(samples))

	for _, s := range samples {
		if m.config.Prefix != "" && !strings.HasPrefix(s.name, m.config.Prefix) {
			continue
		}

		if promReservedTypes[s.name] {
			if !m.promSkipped[s.name] {
				m.promSkipped[s.name] = true
				log.Printf("Metrics: prometheus scrape skipping metric %q, "+
					"which collides with a node configuration point type", s.name)
			}

			continue
		}

		keep = append(keep, s)
	}

	// sorting before the cap means the same series survive each scrape rather
	// than flapping with whatever order the endpoint served
	sort.Slice(keep, func(a, b int) bool {
		if keep[a].name != keep[b].name {
			return keep[a].name < keep[b].name
		}

		return keep[a].key < keep[b].key
	})

	var errMsg string

	limit := m.config.MaxSeries
	if limit < 1 {
		limit = promDefaultMaxSeries
	}

	if len(keep) > limit {
		errMsg = fmt.Sprintf("scrape truncated: %v series exceeds maxSeries %v",
			len(keep), limit)
		log.Println("Metrics:", errMsg)

		keep = keep[:limit]
	}

	pts := make(data.Points, 0, len(keep))

	// counters that are no longer served, because the application dropped a
	// label value or stopped exporting a metric, are forgotten rather than
	// held forever
	seen := make(map[string]float64, len(keep))

	for _, s := range keep {
		pts = append(pts, data.NewPointFloat(s.name, s.key, s.val))

		if !s.counter {
			continue
		}

		id := s.counterID()

		prev, have := m.promCounters[id]
		seen[id] = s.val

		if !m.config.CounterDelta || !have {
			// nothing to subtract from on the first scrape of a series
			continue
		}

		delta := s.val - prev
		if s.val < prev {
			// a counter that went backwards means the process restarted,
			// so the current value is the count since that restart
			delta = s.val
		}

		pts = append(pts,
			data.NewPointFloat(s.name+data.CounterDeltaSuffix, s.key, delta))
	}

	m.promCounters = seen

	return pts, errMsg
}

// counterID identifies a series in the counter state map. The separator cannot
// appear in either field, so two series cannot share an entry.
func (s sample) counterID() string {
	return s.name + "\x00" + s.key
}

// promSetError publishes an error point when the message has changed, so a
// persistent failure is not resent every period. An empty message clears a
// previously published error.
func (m *MetricsClient) promSetError(msg string) {
	if msg == m.promError {
		return
	}

	m.promError = msg

	err := SendNodePoint(m.nc, m.config.ID,
		data.NewPointString(data.PointTypeError, "", msg), false)
	if err != nil {
		log.Println("Metrics: error sending error point:", err)
	}
}
