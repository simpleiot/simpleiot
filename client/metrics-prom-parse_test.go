package client

import (
	"os"
	"strings"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

// findSample returns the sample with the given name and key
func findSample(samples []sample, name, key string) (sample, bool) {
	for _, s := range samples {
		if s.name == name && s.key == key {
			return s, true
		}
	}

	return sample{}, false
}

// The cases below are the worked examples from the plan
// (plans/2026-08-11-prometheus-scrape-metrics.md), which were chosen to cover
// what the mapping can get wrong.
func TestParseExposition(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		name string
		key  string
		val  float64
	}{
		{
			desc: "sample with no labels gets an empty key",
			in:   "# TYPE myapp_queue_depth gauge\nmyapp_queue_depth 17\n",
			name: "myapp_queue_depth",
			key:  "",
			val:  17,
		},
		{
			desc: "single label",
			in:   `myapp_hits{code="200"} 5`,
			name: "myapp_hits",
			key:  "code=200",
			val:  5,
		},
		{
			desc: "labels are sorted by name, not left in emitted order",
			in:   `myapp_requests_total{method="post",code="200"} 1027`,
			name: "myapp_requests_total",
			key:  "code=200,method=post",
			val:  1027,
		},
		{
			desc: "a comma inside a label value is not split on",
			in:   `myapp_hits{file="a,b.prom"} 1`,
			name: "myapp_hits",
			key:  "file=a,b_prom",
			val:  1,
		},
		{
			desc: "a brace inside a label value is not split on",
			in:   `myapp_hits{expr="sum{job}"} 1`,
			name: "myapp_hits",
			key:  "expr=sum{job}",
			val:  1,
		},
		{
			desc: "an escaped quote round-trips, and the space is made safe",
			in:   `myapp_hits{note="say \"hi\""} 1`,
			name: "myapp_hits",
			key:  `note=say_"hi"`,
			val:  1,
		},
		{
			desc: "an escaped newline becomes whitespace, which is then made safe",
			in:   `myapp_hits{note="a\nb"} 1`,
			name: "myapp_hits",
			key:  "note=a_b",
			val:  1,
		},
		{
			desc: "a histogram bucket boundary is made subject safe",
			in:   `myapp_d_bucket{le="0.005"} 24054`,
			name: "myapp_d_bucket",
			key:  "le=0_005",
			val:  24054,
		},
		{
			desc: "a +Inf bucket keeps its label value",
			in:   `myapp_d_bucket{le="+Inf"} 144320`,
			name: "myapp_d_bucket",
			key:  "le=+Inf",
			val:  144320,
		},
		{
			desc: "a summary quantile is made subject safe",
			in:   `go_gc_duration_seconds{quantile="0.25"} 0.000103`,
			name: "go_gc_duration_seconds",
			key:  "quantile=0_25",
			val:  0.000103,
		},
		{
			desc: "periods are replaced but slashes are kept",
			in:   `myapp_build_info{version="1.4.2",branch="feat/scrape"} 1`,
			name: "myapp_build_info",
			key:  "branch=feat/scrape,version=1_4_2",
			val:  1,
		},
		{
			desc: "a space in a label value is replaced",
			in:   `myapp_hits{path="/api/v1",agent="curl 8.5"} 3`,
			name: "myapp_hits",
			key:  "agent=curl_8_5,path=/api/v1",
			val:  3,
		},
		{
			desc: "a colon in a metric name is left alone",
			in:   `myapp:hits:rate5m 0.5`,
			name: "myapp:hits:rate5m",
			key:  "",
			val:  0.5,
		},
		{
			desc: "an explicit timestamp is parsed and discarded",
			in:   `myapp_hits{code="200"} 1027 1395066363000`,
			name: "myapp_hits",
			key:  "code=200",
			val:  1027,
		},
		{
			desc: "an OpenMetrics exemplar is stripped",
			in:   `myapp_d_bucket{le="1"} 17 # {trace_id="abc"} 0.5 1395066363`,
			name: "myapp_d_bucket",
			key:  "le=1",
			val:  17,
		},
		{
			desc: "a trailing comma in the label set is allowed",
			in:   `myapp_hits{code="200",} 1`,
			name: "myapp_hits",
			key:  "code=200",
			val:  1,
		},
		{
			desc: "an exponent value parses",
			in:   `myapp_bytes 8.399016e+06`,
			name: "myapp_bytes",
			key:  "",
			val:  8.399016e+06,
		},
		{
			desc: "a negative value parses",
			in:   `myapp_offset_seconds -0.25`,
			name: "myapp_offset_seconds",
			key:  "",
			val:  -0.25,
		},
		{
			desc: "leading whitespace is tolerated",
			in:   "   myapp_hits 4",
			name: "myapp_hits",
			key:  "",
			val:  4,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			samples, bad, err := parseExposition(strings.NewReader(test.in))
			if err != nil {
				t.Fatal("Error parsing:", err)
			}

			if bad != 0 {
				t.Fatalf("Expected no bad lines, got %v", bad)
			}

			if len(samples) != 1 {
				t.Fatalf("Expected 1 sample, got %v: %+v", len(samples), samples)
			}

			s := samples[0]

			if s.name != test.name {
				t.Errorf("Expected name %q, got %q", test.name, s.name)
			}

			if s.key != test.key {
				t.Errorf("Expected key %q, got %q", test.key, s.key)
			}

			if s.val != test.val {
				t.Errorf("Expected value %v, got %v", test.val, s.val)
			}
		})
	}
}

// 0.005 and 0.05 both hold a period the store rejects, and they have to stay
// distinct once it is replaced, or two histogram buckets collapse into one.
func TestParseExpositionBucketsStayDistinct(t *testing.T) {
	in := `myapp_d_bucket{le="0.005"} 1
myapp_d_bucket{le="0.05"} 2
myapp_d_bucket{le="0.5"} 3
`

	samples, _, err := parseExposition(strings.NewReader(in))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	keys := map[string]bool{}

	for _, s := range samples {
		if keys[s.key] {
			t.Errorf("Duplicate key %q; buckets collapsed", s.key)
		}

		keys[s.key] = true
	}

	for _, exp := range []string{"le=0_005", "le=0_05", "le=0_5"} {
		if !keys[exp] {
			t.Errorf("Expected key %q, got %v", exp, keys)
		}
	}
}

// A non-finite sample value cannot be rendered by anything downstream, so it
// is dropped rather than published. This is the value, not a le="+Inf" label.
func TestParseExpositionNonFiniteValues(t *testing.T) {
	in := `myapp_a NaN
myapp_b +Inf
myapp_c -Inf
myapp_d 1
`

	samples, bad, err := parseExposition(strings.NewReader(in))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	if bad != 0 {
		t.Fatalf("Expected no bad lines, got %v", bad)
	}

	if len(samples) != 1 || samples[0].name != "myapp_d" {
		t.Fatalf("Expected only myapp_d, got %+v", samples)
	}
}

func TestParseExpositionCounterTypes(t *testing.T) {
	in := `# TYPE myapp_requests_total counter
myapp_requests_total 5
# TYPE myapp_queue_depth gauge
myapp_queue_depth 17
# TYPE myapp_d histogram
myapp_d_sum 1
myapp_d_count 2
myapp_d_bucket{le="1"} 3
`

	samples, _, err := parseExposition(strings.NewReader(in))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	exp := map[string]bool{
		"myapp_requests_total": true,
		"myapp_queue_depth":    false,
		// a histogram's components are monotonic, but only a name declared
		// counter is delta eligible in this pass -- see the plan
		"myapp_d_sum":    false,
		"myapp_d_count":  false,
		"myapp_d_bucket": false,
	}

	for _, s := range samples {
		want, ok := exp[s.name]
		if !ok {
			t.Errorf("Unexpected sample %q", s.name)
			continue
		}

		if s.counter != want {
			t.Errorf("Expected %q counter to be %v, got %v", s.name, want, s.counter)
		}
	}
}

// A TYPE line that follows its samples still marks them, since an exporter is
// not required to declare a metric before emitting it.
func TestParseExpositionLateTypeComment(t *testing.T) {
	in := "myapp_requests_total 5\n# TYPE myapp_requests_total counter\n"

	samples, _, err := parseExposition(strings.NewReader(in))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	if len(samples) != 1 || !samples[0].counter {
		t.Fatalf("Expected one counter sample, got %+v", samples)
	}
}

// One bad line should not cost us the good lines around it
func TestParseExpositionBadLines(t *testing.T) {
	in := `myapp_a 1
myapp_b{unterminated="x 2
myapp_c not_a_number
myapp_d
myapp_e{noquotes=x} 4
myapp_f 6
`

	samples, bad, err := parseExposition(strings.NewReader(in))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	if bad != 4 {
		t.Errorf("Expected 4 bad lines, got %v", bad)
	}

	if len(samples) != 2 {
		t.Fatalf("Expected 2 samples, got %+v", samples)
	}

	if samples[0].name != "myapp_a" || samples[1].name != "myapp_f" {
		t.Errorf("Expected myapp_a and myapp_f, got %+v", samples)
	}
}

func TestParseExpositionComments(t *testing.T) {
	in := `# HELP myapp_a A metric with \\ and \n in its help text.
# TYPE myapp_a counter
# a free-form comment
myapp_a 1
# EOF
`

	samples, bad, err := parseExposition(strings.NewReader(in))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	if bad != 0 {
		t.Errorf("Expected no bad lines, got %v", bad)
	}

	if len(samples) != 1 || !samples[0].counter {
		t.Fatalf("Expected one counter sample, got %+v", samples)
	}
}

// Every point a real exporter's output produces has to be publishable. This is
// the assertion that guards against the store rejection added in 0.23.2.
func TestParseExpositionFixtures(t *testing.T) {
	for _, name := range []string{
		"testdata/prom-client-golang.txt",
		"testdata/prom-node-exporter.txt",
	} {
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(name)
			if err != nil {
				t.Fatal("Error opening fixture:", err)
			}

			defer f.Close()

			samples, bad, err := parseExposition(f)
			if err != nil {
				t.Fatal("Error parsing fixture:", err)
			}

			if bad != 0 {
				t.Errorf("Expected no bad lines, got %v", bad)
			}

			if len(samples) < 5 {
				t.Fatalf("Expected the fixture to produce samples, got %v", len(samples))
			}

			for _, s := range samples {
				p := data.Point{Type: s.name, Key: s.key}
				if err := p.CheckSubjectTokens(); err != nil {
					t.Errorf("Sample is not publishable: %v", err)
				}
			}
		})
	}
}

func TestParseExpositionClientGolangFixture(t *testing.T) {
	f, err := os.Open("testdata/prom-client-golang.txt")
	if err != nil {
		t.Fatal("Error opening fixture:", err)
	}

	defer f.Close()

	samples, _, err := parseExposition(f)
	if err != nil {
		t.Fatal("Error parsing fixture:", err)
	}

	tests := []struct {
		name    string
		key     string
		val     float64
		counter bool
	}{
		// a summary is the one shape where the base metric name itself
		// carries a label
		{"go_gc_duration_seconds", "quantile=0_25", 0.000103, false},
		{"go_gc_duration_seconds_sum", "", 0.29, false},
		{"go_gc_duration_seconds_count", "", 3016, false},
		{"myapp_queue_depth", "", 17, false},
		{"myapp_requests_total", "code=200,method=post", 1027, true},
		{"myapp_requests_total", "code=200,method=get", 8621, true},
		{"myapp_request_duration_seconds_bucket", "le=0_005", 24054, false},
		{"myapp_request_duration_seconds_bucket", "le=+Inf", 144320, false},
		{"myapp_request_duration_seconds_sum", "", 53423.0, false},
		{"myapp_build_info", "branch=feat/scrape,version=1_4_2", 1, false},
		{"promhttp_metric_handler_requests_total", "code=500", 0, true},
	}

	for _, test := range tests {
		s, ok := findSample(samples, test.name, test.key)
		if !ok {
			t.Errorf("Expected a sample %q key %q", test.name, test.key)
			continue
		}

		if s.val != test.val {
			t.Errorf("Expected %q value %v, got %v", test.name, test.val, s.val)
		}

		if s.counter != test.counter {
			t.Errorf("Expected %q counter to be %v, got %v",
				test.name, test.counter, s.counter)
		}
	}

	// the NaN gauge in the fixture is dropped
	if _, ok := findSample(samples, "myapp_last_reload_success_timestamp_seconds", ""); ok {
		t.Error("Expected the NaN sample to be dropped")
	}
}

func TestParseExpositionNodeExporterFixture(t *testing.T) {
	f, err := os.Open("testdata/prom-node-exporter.txt")
	if err != nil {
		t.Fatal("Error opening fixture:", err)
	}

	defer f.Close()

	samples, _, err := parseExposition(f)
	if err != nil {
		t.Fatal("Error parsing fixture:", err)
	}

	tests := []struct {
		name string
		key  string
		val  float64
	}{
		// a mount point keeps its slashes, and a device name its periods
		// replaced
		{"node_filesystem_avail_bytes",
			"device=/dev/nvme0n1p2,fstype=ext4,mountpoint=/", 1.2412416e+11},
		{"node_network_receive_bytes_total", "device=eth0_100", 8.4172483e+08},
		{"node_hwmon_temp_celsius", "chip=platform_coretemp_0,sensor=temp1", 42.5},
		// a uname version string carries spaces, a '#', and periods, all of
		// which have to survive as one key
		{"node_uname_info",
			"release=6_1_0-18-amd64,sysname=Linux,version=#1_SMP_PREEMPT_DYNAMIC_Debian_6_1_76-1", 1},
		// a comma and an escaped quote inside a value
		{"node_textfile_mtime_seconds", "file=a,b_prom", 1.7e+09},
		{"node_textfile_mtime_seconds", `file=say_"hi"_prom`, 1.7e+09},
	}

	for _, test := range tests {
		s, ok := findSample(samples, test.name, test.key)
		if !ok {
			t.Errorf("Expected a sample %q key %q", test.name, test.key)
			continue
		}

		if s.val != test.val {
			t.Errorf("Expected %q value %v, got %v", test.name, test.val, s.val)
		}
	}
}
