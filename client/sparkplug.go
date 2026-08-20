package client

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb/sparkplug"
	"google.golang.org/protobuf/proto"
)

// sparkplugNamespace is the only topic namespace the specification defines.
const sparkplugNamespace = "spBv1.0"

// sparkplugFilter subscribes to everything in the namespace. Simple IoT does
// not know which groups exist until they publish.
const sparkplugFilter = sparkplugNamespace + "/#"

// sparkplugRebirthMetric is the metric an edge node exposes so a consumer can
// ask it to republish its birth certificates.
const sparkplugRebirthMetric = "Node Control/Rebirth"

// sparkplugRebirthInterval bounds how often a rebirth is requested from one
// edge node, so a gateway that cannot satisfy the request is asked politely
// rather than continuously.
const sparkplugRebirthInterval = time.Second * 30

// SparkplugGroup is created automatically from the group level of a Sparkplug
// topic. Its Sparkplug identity lives in the id point, which is what matches
// it on later messages, so renaming the description is safe.
type SparkplugGroup struct {
	ID          string            `node:"id"`
	Parent      string            `node:"parent"`
	Description string            `point:"description"`
	GroupID     string            `point:"id"`
	Tags        map[string]string `point:"tag"`
}

// SparkplugNode is an edge node, created automatically from a birth
// certificate.
type SparkplugNode struct {
	ID          string            `node:"id"`
	Parent      string            `node:"parent"`
	Description string            `point:"description"`
	EdgeNodeID  string            `point:"id"`
	SysState    string            `point:"sysState"`
	Tags        map[string]string `point:"tag"`
}

// SparkplugDevice is a device below an edge node, created automatically from a
// device birth certificate.
type SparkplugDevice struct {
	ID          string            `node:"id"`
	Parent      string            `node:"parent"`
	Description string            `point:"description"`
	DeviceID    string            `point:"id"`
	SysState    string            `point:"sysState"`
	Tags        map[string]string `point:"tag"`
}

// sparkplugTopic is a parsed Sparkplug B topic. Device is empty for edge node
// messages, and HostID is set only for STATE topics.
type sparkplugTopic struct {
	Group    string
	MsgType  string
	EdgeNode string
	Device   string
	HostID   string
}

// parseSparkplugTopic splits a topic in the namespace
// spBv1.0/{group}/{message type}/{edge node}/{device}, along with the
// spBv1.0/STATE/{host} form a primary host application publishes.
func parseSparkplugTopic(topic string) (sparkplugTopic, error) {
	parts := strings.Split(topic, "/")

	if len(parts) < 3 || parts[0] != sparkplugNamespace {
		return sparkplugTopic{}, fmt.Errorf("not a Sparkplug topic: %v", topic)
	}

	if parts[1] == "STATE" {
		if len(parts) != 3 {
			return sparkplugTopic{}, fmt.Errorf("malformed STATE topic: %v", topic)
		}

		return sparkplugTopic{MsgType: "STATE", HostID: parts[2]}, nil
	}

	t := sparkplugTopic{Group: parts[1], MsgType: parts[2]}

	switch t.MsgType {
	case "NBIRTH", "NDEATH", "NDATA", "NCMD":
		if len(parts) != 4 {
			return sparkplugTopic{}, fmt.Errorf("%v topic needs an edge node: %v", t.MsgType, topic)
		}

		t.EdgeNode = parts[3]

	case "DBIRTH", "DDEATH", "DDATA", "DCMD":
		if len(parts) != 5 {
			return sparkplugTopic{}, fmt.Errorf("%v topic needs a device: %v", t.MsgType, topic)
		}

		t.EdgeNode = parts[3]
		t.Device = parts[4]

	default:
		return sparkplugTopic{}, fmt.Errorf("unknown Sparkplug message type %q in %v", t.MsgType, topic)
	}

	if t.Group == "" || t.EdgeNode == "" {
		return sparkplugTopic{}, fmt.Errorf("empty level in the Sparkplug topic %v", topic)
	}

	return t, nil
}

// sparkplugState holds what the Sparkplug handler learns from the tree and
// from birth certificates. It belongs to one mqtt node and is rebuilt when the
// client restarts.
type sparkplugState struct {
	nc     *nats.Conn
	mqttID string
	desc   string
	debug  int

	// nodes indexes auto-created nodes by Sparkplug identity: "group",
	// "group/edge", and "group/edge/device"
	nodes map[string]string

	// aliases maps "group/edge" to the alias assignments from that edge
	// node's birth certificates. The specification makes an alias unique
	// across an edge node and all of its devices.
	aliases map[string]map[uint64]string

	// aliasSaved is the alias assignment last written to each edge node, so
	// an unchanged assignment is not rewritten on every birth
	aliasSaved map[string]string

	// rebirthAt is when a rebirth was last asked of each edge node
	rebirthAt map[string]time.Time
}

func newSparkplugState(nc *nats.Conn, mqttID, desc string, debug int) *sparkplugState {
	return &sparkplugState{
		nc:         nc,
		mqttID:     mqttID,
		desc:       desc,
		debug:      debug,
		nodes:      make(map[string]string),
		aliases:    make(map[string]map[uint64]string),
		aliasSaved: make(map[string]string),
		rebirthAt:  make(map[string]time.Time),
	}
}

// load indexes the nodes a previous run created, so a restart matches them by
// Sparkplug identity rather than creating a second set.
func (s *sparkplugState) load() error {
	groups, err := GetNodes(s.nc, s.mqttID, "all", data.NodeTypeSparkplugGroup, false)
	if err != nil {
		return err
	}

	for _, g := range groups {
		group := nodeIdentity(g)
		if group == "" {
			continue
		}

		s.nodes[group] = g.ID

		edges, err := GetNodes(s.nc, g.ID, "all", data.NodeTypeSparkplugNode, false)
		if err != nil {
			return err
		}

		for _, e := range edges {
			edge := nodeIdentity(e)
			if edge == "" {
				continue
			}

			edgeKey := group + "/" + edge
			s.nodes[edgeKey] = e.ID
			s.loadAliases(edgeKey, e)

			devices, err := GetNodes(s.nc, e.ID, "all", data.NodeTypeSparkplugDevice, false)
			if err != nil {
				return err
			}

			for _, d := range devices {
				device := nodeIdentity(d)
				if device == "" {
					continue
				}

				s.nodes[group+"/"+edge+"/"+device] = d.ID
			}
		}
	}

	return nil
}

// loadAliases restores the alias assignment an edge node carries, so data that
// arrives before the next birth certificate still resolves.
func (s *sparkplugState) loadAliases(edgeKey string, n data.NodeEdge) {
	p, ok := n.Points.Find(data.PointTypeSparkplugAlias, "")
	if !ok || len(p.Data) == 0 {
		return
	}

	aliases := make(map[uint64]string)

	if err := json.Unmarshal(p.Data, &aliases); err != nil {
		log.Printf("Sparkplug %v: error reading the alias cache for %v: %v\n",
			s.desc, edgeKey, err)
		return
	}

	s.aliases[edgeKey] = aliases
	s.aliasSaved[edgeKey] = string(p.Data)
}

// saveAliases writes the current alias assignment to the edge node. The whole
// assignment is written each time, so it never drifts from what the edge node
// last announced.
func (s *sparkplugState) saveAliases(edgeKey string) {
	edgeID, ok := s.nodes[edgeKey]
	if !ok {
		return
	}

	d, err := json.Marshal(s.aliases[edgeKey])
	if err != nil {
		log.Printf("Sparkplug %v: error encoding the alias cache: %v\n", s.desc, err)
		return
	}

	if s.aliasSaved[edgeKey] == string(d) {
		return
	}

	s.aliasSaved[edgeKey] = string(d)

	s.send(edgeID, data.Points{{
		Type:     data.PointTypeSparkplugAlias,
		Time:     time.Now(),
		DataType: data.PointDataTypeJSON,
		Data:     d,
	}})
}

// nodeIdentity reads the id point that holds a node's Sparkplug name.
func nodeIdentity(n data.NodeEdge) string {
	p, ok := n.Points.Find(data.PointTypeID, "")
	if !ok {
		return ""
	}

	return p.Txt()
}

// handle processes one message from the Sparkplug namespace.
func (s *sparkplugState) handle(topic string, payload []byte) {
	t, err := parseSparkplugTopic(topic)
	if err != nil {
		if s.debug > 0 {
			log.Printf("Sparkplug %v: %v\n", s.desc, err)
		}
		return
	}

	switch t.MsgType {
	case "STATE", "NCMD", "DCMD":
		// commands travel the other way, and acting as a primary host
		// application is not part of this support yet
		return
	}

	var p sparkplug.Payload

	if err := proto.Unmarshal(payload, &p); err != nil {
		log.Printf("Sparkplug %v: error decoding %v: %v\n", s.desc, topic, err)
		return
	}

	if s.debug > 0 {
		log.Printf("Sparkplug %v: %v, %v metrics\n", s.desc, topic, len(p.GetMetrics()))
	}

	switch t.MsgType {
	case "NBIRTH":
		s.birth(t, &p, false)
	case "DBIRTH":
		s.birth(t, &p, true)
	case "NDATA", "DDATA":
		s.dataMsg(t, &p)
	case "NDEATH":
		s.death(t, false)
	case "DDEATH":
		s.death(t, true)
	}
}

// birth creates or refreshes the nodes named by the topic, records the alias
// assignments, and writes the initial value of every metric.
func (s *sparkplugState) birth(t sparkplugTopic, p *sparkplug.Payload, device bool) {
	nodeID, err := s.ensureNodes(t, device)
	if err != nil {
		log.Printf("Sparkplug %v: %v\n", s.desc, err)
		return
	}

	edgeKey := t.Group + "/" + t.EdgeNode

	if !device {
		// an edge node birth restarts the whole alias assignment for that
		// edge node, devices included
		s.aliases[edgeKey] = make(map[uint64]string)
	} else if s.aliases[edgeKey] == nil {
		s.aliases[edgeKey] = make(map[uint64]string)
	}

	aliases := s.aliases[edgeKey]

	pts := make(data.Points, 0, len(p.GetMetrics()))

	for _, m := range p.GetMetrics() {
		name := m.GetName()

		if name != "" && m.GetAlias() != 0 {
			aliases[m.GetAlias()] = name
		}

		if pt, ok := s.metricPoint(m, name, p.GetTimestamp()); ok {
			pts = append(pts, pt)
		}
	}

	pts = append(pts, s.statePoint(data.PointValueSysStateOnline))

	s.send(nodeID, pts)
	s.saveAliases(edgeKey)
}

// dataMsg publishes the metrics of an NDATA or DDATA message, resolving the
// numeric aliases the birth certificate assigned.
func (s *sparkplugState) dataMsg(t sparkplugTopic, p *sparkplug.Payload) {
	key := t.Group + "/" + t.EdgeNode
	if t.Device != "" {
		key += "/" + t.Device
	}

	nodeID, known := s.nodes[key]
	if !known {
		// data before a birth: ask for the birth certificates that describe it
		s.requestRebirth(t)
		return
	}

	aliases := s.aliases[t.Group+"/"+t.EdgeNode]

	pts := make(data.Points, 0, len(p.GetMetrics()))

	for _, m := range p.GetMetrics() {
		name := m.GetName()

		if name == "" {
			name = aliases[m.GetAlias()]
		}

		if name == "" {
			s.requestRebirth(t)
			return
		}

		if pt, ok := s.metricPoint(m, name, p.GetTimestamp()); ok {
			pts = append(pts, pt)
		}
	}

	s.send(nodeID, pts)
}

// death marks a node offline. The node is kept, because a device that has gone
// quiet is still part of the plant.
func (s *sparkplugState) death(t sparkplugTopic, device bool) {
	key := t.Group + "/" + t.EdgeNode
	if device {
		key += "/" + t.Device
	}

	nodeID, known := s.nodes[key]
	if !known {
		return
	}

	s.send(nodeID, data.Points{s.statePoint(data.PointValueSysStateOffline)})

	if device {
		return
	}

	// an edge node death takes its devices with it
	prefix := key + "/"

	for k, id := range s.nodes {
		if strings.HasPrefix(k, prefix) {
			s.send(id, data.Points{s.statePoint(data.PointValueSysStateOffline)})
		}
	}
}

// ensureNodes creates the group, edge node, and device nodes named by a topic
// if they are not there yet, and returns the node the metrics belong on.
func (s *sparkplugState) ensureNodes(t sparkplugTopic, device bool) (string, error) {
	groupID, ok := s.nodes[t.Group]

	if !ok {
		groupID = uuid.New().String()

		g := SparkplugGroup{
			ID:          groupID,
			Parent:      s.mqttID,
			Description: t.Group,
			GroupID:     t.Group,
			Tags:        map[string]string{data.NodeTypeSparkplugGroup: t.Group},
		}

		if err := SendNodeType(s.nc, g, s.mqttID); err != nil {
			return "", fmt.Errorf("error creating Sparkplug group %v: %w", t.Group, err)
		}

		log.Printf("Sparkplug %v: added group %v\n", s.desc, t.Group)
		s.nodes[t.Group] = groupID
	}

	edgeKey := t.Group + "/" + t.EdgeNode

	edgeID, ok := s.nodes[edgeKey]

	if !ok {
		edgeID = uuid.New().String()

		e := SparkplugNode{
			ID:          edgeID,
			Parent:      groupID,
			Description: t.EdgeNode,
			EdgeNodeID:  t.EdgeNode,
			Tags:        map[string]string{data.NodeTypeSparkplugNode: t.EdgeNode},
		}

		if err := SendNodeType(s.nc, e, s.mqttID); err != nil {
			return "", fmt.Errorf("error creating Sparkplug edge node %v: %w", edgeKey, err)
		}

		log.Printf("Sparkplug %v: added edge node %v\n", s.desc, edgeKey)
		s.nodes[edgeKey] = edgeID
	}

	if !device {
		return edgeID, nil
	}

	deviceKey := edgeKey + "/" + t.Device

	deviceID, ok := s.nodes[deviceKey]

	if !ok {
		deviceID = uuid.New().String()

		d := SparkplugDevice{
			ID:          deviceID,
			Parent:      edgeID,
			Description: t.Device,
			DeviceID:    t.Device,
			Tags:        map[string]string{data.NodeTypeSparkplugDevice: t.Device},
		}

		if err := SendNodeType(s.nc, d, s.mqttID); err != nil {
			return "", fmt.Errorf("error creating Sparkplug device %v: %w", deviceKey, err)
		}

		log.Printf("Sparkplug %v: added device %v\n", s.desc, deviceKey)
		s.nodes[deviceKey] = deviceID
	}

	return deviceID, nil
}

// metricPoint converts one Sparkplug metric into a point. The metric name
// becomes the point key, so a device node carries one point per metric.
func (s *sparkplugState) metricPoint(m *sparkplug.Payload_Metric, name string,
	payloadTime uint64) (data.Point, bool) {

	if name == "" || m.GetIsNull() {
		return data.Point{}, false
	}

	key := data.SubjectSafeToken(name)

	var p data.Point

	switch v := m.GetValue().(type) {
	case *sparkplug.Payload_Metric_IntValue:
		p = data.NewPointInt(data.PointTypeValue, key, int64(v.IntValue))
	case *sparkplug.Payload_Metric_LongValue:
		p = data.NewPointInt(data.PointTypeValue, key, int64(v.LongValue))
	case *sparkplug.Payload_Metric_FloatValue:
		p = data.NewPointFloat(data.PointTypeValue, key, float64(v.FloatValue))
	case *sparkplug.Payload_Metric_DoubleValue:
		p = data.NewPointFloat(data.PointTypeValue, key, v.DoubleValue)
	case *sparkplug.Payload_Metric_BooleanValue:
		b := int64(0)
		if v.BooleanValue {
			b = 1
		}
		p = data.NewPointInt(data.PointTypeValue, key, b)
	case *sparkplug.Payload_Metric_StringValue:
		p = data.NewPointString(data.PointTypeValue, key, v.StringValue)
	default:
		// datasets, templates, and file payloads have no point representation
		// yet; skipping them leaves the rest of the message usable
		if s.debug > 0 {
			log.Printf("Sparkplug %v: skipping metric %v of type %v\n",
				s.desc, name, m.GetDatatype())
		}
		return data.Point{}, false
	}

	ts := m.GetTimestamp()
	if ts == 0 {
		ts = payloadTime
	}

	if ts != 0 {
		p.Time = time.UnixMilli(int64(ts))
	}

	return p, true
}

func (s *sparkplugState) statePoint(state string) data.Point {
	return data.NewPointString(data.PointTypeSysState, "", state)
}

// send publishes points to an auto-created node, marking them with the mqtt
// node as their origin since they belong to a node this client does not own.
func (s *sparkplugState) send(nodeID string, pts data.Points) {
	if len(pts) == 0 {
		return
	}

	for i := range pts {
		pts[i].Origin = s.mqttID
	}

	if err := SendNodePoints(s.nc, nodeID, pts, false); err != nil {
		log.Printf("Sparkplug %v: error sending points: %v\n", s.desc, err)
	}
}

// requestRebirth asks an edge node to republish its birth certificates, which
// is how Simple IoT recovers the alias assignments after its own restart. The
// NCMD goes out on the NATS subject the broker maps the topic to, so it
// reaches MQTT subscribers without an MQTT client.
func (s *sparkplugState) requestRebirth(t sparkplugTopic) {
	key := t.Group + "/" + t.EdgeNode

	if time.Since(s.rebirthAt[key]) < sparkplugRebirthInterval {
		return
	}

	s.rebirthAt[key] = time.Now()

	name := sparkplugRebirthMetric
	datatype := uint32(11) // Boolean
	now := uint64(time.Now().UnixMilli())

	payload := &sparkplug.Payload{
		Timestamp: &now,
		Metrics: []*sparkplug.Payload_Metric{{
			Name:      &name,
			Timestamp: &now,
			Datatype:  &datatype,
			Value:     &sparkplug.Payload_Metric_BooleanValue{BooleanValue: true},
		}},
	}

	body, err := proto.Marshal(payload)
	if err != nil {
		log.Printf("Sparkplug %v: error encoding rebirth request: %v\n", s.desc, err)
		return
	}

	topic := fmt.Sprintf("%v/%v/NCMD/%v", sparkplugNamespace, t.Group, t.EdgeNode)

	subject, err := data.MQTTTopicToSubject(topic)
	if err != nil {
		log.Printf("Sparkplug %v: %v\n", s.desc, err)
		return
	}

	log.Printf("Sparkplug %v: requesting a rebirth from %v\n", s.desc, key)

	if err := s.nc.Publish(subject, body); err != nil {
		log.Printf("Sparkplug %v: error requesting a rebirth: %v\n", s.desc, err)
	}
}
