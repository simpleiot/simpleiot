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
	ID          string    `node:"id"`
	Parent      string    `node:"parent"`
	Description string    `point:"description"`
	URI         string    `point:"uri"`
	Disabled    bool      `point:"disabled"`
	Debug       int       `point:"debug"`
	Error       string    `point:"error"`
	Subs        []MqttSub `child:"mqttSub"`
}

// errMqttExternalBroker is set on the node when a URI is configured. Connecting
// to an external broker needs an MQTT client and is not implemented yet.
const errMqttExternalBroker = "external MQTT brokers are not supported yet; leave the URI blank to use the built-in broker"

// mqttMsg carries a message from a NATS subscription callback into the client
// run loop, where the configuration can be read without a lock.
type mqttMsg struct {
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
			if err := data.MergePoints(pts.ID, pts.Points, &c.config); err != nil {
				log.Println("MQTT: error merging new points:", err)
			}

			c.sync()

		case pts := <-c.newEdgePoints:
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

	c.syncTopicTags()
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
