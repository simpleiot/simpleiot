package data

// A node can live in more than one place in the tree, and for most node types
// that is the whole point: a user belongs to two groups, a rule is visible from
// two places. For a node that owns something outside the tree -- a Modbus bus,
// a GPIO line, an MQTT broker connection -- it is not. Two clients acting on
// one piece of hardware from two instances have no way to coordinate, so one
// edge is the primary and the rest are mirrors that display the node without
// running anything.
//
// The role is recorded on the edge with a PointTypePrimary or PointTypeMirror
// point. An edge that carries neither belongs to a node with no primary
// location, and behaves as edges always have.

// EdgeRole describes what an edge means for the node below it.
type EdgeRole int

const (
	// EdgeRoleNone is an edge for a node with no primary location -- a
	// user, a group, a rule. Several such edges are meaningful and each
	// one runs a client. Edges created before primary and mirror edge
	// points existed also read as EdgeRoleNone.
	EdgeRoleNone EdgeRole = iota
	// EdgeRolePrimary is the one edge that owns the node. The client runs
	// here.
	EdgeRolePrimary
	// EdgeRoleMirror is an edge that exists for organization or access
	// control. No client runs here.
	EdgeRoleMirror
)

func (r EdgeRole) String() string {
	switch r {
	case EdgeRolePrimary:
		return "primary"
	case EdgeRoleMirror:
		return "mirror"
	default:
		return "none"
	}
}

// roleSet reports whether a scalar edge point of this type is set. The store
// rewrites a blank Key to "0" on the way in, so a point built in memory and
// the same point read back afterward carry different keys. Points.Find
// normalizes the key it is given but compares against Key literally, so it
// matches only the stored form. Edge roles are read on both sides of that
// boundary -- SendNode builds them, the client manager reads them back -- so
// this treats the two spellings of a scalar key as one.
func roleSet(edgePoints Points, typ string) bool {
	for _, p := range edgePoints {
		if p.Type == typ && (p.Key == "" || p.Key == "0") {
			return p.Bool()
		}
	}

	return false
}

// edgeRole reads the role from a set of edge points. An edge carrying both
// points is treated as a mirror, because declining to run a client is the safe
// direction to fail.
func edgeRole(edgePoints Points) EdgeRole {
	if roleSet(edgePoints, PointTypeMirror) {
		return EdgeRoleMirror
	}

	if roleSet(edgePoints, PointTypePrimary) {
		return EdgeRolePrimary
	}

	return EdgeRoleNone
}

// EdgeRole returns the role this edge plays for the node below it.
func (n NodeEdge) EdgeRole() EdgeRole {
	return edgeRole(n.EdgePoints)
}

// Role returns the role this edge plays for the node below it.
func (e Edge) Role() EdgeRole {
	return edgeRole(e.Points)
}

// primaryTypes own something outside the tree -- a bus, a line, a socket, this
// host's clock -- so exactly one client may act on a node of this type.
var primaryTypes = map[string]bool{
	NodeTypeModbus:               true,
	NodeTypeModbusIO:             true,
	NodeTypeOneWire:              true,
	NodeTypeOneWireIO:            true,
	NodeTypeShelly:               true,
	NodeTypeShellyIo:             true,
	NodeTypeGPIO:                 true,
	NodeTypeGPS:                  true,
	NodeTypeSerialDev:            true,
	NodeTypeCanBus:               true,
	NodeTypeParticle:             true,
	NodeTypeNetworkManager:       true,
	NodeTypeNetworkManagerDevice: true,
	NodeTypeNetworkManagerConn:   true,
	NodeTypeNTP:                  true,
	NodeTypeBrowser:              true,
	NodeTypeUpdate:               true,
	NodeTypeProvisioning:         true,
	NodeTypeProvisioningFile:     true,
	NodeTypeSync:                 true,
	NodeTypeMetrics:              true,
	// A signal generator holds no hardware, but Destination.Subject
	// resolves to its own node ID unless a destination is configured, and
	// a mirrored node is one node reached twice. Two instances would write
	// the same waveform into one point stream at twice the sample rate.
	NodeTypeSignalGenerator: true,
	// An MQTT connection subscribes to broker traffic and writes into its
	// mqttSub nodes, and its topic schema and Sparkplug builders mint node
	// IDs from an in-memory map. Two instances would double every point
	// and build two parallel node trees for the same devices.
	NodeTypeMqtt:    true,
	NodeTypeMqttSub: true,
	// Nodes the builders above create stand for a device publishing to the
	// broker.
	NodeTypeMqttDevice:      true,
	NodeTypeSparkplugGroup:  true,
	NodeTypeSparkplugNode:   true,
	NodeTypeSparkplugDevice: true,
}

// treeScopedTypes take their meaning from where they sit, so several instances
// are meaningful and each one runs a client. A db client records the subtree
// under its parent, a msgService client sees notifications raised under its
// parent, and a user mirrored into two groups holds a role in each.
var treeScopedTypes = map[string]bool{
	NodeTypeDevice:         true,
	NodeTypeUser:           true,
	NodeTypeJWT:            true,
	NodeTypeGroup:          true,
	NodeTypeDb:             true,
	NodeTypeRule:           true,
	NodeTypeCondition:      true,
	NodeTypeAction:         true,
	NodeTypeActionInactive: true,
	NodeTypeMsgService:     true,
	NodeTypeVariable:       true,
	NodeTypeFile:           true,
}

// nodeTypeOwners names the parent type a node type must live under. A node
// listed here is found by walking down from its parent rather than from the
// tree root, so moving one somewhere else leaves it inert.
var nodeTypeOwners = map[string]string{
	NodeTypeModbusIO:             NodeTypeModbus,
	NodeTypeOneWireIO:            NodeTypeOneWire,
	NodeTypeShellyIo:             NodeTypeShelly,
	NodeTypeMqttSub:              NodeTypeMqtt,
	NodeTypeCondition:            NodeTypeRule,
	NodeTypeAction:               NodeTypeRule,
	NodeTypeActionInactive:       NodeTypeRule,
	NodeTypeNetworkManagerDevice: NodeTypeNetworkManager,
	NodeTypeNetworkManagerConn:   NodeTypeNetworkManager,
	NodeTypeProvisioningFile:     NodeTypeProvisioning,
	// The Sparkplug client rebuilds its topic map by walking groups under
	// the MQTT node, edge nodes under each group, and devices under each
	// edge node, so the chain has to hold.
	NodeTypeSparkplugGroup:  NodeTypeMqtt,
	NodeTypeSparkplugNode:   NodeTypeSparkplugGroup,
	NodeTypeSparkplugDevice: NodeTypeSparkplugNode,
}

// NodeTypeIsPrimary reports whether a node of this type owns something outside
// the tree, so that exactly one of its edges may run a client. A type the
// system does not know -- one a user invents -- reports false and keeps
// behaving as it always has.
func NodeTypeIsPrimary(typ string) bool {
	return primaryTypes[typ]
}

// NodeTypeOwner returns the parent type a node of this type must live under,
// or "" when the type may live anywhere. A modbusIo is found through its
// modbus bus, so moving it elsewhere leaves it inert.
func NodeTypeOwner(typ string) string {
	return nodeTypeOwners[typ]
}
