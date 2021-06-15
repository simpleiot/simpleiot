package data

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Point is a flexible data structure that can be used to represent
// a sensor value or a configuration parameter.
// ID, Type, and Index uniquely identify a point in a device
type Point struct {
	// ID of the sensor that provided the point
	ID string `json:"id,omitempty"`

	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty"`

	// Index is used to specify a position in an array such as
	// which pump, temp sensor, etc.
	Index int `json:"index,omitempty"`

	// Time the point was taken
	Time time.Time `json:"time,omitempty"`

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

func (p Point) String() string {
	t := ""

	if p.Type != "" {
		t += "T:" + p.Type + " "
	}

	if p.Text != "" {
		t += fmt.Sprintf("V:%v ", p.Text)
	} else {
		t += fmt.Sprintf("V:%v ", p.Value)
	}

	if p.Index != 0 {
		t += fmt.Sprintf("I:%v ", p.Index)
	}

	if p.ID != "" {
		t += fmt.Sprintf("ID:%v ", p.ID)
	}

	t += p.Time.Format(time.RFC3339)

	return t
}

// IsMatch returns true if the point matches the params passed in
func (p Point) IsMatch(id, typ string, index int) bool {
	if id != "" && id != p.ID {
		return false
	}

	if typ != "" && typ != p.Type {
		return false
	}

	if index != p.Index {
		return false
	}

	return true
}

// ToPb encodes point in protobuf format
func (p Point) ToPb() (pb.Point, error) {
	ts, err := ptypes.TimestampProto(p.Time)
	if err != nil {
		return pb.Point{}, err
	}

	return pb.Point{
		Type:     p.Type,
		Id:       p.ID,
		Index:    int32(p.Index),
		Value:    float32(p.Value),
		Text:     p.Text,
		Time:     ts,
		Duration: ptypes.DurationProto(p.Duration),
	}, nil
}

// Bool returns a bool representation of value
func (p *Point) Bool() bool {
	if p.Value == 0 {
		return false
	}
	return true
}

// Points is an array of Point
type Points []Point

// Desc returns a Description of a set of points
func (ps Points) Desc() string {
	firstName, _ := ps.Text("", PointTypeFirstName, 0)
	if firstName != "" {
		lastName, _ := ps.Text("", PointTypeLastName, 0)
		if lastName == "" {
			return firstName
		}

		return firstName + " " + lastName
	}

	desc, _ := ps.Text("", PointTypeDescription, 0)
	if desc != "" {
		return desc
	}

	return ""
}

// Find fetches a point given ID, Type, and Index
// and true of found, or false if not found
func (ps *Points) Find(id, typ string, index int) (Point, bool) {
	for _, p := range *ps {
		if !p.IsMatch(id, typ, index) {
			continue
		}

		return p, true
	}

	return Point{}, false
}

// Value fetches a value from an array of points given ID, Type, and Index.
// If ID or Type are set to "", they are ignored.
func (ps *Points) Value(id, typ string, index int) (float64, bool) {
	p, ok := ps.Find(id, typ, index)
	return p.Value, ok
}

// ValueInt returns value as integer
func (ps *Points) ValueInt(id, typ string, index int) (int, bool) {
	f, ok := ps.Value(id, typ, index)
	return int(f), ok
}

// ValueBool returns value as bool
func (ps *Points) ValueBool(id, typ string, index int) (bool, bool) {
	f, ok := ps.Value(id, typ, index)
	return FloatToBool(f), ok
}

// Text fetches a text value from an array of points given ID, Type, and Index.
// If ID or Type are set to "", they are ignored.
func (ps *Points) Text(id, typ string, index int) (string, bool) {
	p, ok := ps.Find(id, typ, index)
	return p.Text, ok
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

// ToPb encodes an array of points into protobuf
func (ps *Points) ToPb() ([]byte, error) {
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

// Hash returns the hash of points
func (ps *Points) Hash() []byte {
	h := md5.New()

	for _, p := range *ps {
		d := make([]byte, 8)
		binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
		h.Write(d)
	}

	return h.Sum(nil)
}

// ProcessPoint takes a point and updates an existing array of points
func (ps *Points) ProcessPoint(pIn Point) {
	pFound := false
	for i, p := range *ps {
		if p.ID == pIn.ID && p.Type == pIn.Type && p.Index == pIn.Index {
			pFound = true
			if pIn.Time.After(p.Time) {
				(*ps)[i] = pIn
			}
		}
	}

	if !pFound {
		*ps = append(*ps, pIn)
	}
}

// Implement methods needed by sort.Interface

// Len returns the number of points
func (ps Points) Len() int {
	return len([]Point(ps))
}

// Less is required by sort.Interface
func (ps Points) Less(i, j int) bool {
	return ps[i].Time.Before(ps[j].Time)
}

// Swap is required by sort.Interface
func (ps Points) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

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
		Index:    int(sPb.Index),
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

// FloatToBool converts a float to bool
func FloatToBool(v float64) bool {
	if v == 0 {
		return false
	}

	return true
}

// BoolToFloat converts bool to float
func BoolToFloat(v bool) float64 {
	if !v {
		return 0
	}
	return 1
}
