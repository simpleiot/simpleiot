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

	// promMaxSeriesLimit is the largest series limit a node may configure.
	//
	// Points are current state on the node, and a node request encodes the
	// node and all of its points into a single NATS message. A scraped point
	// encodes to roughly 100 bytes, so 10000 of them reach about 1 MB, which
	// is the NATS max_payload SIOT runs with. Past that the store cannot
	// publish a reply at all: the request times out, and because the reply
	// carries a subtree rather than one node, every tree fetch that includes
	// the node fails rather than just the node itself. data.DecodePoints
	// refuses an array over 10000 for the same reason.
	//
	// This ceiling keeps a node's points near 300 KB, well clear of both
	// limits and leaving room for the rest of the subtree in the same reply.
	// A configured value above it is not honored, and the node reports it.
	promMaxSeriesLimit = 3000

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

// promReservedTypes are point types a scrape may not publish under their own
// name.
//
// Run feeds every point arriving on the node through data.MergePoints into the
// Metrics config struct, which matches on point type, so a metric named period
// would rewrite the scrape interval and one named disabled would switch the
// client off. Prometheus naming convention makes a collision unlikely -- a real
// metric is namespaced and unit suffixed, as in myapp_requests_total -- but the
// failure is quiet enough to be worth closing rather than documenting.
//
// A collision renames rather than drops the sample; see promRename.
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
	m.promRenamed = make(map[string]bool)
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

// promRename returns the point type to publish a reserved metric name under.
//
// Publishing the name as-is would merge into the node's own configuration and
// rewrite a setting, so the name has to change. An underscore is appended
// until it is free, which keeps the reading rather than dropping it, keeps the
// metric recognizable, and sorts it next to where it would otherwise be. One
// underscore is always enough to clear the reserved set, since no reserved
// name ends in one; the loop is for the case where the endpoint also serves
// the renamed name itself, so that two metrics never land on one point.
//
// The rename is logged once per name, since a metric colliding with a node
// setting is worth renaming at the source.
func (m *MetricsClient) promRename(name string, served map[string]bool) string {
	renamed := name + "_"

	for promReservedTypes[renamed] || served[renamed] {
		renamed += "_"
	}

	if !m.promRenamed[name] {
		m.promRenamed[name] = true
		log.Printf("Metrics: prometheus scrape publishing metric %q as %q, "+
			"since its name collides with a node configuration point type",
			name, renamed)
	}

	return renamed
}

// resolveMaxSeries returns the series limit to apply, along with a message
// describing a configured limit that could not be honored.
//
// A value above promMaxSeriesLimit is clamped rather than accepted, because
// accepting it breaks more than this node: once the node's points no longer
// fit in one NATS message, the store cannot answer any node request covering
// the subtree, and the UI stops loading. Keeping the failure inside the node
// is the whole point of having a limit.
func (m *MetricsClient) resolveMaxSeries() (int, string) {
	switch {
	case m.config.MaxSeries < 1:
		return promDefaultMaxSeries, ""

	case m.config.MaxSeries > promMaxSeriesLimit:
		msg := fmt.Sprintf(
			"maxSeries %v exceeds the limit of %v and is being ignored; "+
				"collecting %v series. Narrow the scrape with a metric "+
				"prefix, or split it across several nodes.",
			m.config.MaxSeries, promMaxSeriesLimit, promMaxSeriesLimit)

		if !m.promClampLogged {
			m.promClampLogged = true
			log.Println("Metrics:", msg)
		}

		return promMaxSeriesLimit, msg
	}

	return m.config.MaxSeries, ""
}

// promPrefixMatch reports whether a metric name passes the prefix filter. A
// sample is kept when it matches any configured prefix, so an application that
// namespaces its metrics under more than one name, or a node collecting a
// couple of subsystems from a larger exporter, needs one node rather than
// several.
//
// A filter with no entries collects everything. So does one whose entries are
// all empty, which is what a prefix added in the UI but not yet filled in looks
// like -- a half-typed entry should not quietly widen the filter to everything
// while the other entries are ignored.
func (m *MetricsClient) promPrefixMatch(name string) bool {
	var configured bool

	for _, p := range m.config.Prefixes {
		if p == "" {
			continue
		}

		configured = true

		if strings.HasPrefix(name, p) {
			return true
		}
	}

	return !configured
}

// promPoints turns samples into points, applying the prefix filter, the
// reserved names, the series cap, and the counter deltas. It returns the points
// to publish along with a message describing anything the operator should know
// about, which is empty when the scrape was clean.
func (m *MetricsClient) promPoints(samples []sample) (data.Points, string) {
	keep := make([]sample, 0, len(samples))

	// every name the endpoint served, so a rename cannot land on a metric the
	// endpoint is already reporting
	served := make(map[string]bool, len(samples))
	for _, s := range samples {
		served[s.name] = true
	}

	for _, s := range samples {
		if !m.promPrefixMatch(s.name) {
			continue
		}

		if promReservedTypes[s.name] {
			s.name = m.promRename(s.name, served)
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

	limit, errMsg := m.resolveMaxSeries()

	if len(keep) > limit {
		truncated := fmt.Sprintf(
			"scrape truncated: %v series exceeds maxSeries %v", len(keep), limit)
		log.Println("Metrics:", truncated)

		if errMsg != "" {
			errMsg += "; " + truncated
		} else {
			errMsg = truncated
		}

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
