package data

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Point is a flexible data structure that can be used to represent
// a sensor value or a configuration parameter.
// ID, Type, and Index uniquely identify a point in a device
type Point struct {
	//-------------------------------------------------------
	//1st three fields uniquely identify a point when receiving updates

	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty"`

	// Key is used to allow a group of points to represent a "map"
	Key string `json:"key,omitempty"`

	//-------------------------------------------------------
	// The following fields are the values for a point

	// Time the point was taken
	Time time.Time `json:"time,omitempty"`

	// Index is used to specify a position in an array such as
	// which pump, temp sensor, etc.
	Index float32 `json:"index,omitempty"`

	// Instantaneous analog or digital value of the point.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// Optional text value of the point for data that is best represented
	// as a string rather than a number.
	Text string `json:"text,omitempty"`

	// catchall field for data that does not fit into float or string --
	// should be used sparingly
	Data []byte `json:"data,omitempty"`

	//-------------------------------------------------------
	// Metadata

	// Used to indicate a point has been deleted. This value is only
	// ever incremented. Odd values mean point is deleted.
	Tombstone int `json:"tombstone,omitempty"`

	// Where did this point come from. If from the owning node, it may be blank.
	Origin string `json:"origin"`
}

// CRC returns a CRC for the point
func (p Point) CRC() uint32 {
	// Node type points are not returned so don't include that in hash
	if p.Type == PointTypeNodeType {
		return 0
	}
	// we are using this in a XOR checksum, so simply hashing time is probably
	// not good enough, because if we send a bunch of points with the same time,
	// they will have the CRC and simply cancel each other out.
	h := crc32.NewIEEE()
	d := make([]byte, 8)
	binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
	h.Write(d)
	h.Write([]byte(p.Type))
	h.Write([]byte(p.Key))
	h.Write([]byte(p.Text))
	binary.LittleEndian.PutUint64(d, math.Float64bits(p.Value))
	h.Write(d)

	return h.Sum32()
}

func (p Point) String() string {
	t := ""

	if p.Type != "" {
		t += "T:" + p.Type + " "
	}

	if p.Text != "" {
		t += fmt.Sprintf("V:%v ", p.Text)
	} else {
		t += fmt.Sprintf("V:%.3f ", p.Value)
	}

	if p.Index != 0 {
		t += fmt.Sprintf("I:%v ", p.Index)
	}

	if p.Key != "" {
		t += fmt.Sprintf("K:%v ", p.Key)
	}

	if p.Origin != "" {
		t += fmt.Sprintf("O:%v ", p.Origin)
	}

	t += p.Time.Format(time.RFC3339)

	return t
}

// IsMatch returns true if the point matches the params passed in
func (p Point) IsMatch(typ, key string) bool {
	if typ != "" && typ != p.Type {
		return false
	}

	if key != p.Key {
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
		Type:      p.Type,
		Index:     p.Index,
		Key:       p.Key,
		Value:     p.Value,
		Text:      p.Text,
		Time:      ts,
		Tombstone: int32(p.Tombstone),
		Origin:    p.Origin,
	}, nil
}

// ToSerial encodes point in serial protobuf format
func (p Point) ToSerial() (pb.SerialPoint, error) {
	return pb.SerialPoint{
		Type:      p.Type,
		Index:     p.Index,
		Key:       p.Key,
		Value:     float32(p.Value),
		Text:      p.Text,
		Time:      p.Time.UnixNano(),
		Tombstone: int32(p.Tombstone),
		Origin:    p.Origin,
	}, nil
}

// Bool returns a bool representation of value
func (p *Point) Bool() bool {
	return p.Value == 1
}

// Points is an array of Point
type Points []Point

func (ps Points) String() string {
	ret := ""
	for _, p := range ps {
		ret += p.String() + "\n"
	}

	return ret
}

// Desc returns a Description of a set of points
func (ps Points) Desc() string {
	firstName, _ := ps.Text(PointTypeFirstName, "")
	if firstName != "" {
		lastName, _ := ps.Text(PointTypeLastName, "")
		if lastName == "" {
			return firstName
		}

		return firstName + " " + lastName
	}

	desc, _ := ps.Text(PointTypeDescription, "")
	if desc != "" {
		return desc
	}

	return ""
}

// Find fetches a point given ID, Type, and Index
// and true of found, or false if not found
func (ps *Points) Find(typ, key string) (Point, bool) {
	for _, p := range *ps {
		if !p.IsMatch(typ, key) {
			continue
		}

		return p, true
	}

	return Point{}, false
}

// Value fetches a value from an array of points given ID, Type, and Index.
// If ID or Type are set to "", they are ignored.
func (ps *Points) Value(typ, key string) (float64, bool) {
	p, ok := ps.Find(typ, key)
	return p.Value, ok
}

// ValueInt returns value as integer
func (ps *Points) ValueInt(typ, key string) (int, bool) {
	f, ok := ps.Value(typ, key)
	return int(f), ok
}

// ValueBool returns value as bool
func (ps *Points) ValueBool(typ, key string) (bool, bool) {
	f, ok := ps.Value(typ, key)
	return FloatToBool(f), ok
}

// Text fetches a text value from an array of points given ID, Type, and Index.
// If ID or Type are set to "", they are ignored.
func (ps *Points) Text(typ, key string) (string, bool) {
	p, ok := ps.Find(typ, key)
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
func (ps *Points) Hash() uint32 {
	var ret uint32

	for _, p := range *ps {
		ret = ret ^ p.CRC()
	}

	return ret
}

// Add takes a point and updates an existing array of points. Existing points
// are replaced if the Timestamp in pIn is > than the existing timestamp. If
// the pIn timestamp is zero, the current time is used.
func (ps *Points) Add(pIn Point) {
	pFound := false

	if pIn.Time.IsZero() {
		pIn.Time = time.Now()
	}

	for i, p := range *ps {
		if p.Key == pIn.Key && p.Type == pIn.Type {
			pFound = true
			// largest tombstone value always wins
			tombstone := p.Tombstone
			if pIn.Tombstone > p.Tombstone {
				tombstone = pIn.Tombstone
			}

			if pIn.Time.After(p.Time) {
				(*ps)[i] = pIn
			}
			(*ps)[i].Tombstone = tombstone
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

// PbToPoint converts pb point to point
func PbToPoint(sPb *pb.Point) (Point, error) {

	ts, err := ptypes.Timestamp(sPb.Time)
	if err != nil {
		return Point{}, err
	}

	ret := Point{
		Type:      sPb.Type,
		Text:      sPb.Text,
		Key:       sPb.Key,
		Index:     sPb.Index,
		Value:     sPb.Value,
		Time:      ts,
		Tombstone: int(sPb.Tombstone),
		Origin:    sPb.Origin,
	}

	return ret, nil
}

// SerialToPoint converts serial pb point to point
func SerialToPoint(sPb *pb.SerialPoint) (Point, error) {
	ret := Point{
		Type:      sPb.Type,
		Text:      sPb.Text,
		Key:       sPb.Key,
		Index:     sPb.Index,
		Value:     float64(sPb.Value),
		Time:      time.Unix(0, sPb.Time),
		Tombstone: int(sPb.Tombstone),
		Origin:    sPb.Origin,
	}

	return ret, nil
}

// PbDecodePoints decode protobuf encoded points
func PbDecodePoints(data []byte) (Points, error) {
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

// DecodeSerialHrPayload decodes a serial high-rate payload. Payload format.
//   - type         (off:0, 16 bytes) point type
//   - key          (off:16, 16 bytes) point key
//   - starttime    (off:32, uint64) starting time of samples in ns since Unix Epoch
//   - sampleperiod (off:40, uint32) time between samples in ns
//   - data         (off:44) packed 32-bit floating point samples
func DecodeSerialHrPayload(payload []byte, callback func(Point)) error {
	if len(payload) < 16+16+8+4+4 {
		return fmt.Errorf("Payload is not long enough")
	}

	typ := string(bytes.Trim(payload[0:16], "\x00"))
	key := string(bytes.Trim(payload[16:32], "\x00"))
	startNs := int64(binary.LittleEndian.Uint64(payload[32:40]))
	if startNs == 0 {
		// if MCU does not send a time, fill in current time
		startNs = time.Now().UnixNano()
	}
	sampNs := int64(binary.LittleEndian.Uint32(payload[40:44]))

	sampCount := (len(payload) - (16 + 16 + 8 + 4)) / 4
	for i := 0; i < sampCount; i++ {
		callback(Point{
			Time: time.Unix(0, startNs+int64(i)*sampNs),
			Type: typ,
			Key:  key,
			Value: float64(math.Float32frombits(
				binary.LittleEndian.Uint32(payload[44+i*4 : 44+4+i*4]))),
		})
	}

	return nil
}

// PbDecodeSerialPoints can be used to decode serial points
func PbDecodeSerialPoints(d []byte) (Points, error) {
	pbSerial := &pb.SerialPoints{}

	err := proto.Unmarshal(d, pbSerial)
	if err != nil {
		return nil, fmt.Errorf("PB decode error: %v", err)
	}

	points := make([]Point, len(pbSerial.Points))

	for i, sPb := range pbSerial.Points {
		s, err := SerialToPoint(sPb)
		if err != nil {
			return nil, fmt.Errorf("Point decode error: %v", err)
		}
		points[i] = s
	}

	return points, nil
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
	for i, p := range sf.points {
		if point.Key == p.Key &&
			point.Type == p.Type &&
			point.Index == p.Index {
			if point.Value == p.Value {
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
	return v == 1
}

// BoolToFloat converts bool to float
func BoolToFloat(v bool) float64 {
	if !v {
		return 0
	}
	return 1
}
