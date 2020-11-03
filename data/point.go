package data

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// define common point types
const (
	// the following are point types that describe the
	// state of a device (vs sensor values).
	PointTypeDescription          string = "description"
	PointTypeCmdPending                  = "cmdPending"
	PointTypeSwUpdateState               = "swUpdateState"
	PointTypeStartApp                    = "startApp"
	PointTypeStartSystem                 = "startSystem"
	PointTypeUpdateOS                    = "updateOS"
	PointTypeUpdateApp                   = "updateApp"
	PointTypeSysState                    = "sysState"
	PointTypeSwUpdateRunning             = "swUpdateRunning"
	PointTypeSwUpdateError               = "swUpdateError"
	PointTypeSwUpdatePercComplete        = "swUpdatePercComplete"
	PointTypeOSVersion                   = "osVersion"
	PointTypeAppVersion                  = "appVersion"
	PointTypeHwVersion                   = "hwVersion"
	PointMsgAll                          = "msgAll"
	PointMsgUser                         = "msgUser"
)

// Point is a flexible data structure that can be used to represent
// a sensor value or a configuration parameter.
// ID, Type, and Index uniquely identify a point in a device
type Point struct {
	// ID of the sensor that provided the point
	ID string `json:"id,omitempty"`

	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty" boltholdIndex:"Type"`

	// Index is used to specify a position in an array such as
	// which pump, temp sensor, etc.
	Index int `json:"index,omitempty"`

	// Time the point was taken
	Time time.Time `json:"time,omitempty" boltholdKey:"Time" gob:"-"`

	// Duration over which the point was taken. This is useful
	// for averaged values to know what time period the value applies
	// to.
	Duration time.Duration `json:"duration,omitempty"`

	// Average OR
	// Instantaneous analog or digital value of the point.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// Optional text value of the point for data that is best represented
	// as a string rather than a number.
	Text string `json:"text,omitempty"`

	// statistical values that may be calculated over the duration of the point
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`
}

// ToPb encodes point in protobuf format
func (s Point) ToPb() (pb.Point, error) {
	ts, err := ptypes.TimestampProto(s.Time)
	if err != nil {
		return pb.Point{}, err
	}

	return pb.Point{
		Type:     s.Type,
		Id:       s.ID,
		Value:    float32(s.Value),
		Text:     s.Text,
		Time:     ts,
		Duration: ptypes.DurationProto(s.Duration),
	}, nil
}

// Bool returns a bool representation of value
func (s *Point) Bool() bool {
	if s.Value == 0 {
		return false
	}
	return true
}

// Points is an array of Point
type Points []Point

// Value fetches a value from an array of points given ID, Type, and Index.
// If ID or Type are set to "", they are ignored.
func (ps *Points) Value(id, typ string, index int) (float64, bool) {
	for _, p := range *ps {
		if id != "" && id != p.ID {
			continue
		}

		if typ != "" && typ != p.Type {
			continue
		}

		if index != p.Index {
			continue
		}

		return p.Value, true
	}

	return 0, false
}

// Text fetches a text value from an array of points given ID, Type, and Index.
// If ID or Type are set to "", they are ignored.
func (ps *Points) Text(id, typ string, index int) (string, bool) {
	for _, p := range *ps {
		if id != "" && id != p.ID {
			continue
		}

		if typ != "" && typ != p.Type {
			continue
		}

		if index != p.Index {
			continue
		}

		return p.Text, true
	}

	return "", false
}

// LatestTime returns the latest timestamp of a devices points
func (ps *Points) LatestTime() time.Time {
	ret := time.Time{}
	for _, p := range *ps {
		if p.Time.After(ret) {
			ret = p.Time
		}
	}

	return ret
}

// PbEncode encodes an array of points into protobuf
func (ps *Points) PbEncode() ([]byte, error) {
	pbPoints := make([]*pb.Point, len(*ps))
	for i, s := range *ps {
		sPb, err := s.ToPb()
		if err != nil {
			return []byte{}, err
		}

		pbPoints[i] = &sPb
	}

	return proto.Marshal(&pb.Points{Points: pbPoints})
}

// question -- should be using []*Point instead of []Point?

//PbToPoint converts pb point to point
func PbToPoint(sPb *pb.Point) (Point, error) {

	ts, err := ptypes.Timestamp(sPb.Time)
	if err != nil {
		return Point{}, err
	}

	dur, err := ptypes.Duration(sPb.Duration)
	if err != nil {
		return Point{}, err
	}

	ret := Point{
		ID:       sPb.Id,
		Type:     sPb.Type,
		Text:     sPb.Text,
		Value:    float64(sPb.Value),
		Time:     ts,
		Duration: dur,
	}

	return ret, nil
}

// PbDecodePoints decode protobuf encoded points
func PbDecodePoints(data []byte) ([]Point, error) {
	pbPoints := &pb.Points{}
	err := proto.Unmarshal(data, pbPoints)
	if err != nil {
		return []Point{}, err
	}

	ret := make([]Point, len(pbPoints.Points))

	for i, sPb := range pbPoints.Points {
		s, err := PbToPoint(sPb)
		if err != nil {
			return []Point{}, err
		}
		ret[i] = s
	}

	return ret, nil
}

// PointFilter is used to send points upstream. It only sends
// the data has changed, and at a max frequency
type PointFilter struct {
	minSend          time.Duration
	periodicSend     time.Duration
	points           []Point
	lastSent         time.Time
	lastPeriodicSend time.Time
}

// NewPointFilter is used to creat a new point filter
// If points have changed that get sent out at a minSend interval
// frequency of minSend.
// All points are periodically sent at lastPeriodicSend interval.
// Set minSend to 0 for things like config settings where you want them
// to be sent whenever anything changes.
func NewPointFilter(minSend, periodicSend time.Duration) *PointFilter {
	return &PointFilter{
		minSend:      minSend,
		periodicSend: periodicSend,
	}
}

// returns true if point has changed, and merges point with saved points
func (sf *PointFilter) add(point Point) bool {
	for i, s := range sf.points {
		if point.ID == s.ID && point.Type == s.Type {
			if point.Value == s.Value {
				return false
			}

			sf.points[i].Value = point.Value
			return true
		}
	}

	// point not found, add to array
	sf.points = append(sf.points, point)
	return true
}

// Add adds points and returns points that meet the filter criteria
func (sf *PointFilter) Add(points []Point) []Point {
	if time.Since(sf.lastPeriodicSend) > sf.periodicSend {
		// send all points
		for _, s := range points {
			sf.add(s)
		}

		sf.lastPeriodicSend = time.Now()
		sf.lastSent = sf.lastPeriodicSend
		return sf.points
	}

	if sf.minSend != 0 && time.Since(sf.lastSent) < sf.minSend {
		// don't return anything as
		return []Point{}
	}

	// now check if anything has changed and just send what has changed
	// only
	var ret []Point

	for _, s := range points {
		if sf.add(s) {
			ret = append(ret, s)
		}
	}

	if len(ret) > 0 {
		sf.lastSent = time.Now()
	}

	return ret
}
