package data

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api"
)

// Regular expressions to sanitize Flux queries
var (
	validField = regexp.MustCompile(`^[a-zA-Z\d.]+$`)
	validValue = regexp.MustCompile(`^[^\\"']*$`)
	// validOp    = regexp.MustCompile(`^(>|>=|<|<=|between|=)$`)
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
		`import "math"
		data = from(bucket: "%v")
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
	err := qry.TagFilters.Flux(sb)
	if err != nil {
		return "", err
	}
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

	return sb.String(), nil
}

// Execute generates the Flux query and executes it, populating the specified
// HistoryResults
func (qry HistoryQuery) Execute(
	ctx context.Context,
	api api.QueryAPI,
	bucket, measurement string,
	results *HistoryResults,
) {
	query, err := qry.Flux(bucket, measurement)
	if err != nil {
		results.ErrorMessage = "generating query: " + err.Error()
		return
	}
	rawResults, err := api.Query(ctx, query)
	if err != nil {
		results.ErrorMessage = "executing query: " + err.Error()
		return
	}

	// Populate results
	for rawResults.Next() {
		var (
			ts                            time.Time
			typeTag, keyTag, textField    string
			valueField                    float64
			meanField, minField, maxField float64
			countField                    int64
		)
		nodeTags := make(map[string]string)
		for key, val := range rawResults.Record().Values() {
			var ok bool
			switch key {
			case "_time":
				ts, ok = val.(time.Time)
				if !ok {
					results.ErrorMessage = "error decoding field: time"
					return
				}
			case "type":
				typeTag, ok = val.(string)
				if !ok {
					results.ErrorMessage = "error decoding tag: type"
					return
				}
			case "key":
				keyTag, ok = val.(string)
				if !ok {
					results.ErrorMessage = "error decoding tag: key"
					return
				}
			case "text":
				textField, ok = val.(string)
				if !ok {
					results.ErrorMessage = "error decoding field: text"
					return
				}
			case "value":
				valueField, ok = val.(float64)
				if !ok {
					results.ErrorMessage = "error decoding field: value"
					return
				}
			case "mean":
				meanField, ok = val.(float64)
				if !ok {
					results.ErrorMessage = "error decoding field: mean"
					return
				}
			case "min":
				minField, ok = val.(float64)
				if !ok {
					results.ErrorMessage = "error decoding field: min"
					return
				}
			case "max":
				maxField, ok = val.(float64)
				if !ok {
					results.ErrorMessage = "error decoding field: max"
					return
				}
			case "count":
				countField, ok = val.(int64)
				if !ok {
					results.ErrorMessage = "error decoding field: count"
					return
				}
			default:
				if strings.HasPrefix(key, "node.") {
					tag, ok := val.(string)
					if !ok {
						results.ErrorMessage = "error decoding tag: " + key
						return
					}
					nodeTags[key] = tag
				}
			}
		}

		if qry.AggregateWindow == nil {
			hp := HistoryPoint{
				Time:     ts,
				NodeTags: nodeTags,
				Type:     typeTag,
				Key:      keyTag,
				Value:    valueField,
				Text:     textField,
			}
			results.Points = append(results.Points, hp)
		} else {
			hap := HistoryAggregatedPoint{
				Time:     ts,
				NodeTags: nodeTags,
				Type:     typeTag,
				Key:      keyTag,
				Mean:     meanField,
				Min:      minField,
				Max:      maxField,
				Count:    countField,
			}
			results.AggregatedPoints = append(results.AggregatedPoints, hap)
		}
	}
}

// TagFilters further reduces Influx query results by tag
type TagFilters map[string]string

// Flux writes a clause for a Flux query (to be added to the filter function
// body) to the specified string.Builder. Returns an error if a tag filter
// could not be sanitized.
func (t TagFilters) Flux(sb *strings.Builder) error {
	for k, v := range t {
		// Sanitize input
		if !validField.MatchString(k) {
			return errors.New("invalid tag filter " + k)
		}
		if !validValue.MatchString(v) {
			return errors.New("invalid tag filter value for " + k)
		}
		fmt.Fprintf(sb, ` and r["%s"] == "%s"`, k, v)
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
	Count int64 `json:"count"`
}
