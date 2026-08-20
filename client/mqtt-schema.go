package client

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// mqttScalarField is the field name a gateway publishing one measurement per
// topic wraps its value in. A payload carrying only this field is treated as a
// scalar, so the key comes from the topic rather than gaining a "value" level.
const mqttScalarField = "value"

// mqttMaxNodesDefault bounds how many nodes a topic schema creates when the
// mqtt node does not set a limit of its own.
const mqttMaxNodesDefault = 1000

// mqttSchemaLevel is one level of a topic schema: either a literal that a
// topic has to match, or a named level whose value becomes a node.
type mqttSchemaLevel struct {
	literal string
	label   string
}

// mqttSchema is a parsed topic schema such as "{site}/{gateway}/{device}".
type mqttSchema struct {
	levels []mqttSchemaLevel
	// filter is the MQTT topic filter that covers every topic the schema
	// describes: a single-level wildcard for each named level, and the
	// remainder wildcard for whatever a topic carries beyond them
	filter string
}

// parseMqttSchema reads a topic schema. Named levels are written in braces,
// and anything else is a literal that a topic has to match exactly. A trailing
// "#" may be written to make the remainder explicit; it is what the schema
// covers in any case.
func parseMqttSchema(schema string) (mqttSchema, error) {
	if strings.TrimSpace(schema) == "" {
		return mqttSchema{}, fmt.Errorf("the topic schema is empty")
	}

	parts := strings.Split(schema, "/")

	var (
		s      mqttSchema
		filter []string
		labels = make(map[string]bool)
	)

	for i, p := range parts {
		switch {
		case p == "":
			return mqttSchema{}, fmt.Errorf("topic schema %q has an empty level", schema)

		case p == "#":
			if i != len(parts)-1 {
				return mqttSchema{}, fmt.Errorf(
					"topic schema %q names a level after the remainder", schema)
			}

		case strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}"):
			label := p[1 : len(p)-1]

			if label == "" {
				return mqttSchema{}, fmt.Errorf("topic schema %q has an unnamed level", schema)
			}

			if labels[label] {
				return mqttSchema{}, fmt.Errorf(
					"topic schema %q names %q more than once", schema, label)
			}

			labels[label] = true
			s.levels = append(s.levels, mqttSchemaLevel{label: label})
			filter = append(filter, "+")

		case strings.ContainsAny(p, "{}+#"):
			return mqttSchema{}, fmt.Errorf(
				"topic schema %q has a level %q that is neither a name nor a literal", schema, p)

		default:
			s.levels = append(s.levels, mqttSchemaLevel{literal: p})
			filter = append(filter, p)
		}
	}

	if len(labels) == 0 {
		return mqttSchema{}, fmt.Errorf(
			"topic schema %q names no levels, so it creates nothing", schema)
	}

	s.filter = strings.Join(filter, "/") + "/#"

	return s, nil
}

// match checks a topic against the schema, returning the value of each named
// level and the topic levels left over, which become part of the point key.
func (s mqttSchema) match(topic string) (values []string, rest []string, ok bool) {
	parts := strings.Split(topic, "/")

	if len(parts) < len(s.levels) {
		return nil, nil, false
	}

	for i, l := range s.levels {
		if l.label == "" {
			if parts[i] != l.literal {
				return nil, nil, false
			}
			continue
		}

		if parts[i] == "" {
			return nil, nil, false
		}

		values = append(values, parts[i])
	}

	return values, parts[len(s.levels):], true
}

// labels returns the schema label of each named level, in order.
func (s mqttSchema) labels() []string {
	var out []string

	for _, l := range s.levels {
		if l.label != "" {
			out = append(out, l.label)
		}
	}

	return out
}

// mqttFilterMatch reports whether an MQTT topic filter matches a topic,
// following the wildcard rules of the specification. It is what gives an
// explicit subscription precedence over the topic schema.
func mqttFilterMatch(filter, topic string) bool {
	f := strings.Split(filter, "/")
	t := strings.Split(topic, "/")

	for i, level := range f {
		if level == "#" {
			// # matches the remainder, including no levels at all
			return i <= len(t)
		}

		if i >= len(t) {
			return false
		}

		if level == "+" {
			continue
		}

		if level != t[i] {
			return false
		}
	}

	return len(f) == len(t)
}

// mqttSchemaState holds the nodes a topic schema has created for one mqtt
// node. It is rebuilt from the tree when the client restarts, so nodes are
// matched by the identity point rather than created a second time.
type mqttSchemaState struct {
	nc       *nats.Conn
	mqttID   string
	desc     string
	debug    int
	schema   mqttSchema
	maxNodes int

	// nodes indexes auto-created nodes by the topic levels they came from,
	// joined with "/"
	nodes map[string]string

	// full records whether the node limit has been reported, so it is
	// reported once rather than on every message
	full bool
}

func newMqttSchemaState(nc *nats.Conn, mqttID, desc string, debug, maxNodes int,
	schema mqttSchema) *mqttSchemaState {

	if maxNodes <= 0 {
		maxNodes = mqttMaxNodesDefault
	}

	return &mqttSchemaState{
		nc:       nc,
		mqttID:   mqttID,
		desc:     desc,
		debug:    debug,
		schema:   schema,
		maxNodes: maxNodes,
		nodes:    make(map[string]string),
	}
}

// load indexes the nodes a previous run created. Intermediate levels are group
// nodes and the last named level is an mqttDevice, and each carries an id
// point holding the raw topic level it came from.
func (s *mqttSchemaState) load() error {
	depth := len(s.schema.labels())

	var walk func(parentID string, path []string, level int) error

	walk = func(parentID string, path []string, level int) error {
		nodeType := data.NodeTypeGroup
		if level == depth-1 {
			nodeType = data.NodeTypeMqttDevice
		}

		nodes, err := GetNodes(s.nc, parentID, "all", nodeType, false)
		if err != nil {
			return err
		}

		for _, n := range nodes {
			identity := nodeIdentity(n)
			if identity == "" {
				continue
			}

			p := append(append([]string{}, path...), identity)
			s.nodes[strings.Join(p, "/")] = n.ID

			if level < depth-1 {
				if err := walk(n.ID, p, level+1); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return walk(s.mqttID, nil, 0)
}

// handle turns one message into points on the node its topic names, creating
// the nodes along the way the first time a topic is seen.
func (s *mqttSchemaState) handle(topic string, payload []byte) {
	values, rest, ok := s.schema.match(topic)
	if !ok {
		return
	}

	nodeID, err := s.ensureNodes(values)
	if err != nil {
		s.reportFull(err)
		return
	}

	if nodeID == "" {
		return
	}

	pts, err := mqttSchemaPoints(payload, rest)
	if err != nil {
		if s.debug > 0 {
			log.Printf("MQTT %v: %v: %v\n", s.desc, topic, err)
		}
		return
	}

	for i := range pts {
		pts[i].Origin = s.mqttID
	}

	if err := SendNodePoints(s.nc, nodeID, pts, false); err != nil {
		log.Println("MQTT: error sending schema points:", err)
	}
}

// ensureNodes walks the named levels of a topic, creating a node for each one
// that does not have one yet, and returns the device node the points belong
// on. An empty ID with no error means the node limit has been reached.
func (s *mqttSchemaState) ensureNodes(values []string) (string, error) {
	labels := s.schema.labels()
	parentID := s.mqttID

	for i, v := range values {
		key := strings.Join(values[:i+1], "/")

		id, ok := s.nodes[key]

		if !ok {
			if len(s.nodes) >= s.maxNodes {
				return "", fmt.Errorf(
					"the topic schema has created its limit of %v nodes; new topics are being dropped",
					s.maxNodes)
			}

			id = uuid.New().String()

			nodeType := data.NodeTypeGroup
			if i == len(values)-1 {
				nodeType = data.NodeTypeMqttDevice
			}

			node := data.NodeEdge{
				ID:     id,
				Parent: parentID,
				Type:   nodeType,
				Points: data.Points{
					data.NewPointString(data.PointTypeDescription, "", v),
					data.NewPointString(data.PointTypeID, "", v),
					data.NewPointString(data.PointTypeTag, labels[i], v),
				},
			}

			if err := SendNode(s.nc, node, s.mqttID); err != nil {
				return "", fmt.Errorf("error creating %v node %v: %w", nodeType, key, err)
			}

			log.Printf("MQTT %v: added %v %v\n", s.desc, nodeType, key)

			s.nodes[key] = id
			s.clearFull()
		}

		parentID = id
	}

	return parentID, nil
}

// reportFull writes the node limit error on the mqtt node once, so a busy
// broker does not fill the log or the store with the same message.
func (s *mqttSchemaState) reportFull(err error) {
	if s.full {
		return
	}

	s.full = true

	log.Printf("MQTT %v: %v\n", s.desc, err)

	p := data.NewPointString(data.PointTypeError, "", err.Error())

	if e := SendNodePoint(s.nc, s.mqttID, p, false); e != nil {
		log.Println("MQTT: error sending error point:", e)
	}
}

func (s *mqttSchemaState) clearFull() {
	if !s.full {
		return
	}

	s.full = false

	if err := SendNodePoint(s.nc, s.mqttID,
		data.NewPointString(data.PointTypeError, "", ""), false); err != nil {
		log.Println("MQTT: error clearing error point:", err)
	}
}

// mqttSchemaPoints maps a payload into points, with the topic levels left over
// after the named ones joined into the point key. A payload carrying a single
// field named "value" is treated as a scalar, since that is the shape a
// gateway publishing one measurement per topic uses.
func mqttSchemaPoints(payload []byte, rest []string) (data.Points, error) {
	pts, err := MqttSub{}.points(payload)
	if err != nil {
		return nil, err
	}

	prefix := make([]string, 0, len(rest))

	for _, r := range rest {
		if r == "" {
			continue
		}
		prefix = append(prefix, data.SubjectSafeToken(r))
	}

	if len(pts) == 1 && pts[0].Key == mqttScalarField {
		pts[0].Key = ""
	}

	for i := range pts {
		parts := prefix

		if pts[i].Key != "" {
			parts = append(append([]string{}, prefix...), pts[i].Key)
		}

		pts[i].Key = strings.Join(parts, "/")
	}

	return pts, nil
}
