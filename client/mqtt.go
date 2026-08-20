package client

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Mqtt describes an MQTT connection. A blank URI uses the MQTT broker built
// into this instance, which is the only mode supported today -- messages
// published to it arrive as NATS subjects, so the subscriptions below are
// ordinary NATS subscriptions and no MQTT client library is involved.
type Mqtt struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	URI         string `point:"uri"`
	Disabled    bool   `point:"disabled"`
	Debug       int    `point:"debug"`
	Error       string `point:"error"`
	// Sparkplug enables Sparkplug B handling, which builds the nodes below
	// this one from the birth certificates edge nodes publish.
	Sparkplug bool      `point:"sparkplug"`
	Subs      []MqttSub `child:"mqttSub"`
}

// errMqttExternalBroker is set on the node when a URI is configured. Connecting
// to an external broker needs an MQTT client and is not implemented yet.
const errMqttExternalBroker = "external MQTT brokers are not supported yet; leave the URI blank to use the built-in broker"

// mqttMsg carries a message from a NATS subscription callback into the client
// run loop, where the configuration can be read without a lock.
type mqttMsg struct {
	// subID names the mqttSub node the message is for, and is empty for a
	// message from the Sparkplug subscription
	subID   string
	subject string
	data    []byte
}

// MqttClient subscribes to MQTT topics and publishes what arrives as points.
type MqttClient struct {
	nc            *nats.Conn
	config        Mqtt
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	msgs          chan mqttMsg

	// subs holds the live NATS subscription for each mqttSub node
	subs map[string]*nats.Subscription
	// subErr records the error last reported for each node, so an error is
	// written once rather than on every message
	subErr map[string]string

	// spSub is the subscription covering the Sparkplug namespace, and sp
	// holds what the Sparkplug handler has learned
	spSub *nats.Subscription
	sp    *sparkplugState
}

// NewMqttClient returns a new MQTT client for the given node
func NewMqttClient(nc *nats.Conn, config Mqtt) Client {
	return &MqttClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		msgs:          make(chan mqttMsg, 32),
		subs:          make(map[string]*nats.Subscription),
		subErr:        make(map[string]string),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (c *MqttClient) Run() error {
	log.Println("Starting MQTT client:", c.config.Description)

	c.sync()

done:
	for {
		select {
		case <-c.stop:
			break done

		case pts := <-c.newPoints:
			// points also arrive for the nodes Sparkplug creates below this
			// one, which are not part of this client's configuration
			if !c.owns(pts.ID) {
				break
			}

			if err := data.MergePoints(pts.ID, pts.Points, &c.config); err != nil {
				log.Println("MQTT: error merging new points:", err)
			}

			c.sync()

		case pts := <-c.newEdgePoints:
			if !c.owns(pts.ID) {
				break
			}

			if err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &c.config); err != nil {
				log.Println("MQTT: error merging new edge points:", err)
			}

		case m := <-c.msgs:
			c.handle(m)
		}
	}

	log.Println("Stopping MQTT client:", c.config.Description)

	for id, sub := range c.subs {
		_ = sub.Unsubscribe()
		delete(c.subs, id)
	}

	c.stopSparkplug()

	return nil
}

// Stop sends a signal to the Run function to exit
func (c *MqttClient) Stop(_ error) {
	close(c.stop)
}

// Points is called by the Manager when new points for this node are received.
func (c *MqttClient) Points(nodeID string, points []data.Point) {
	c.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node are
// received.
func (c *MqttClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	c.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

// owns reports whether a node is part of this client's configuration -- the
// mqtt node itself or one of its subscription children.
func (c *MqttClient) owns(id string) bool {
	if id == c.config.ID {
		return true
	}

	for _, s := range c.config.Subs {
		if s.ID == id {
			return true
		}
	}

	return false
}

// sync brings the live subscriptions in line with the configuration, which is
// how a topic edit, a disable, or an added subscription takes effect.
func (c *MqttClient) sync() {
	desired := make(map[string]string)

	if c.config.URI != "" {
		c.setError(c.config.ID, errMqttExternalBroker)
	} else {
		c.setError(c.config.ID, "")
	}

	if !c.config.Disabled && c.config.URI == "" {
		for _, s := range c.config.Subs {
			if s.Disabled || s.Topic == "" {
				continue
			}

			subject, err := data.MQTTFilterToSubject(s.Topic)
			if err != nil {
				c.setError(s.ID, err.Error())
				continue
			}

			desired[s.ID] = subject
		}
	}

	for id, sub := range c.subs {
		if desired[id] == sub.Subject {
			continue
		}

		_ = sub.Unsubscribe()
		delete(c.subs, id)
	}

	for id, subject := range desired {
		if _, running := c.subs[id]; running {
			continue
		}

		sub, err := c.nc.Subscribe(subject, func(m *nats.Msg) {
			select {
			case c.msgs <- mqttMsg{subID: id, subject: m.Subject, data: m.Data}:
			case <-c.stop:
			}
		})
		if err != nil {
			c.setError(id, err.Error())
			continue
		}

		if c.config.Debug > 0 {
			log.Printf("MQTT %v: subscribed to %v\n", c.config.Description, subject)
		}

		c.subs[id] = sub
		c.setError(id, "")
	}

	c.syncSparkplug()
	c.syncTopicTags()
}

// syncSparkplug starts or stops the subscription covering the Sparkplug
// namespace as the sparkplug point on this node changes.
func (c *MqttClient) syncSparkplug() {
	want := c.config.Sparkplug && !c.config.Disabled && c.config.URI == ""

	if !want {
		c.stopSparkplug()
		return
	}

	if c.spSub != nil {
		// keep the handler's logging in step with edits to this node
		c.sp.desc = c.config.Description
		c.sp.debug = c.config.Debug
		return
	}

	subject, err := data.MQTTFilterToSubject(sparkplugFilter)
	if err != nil {
		c.setError(c.config.ID, err.Error())
		return
	}

	sp := newSparkplugState(c.nc, c.config.ID, c.config.Description, c.config.Debug)

	if err := sp.load(); err != nil {
		c.setError(c.config.ID, err.Error())
		return
	}

	sub, err := c.nc.Subscribe(subject, func(m *nats.Msg) {
		select {
		case c.msgs <- mqttMsg{subject: m.Subject, data: m.Data}:
		case <-c.stop:
		}
	})
	if err != nil {
		c.setError(c.config.ID, err.Error())
		return
	}

	log.Printf("MQTT %v: Sparkplug B enabled\n", c.config.Description)

	c.sp = sp
	c.spSub = sub
}

func (c *MqttClient) stopSparkplug() {
	if c.spSub == nil {
		return
	}

	_ = c.spSub.Unsubscribe()
	c.spSub = nil
	c.sp = nil
}

// syncTopicTags keeps a tag point named topic on each subscription node
// holding the full topic, so a series can be traced back to the message that
// produced it. See the graphing section of docs/user/plc.md.
func (c *MqttClient) syncTopicTags() {
	for _, s := range c.config.Subs {
		if s.Topic == "" || s.Tags[data.PointTypeTopic] == s.Topic {
			continue
		}

		p := data.NewPointString(data.PointTypeTag, data.PointTypeTopic, s.Topic)
		p.Origin = c.config.ID

		if err := SendNodePoint(c.nc, s.ID, p, false); err != nil {
			log.Println("MQTT: error sending topic tag:", err)
		}
	}
}

// handle turns one message into points on the subscription node.
func (c *MqttClient) handle(m mqttMsg) {
	if m.subID == "" {
		if c.sp != nil {
			c.sp.handle(data.MQTTSubjectToTopic(m.subject), m.data)
		}
		return
	}

	var sub *MqttSub

	for i := range c.config.Subs {
		if c.config.Subs[i].ID == m.subID {
			sub = &c.config.Subs[i]
			break
		}
	}

	if sub == nil {
		// the subscription was removed between delivery and here
		return
	}

	if c.config.Debug > 0 {
		log.Printf("MQTT %v: %v: %s\n", c.config.Description,
			data.MQTTSubjectToTopic(m.subject), m.data)
	}

	pts, err := sub.points(m.data)
	if err != nil {
		c.setError(sub.ID, err.Error())
		return
	}

	for i := range pts {
		pts[i].Origin = c.config.ID
	}

	if err := SendNodePoints(c.nc, sub.ID, pts, false); err != nil {
		log.Println("MQTT: error sending points:", err)
		return
	}

	c.setError(sub.ID, "")
}

// setError writes an error point on a node when what it reports changes, so a
// topic that fails on every message reports once rather than continuously.
func (c *MqttClient) setError(nodeID, msg string) {
	if c.subErr[nodeID] == msg {
		return
	}

	c.subErr[nodeID] = msg

	if msg != "" {
		log.Printf("MQTT %v: %v\n", c.config.Description, msg)
	}

	p := data.NewPointString(data.PointTypeError, "", msg)

	if nodeID != c.config.ID {
		p.Origin = c.config.ID
	}

	if err := SendNodePoint(c.nc, nodeID, p, false); err != nil {
		log.Println("MQTT: error sending error point:", err)
	}
}
