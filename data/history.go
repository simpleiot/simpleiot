package data

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Regular expressions to sanitize Flux queries
var (
	validField = regexp.MustCompile(`^[a-zA-Z\d]+$`)
	validOp    = regexp.MustCompile(`^(>|>=|<|<=|between|=)$`)
	validValue = regexp.MustCompile(`^[^\\"']*$`)
)

// HistoryQuery is a query that is sent to an Influx DB client to request
// historical points
type HistoryQuery struct {
	Start           time.Time      `json:"start"`
	Stop            time.Time      `json:"stop"`
	TagFilters      TagFilters     `json:"tagFilters"`
	AggregateWindow *time.Duration `json:"aggregateWindow"`
}

// Flux generates a Flux query for the HistoryQuery. Returns an error if tag
// filters could not be sanitized.
func (qry HistoryQuery) Flux(bucket, measurement string) (string, error) {
	sb := &strings.Builder{}
	fmt.Fprintf(
		sb,
		`data = from(bucket: "%v")
			|> range(start: %v, stop: %v)
			|> filter(fn: (r) =>
				r._measurement == "%v" and
				r._field == "value"`,
		bucket,
		qry.Start.Format(time.RFC3339),
		qry.Stop.Format(time.RFC3339),
		measurement,
	)
	// Add filters
	qty.TagFilters.Flux(sb)
	sb.WriteString(")\n")

	// Add aggregation (or not)
	if qry.AggregateWindow == nil {
		sb.WriteString("data")
	} else {
		fmt.Fprintf(
			sb,
			`data
				|> window(every: %vs, createEmpty: false)
				|> reduce(
					identity: {
						min: math.mInf(sign: 1),
						max: math.mInf(sign: -1),
						count: 0,
						sum: 0.0,
					}, fn: (r, accumulator) => ({
						min: if r._value < accumulator.min then r._value else accumulator.min,
						max: if r._value > accumulator.max then r._value else accumulator.max,
						count: accumulator.count + 1,
						sum: accumulator.sum + r._value,
					})
				)
				|> map(fn: (r) => ({r with mean: r.sum / float(v: r.count)}))
				|> duplicate(column: "_stop", as: "_time")
				|> window(every: inf)
			`,
			qry.AggregateWindow.Seconds(),
		)
	}
}

// TagFilters further reduces Influx query results by tag
type TagFilters map[string]string

// Flux writes a clause for a Flux query (to be added to the filter function
// body) to the specified string.Builder. Returns an error if a tag filter
// could not be sanitized.
func (t TagFilters) Flux(sb *string.Builder) error {
	for k, v := range t {
		// Sanitize input
		if !validField.MatchString(k) {
			return errors.New("invalid tag filter " + k)
		}
		if !validValue.MatchString(v) {
			return errors.New("invalid tag filter value for " + k)
		}
		fmt.Fprintf(sb, ` and r.%s == "%s"`, k, v)
	}
	return nil
}

// HistoryResults is the result of a history query. The result includes an
// optional error string along with a slice of either points or aggregated
// points.
type HistoryResults struct {
	ErrorMessage     string                   `json:"error,omitempty"`
	Points           []HistoryPoint           `json:"points,omitempty"`
	AggregatedPoints []HistoryAggregatedPoint `json:"aggregatedPoints,omitempty"`
}

// HistoryPoint is a point returned by a non-aggregated history query
type HistoryPoint struct {
	Time time.Time `json:"time"`

	// NodeTags (i.e. "id", "description", "type", and others)
	NodeTags map[string]string `json:"nodeTags"`

	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty"`

	// Key is used to allow a group of points to represent a map or array
	Key string `json:"key,omitempty"`

	// Instantaneous analog or digital value of the point.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// Optional text value of the point for data that is best represented
	// as a string rather than a number.
	Text string `json:"text,omitempty"`
}

// HistoryAggregatedPoint is a group of aggregated points of a history query
type HistoryAggregatedPoint struct {
	Time time.Time `json:"time"`

	// NodeTags (i.e. "id", "description", "type", and others)
	NodeTags map[string]string `json:"nodeTags"`

	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty"`

	// Key is used to allow a group of points to represent a map or array
	Key string `json:"key,omitempty"`

	// Arithmetic mean of the point values in the aggregated window
	Mean float64 `json:"mean"`

	// Minimum point value in the aggregated window
	Min float64 `json:"min"`

	// Maximum point value in the aggregated window
	Max float64 `json:"max"`

	// Count is the number of points in the aggregated window
	Count float64 `json:"count"`
}
