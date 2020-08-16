package data

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// define common sample types
const (
	SampleTypeStartApp    string = "startApp"
	SampleTypeStartSystem        = "startSystem"
	SampleTypeUpdateOS           = "updateOS"
	SampleTypeUpdateApp          = "updateApp"
	SampleTypeSysState           = "sysState"
)

// Sample represents a value in time and should include data that may be
// graphed.
type Sample struct {
	// Type of sample (voltage, current, key, etc)
	Type string `json:"type,omitempty" boltholdIndex:"Type"`

	// ID of the sensor that provided the sample
	ID string `json:"id,omitempty"`

	// Average OR
	// Instantaneous analog or digital value of the sample.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// statistical values that may be calculated
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`

	// Time the sample was taken
	Time time.Time `json:"time,omitempty" boltholdKey:"Time" gob:"-"`

	// Duration over which the sample was taken
	Duration time.Duration `json:"duration,omitempty"`

	// Tags are additional attributes used to describe the sample
	// You might add things like friendly name, etc.
	Tags map[string]string `json:"tags,omitempty"`

	// Attributes are additional numerical values
	Attributes map[string]float64 `json:"attributes,omitempty"`
}

// ToPb encodes sample in protobuf format
func (s Sample) ToPb() (pb.Sample, error) {
	ts, err := ptypes.TimestampProto(s.Time)
	if err != nil {
		return pb.Sample{}, err
	}

	return pb.Sample{
		Type:     s.Type,
		Id:       s.ID,
		Value:    float32(s.Value),
		Time:     ts,
		Duration: ptypes.DurationProto(s.Duration),
	}, nil
}

// ForDevice tells us if a sample is for device (vs IO)
func (s Sample) ForDevice() bool {
	if s.Type == SampleTypeSysState {
		return true
	}

	return false
}

// Bool returns a bool representation of value
func (s *Sample) Bool() bool {
	if s.Value == 0 {
		return false
	}
	return true
}

// Samples is an array of Sample
type Samples []Sample

// PbEncode encodes an array of samples into protobuf
func (s *Samples) PbEncode() ([]byte, error) {
	pbSamples := make([]*pb.Sample, len(*s))
	for i, s := range *s {
		sPb, err := s.ToPb()
		if err != nil {
			return []byte{}, err
		}

		pbSamples[i] = &sPb
	}

	return proto.Marshal(&pb.Samples{Samples: pbSamples})
}

// question -- should be using []*Sample instead of []Sample?

//PbToSample converts pb sample to sample
func PbToSample(sPb *pb.Sample) (Sample, error) {

	ts, err := ptypes.Timestamp(sPb.Time)
	if err != nil {
		return Sample{}, err
	}

	dur, err := ptypes.Duration(sPb.Duration)
	if err != nil {
		return Sample{}, err
	}

	ret := Sample{
		ID:       sPb.Id,
		Type:     sPb.Type,
		Value:    float64(sPb.Value),
		Time:     ts,
		Duration: dur,
	}

	return ret, nil
}

// PbDecodeSamples decode protobuf encoded samples
func PbDecodeSamples(data []byte) ([]Sample, error) {
	pbSamples := &pb.Samples{}
	err := proto.Unmarshal(data, pbSamples)
	if err != nil {
		return []Sample{}, err
	}

	ret := make([]Sample, len(pbSamples.Samples))

	for i, sPb := range pbSamples.Samples {
		s, err := PbToSample(sPb)
		if err != nil {
			return []Sample{}, err
		}
		ret[i] = s
	}

	return ret, nil
}

// SampleFilter is used to send samples upstream. It only sends
// the data has changed, and at a max frequency
type SampleFilter struct {
	minSend          time.Duration
	periodicSend     time.Duration
	samples          []Sample
	lastSent         time.Time
	lastPeriodicSend time.Time
}

// NewSampleFilter is used to creat a new sample filter
// If samples have changed that get sent out at a minSend interval
// frequency of minSend.
// All samples are periodically sent at lastPeriodicSend interval.
// Set minSend to 0 for things like config settings where you want them
// to be sent whenever anything changes.
func NewSampleFilter(minSend, periodicSend time.Duration) *SampleFilter {
	return &SampleFilter{
		minSend:      minSend,
		periodicSend: periodicSend,
	}
}

// returns true if sample has changed, and merges sample with saved samples
func (sf *SampleFilter) add(sample Sample) bool {
	for i, s := range sf.samples {
		if sample.ID == s.ID && sample.Type == s.Type {
			if sample.Value == s.Value {
				return false
			}

			sf.samples[i].Value = sample.Value
			return true
		}
	}

	// sample not found, add to array
	sf.samples = append(sf.samples, sample)
	return true
}

// Add adds samples and returns samples that meet the filter criteria
func (sf *SampleFilter) Add(samples []Sample) []Sample {
	if time.Since(sf.lastPeriodicSend) > sf.periodicSend {
		// send all samples
		for _, s := range samples {
			sf.add(s)
		}

		sf.lastPeriodicSend = time.Now()
		sf.lastSent = sf.lastPeriodicSend
		return sf.samples
	}

	if sf.minSend != 0 && time.Since(sf.lastSent) < sf.minSend {
		// don't return anything as
		return []Sample{}
	}

	// now check if anything has changed and just send what has changed
	// only
	var ret []Sample

	for _, s := range samples {
		if sf.add(s) {
			ret = append(ret, s)
		}
	}

	if len(ret) > 0 {
		sf.lastSent = time.Now()
	}

	return ret
}
