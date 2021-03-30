package api

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/nats"
)

// NatsHandler implements the SIOT NATS api
type NatsHandler struct {
	server    string
	Nc        *natsgo.Conn
	db        *genji.Db
	authToken string
	lock      sync.Mutex
	updates   map[string]time.Time
}

// NewNatsHandler creates a new NATS client for handling SIOT requests
func NewNatsHandler(db *genji.Db, authToken, server string) *NatsHandler {
	log.Println("NATS handler connecting to: ", server)
	return &NatsHandler{
		db:        db,
		authToken: authToken,
		updates:   make(map[string]time.Time),
		server:    server,
	}
}

// Connect to NATS server and set up handlers for things we are interested in
func (nh *NatsHandler) Connect() (*natsgo.Conn, error) {
	nc, err := natsgo.Connect(nh.server,
		natsgo.Timeout(10*time.Second),
		natsgo.PingInterval(60*5*time.Second),
		natsgo.MaxPingsOutstanding(5),
		natsgo.ReconnectBufSize(5*1024*1024),
		natsgo.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		natsgo.Token(nh.authToken),
	)

	if err != nil {
		return nil, err
	}

	nh.Nc = nc

	if _, err := nc.Subscribe("node.*.samples", nh.handlePoints); err != nil {
		return nil, fmt.Errorf("Subscribe node samples error: %w", err)
	}

	if _, err := nc.Subscribe("node.*.points", nh.handlePoints); err != nil {
		return nil, fmt.Errorf("Subscribe node points error: %w", err)
	}

	if _, err := nc.Subscribe("node.*", nh.handleNode); err != nil {
		return nil, fmt.Errorf("Subscribe node error: %w", err)
	}

	return nc, nil
}

// StartUpdate starts an update
func (nh *NatsHandler) StartUpdate(id, url string) error {
	nh.lock.Lock()
	defer nh.lock.Unlock()

	if _, ok := nh.updates[id]; ok {
		return fmt.Errorf("Update already in process for dev: %v", id)
	}

	nh.updates[id] = time.Now()

	err := nh.db.NodeSetSwUpdateState(id, data.SwUpdateState{
		Running: true,
	})

	if err != nil {
		delete(nh.updates, id)
		return err
	}

	go func() {
		err := NatsSendFileFromHTTP(nh.Nc, id, url, func(bytesTx int) {
			err := nh.db.NodeSetSwUpdateState(id, data.SwUpdateState{
				Running:     true,
				PercentDone: bytesTx,
			})

			if err != nil {
				log.Println("Error setting update status in DB: ", err)
			}
		})

		state := data.SwUpdateState{
			Running: false,
		}

		if err != nil {
			state.Error = "Error updating software"
			state.PercentDone = 0
		} else {
			state.PercentDone = 100
		}

		nh.lock.Lock()
		delete(nh.updates, id)
		nh.lock.Unlock()

		err = nh.db.NodeSetSwUpdateState(id, state)
		if err != nil {
			log.Println("Error setting sw update state: ", err)
		}
	}()

	return nil
}

// FIXME consider moving this to db package and then unexporting the NodePoint method
func (nh *NatsHandler) handlePoints(msg *natsgo.Msg) {
	nodeID, points, err := nats.DecodeNodeMsg(msg)

	if err != nil {
		fmt.Errorf("Error decoding nats message: %v: %v", msg.Subject, err)
		nh.reply(msg.Reply, errors.New("error decoding node samples subject"))
		return
	}

	for _, p := range points {
		err = nh.db.NodePoint(nodeID, p)
		if err != nil {
			// TODO track error stats
			log.Printf("Error writing nodeID (%v) point (%+v) to Db: %v", nodeID, p, err)
			log.Println("msg subject: ", msg.Subject)
			nh.reply(msg.Reply, err)
			return
		}

		err = nh.processPointUpstream(nodeID, nodeID, p)
		if err != nil {
			// TODO track error stats
			log.Println("Error processing point in upstream nodes: ", err)
		}
	}

	nh.reply(msg.Reply, nil)
}

func (nh *NatsHandler) handleNode(msg *natsgo.Msg) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 2 {
		log.Println("Error in message subject: ", msg.Subject)
		return
	}

	nodeID := chunks[1]

	node, err := nh.db.Node(nodeID)

	if err != nil {
		log.Printf("NATS: Error getting node %v from db: %v\n", nodeID, err)
		// TODO should we send an error back to requester
	}

	data, err := data.PbEncodeNode(node)

	if err != nil {
		log.Printf("Error pb encoding node: %v\n", err)
		// TODO send error back to client
	}

	err = nh.Nc.Publish(msg.Reply, data)

	if err != nil {
		log.Println("NATS: Error publishing response to node request: ", err)
	}
}

// used for messages that want an ACK
func (nh *NatsHandler) reply(subject string, err error) {
	if subject == "" {
		// node is not expecting a reply
		return
	}

	reply := ""

	if err != nil {
		reply = err.Error()
	}

	nh.Nc.Publish(subject, []byte(reply))
}

func (nh *NatsHandler) processPointUpstream(currentNodeID, nodeID string, p data.Point) error {

	if currentNodeID == nh.db.RootNodeID() {
		// we are at, stop
		return nil
	}

	upIDs, err := nh.db.EdgeUp(currentNodeID)
	if err != nil {
		return err
	}

	for _, id := range upIDs {
		// get children and process any rules
		ruleNodes, err := nh.db.NodeDescendents(id, data.NodeTypeRule, false)
		if err != nil {
			return err
		}

		for _, ruleNode := range ruleNodes {
			conditionNodes, err := nh.db.NodeDescendents(ruleNode.ID, data.NodeTypeCondition, false)
			if err != nil {
				return err
			}

			actionNodes, err := nh.db.NodeDescendents(ruleNode.ID, data.NodeTypeAction, false)
			if err != nil {
				return err
			}

			rule, err := data.NodeToRule(ruleNode, conditionNodes, actionNodes)

			if err != nil {
				return err
			}

			fmt.Printf("CLIFF: rule: %+v\n", rule)

			active, err := ruleProcessPoint(nh.Nc, rule, nodeID, p)

			if err != nil {
				log.Println("Error processing rule point: ", err)
			}

			if active {
				err := ruleRunActions(rule, nh.Nc)
				if err != nil {
					log.Println("Error running rule actions: ", err)
				}
			}
		}

		err = nh.processPointUpstream(id, nodeID, p)
		if err != nil {
			log.Println("Rules -- error processing upstream node: ", err)
		}
	}

	return nil
}

// processPoint runs a point through a rules conditions and and updates condition
// and rule active status. Returns true if point was processed and active is true
func ruleProcessPoint(nc *natsgo.Conn, r *data.Rule, nodeID string, p data.Point) (bool, error) {
	fmt.Println("CLIFF: ruleProcessPoint, nodeID: ", nodeID)
	fmt.Printf("CLIFF: point: %+v\n", p)
	allActive := true
	pointProcessed := false
	for _, c := range r.Conditions {
		if c.NodeID != "" && c.NodeID != nodeID {
			fmt.Println("CLIFF: node id does not match, continue")
			continue
		}

		if c.PointID != "" && c.PointID != p.ID {
			fmt.Println("CLIFF: point id does not match, continue")
			continue
		}

		if c.PointType != "" && c.PointType != p.Type {
			fmt.Println("CLIFF: point type does not match, continue")
			continue
		}

		if c.PointIndex != -1 && c.PointIndex != int(p.Index) {
			fmt.Println("CLIFF: point index does not match, continue")
			continue
		}

		fmt.Println("CLIFF: processing condition, PointValueType: ", c.PointValueType)

		var active bool

		pointProcessed = true

		// conditions match, so check value
		switch c.PointValueType {
		case data.PointValueNumber:
			switch c.Operator {
			case data.PointValueGreaterThan:
				active = p.Value > c.PointValue
			case data.PointValueLessThan:
				active = p.Value < c.PointValue
			case data.PointValueEqual:
				active = p.Value == c.PointValue
			case data.PointValueNotEqual:
				active = p.Value != c.PointValue
			}
		case data.PointValueText:
			switch c.Operator {
			case data.PointValueEqual:
			case data.PointValueNotEqual:
			case data.PointValueContains:
			}
		case data.PointValueOnOff:
			condValue := c.PointValue != 0
			pointValue := p.Value != 0
			fmt.Println("CLIFF: condValue, pointValue: ", condValue, pointValue)
			active = condValue == pointValue
		}

		if !active {
			allActive = false
		}

		fmt.Println("CLIFF: active: ", active)

		if active != c.Active {
			// update condition
			p := data.Point{
				Type:  data.PointTypeActive,
				Time:  time.Now(),
				Value: data.BoolToFloat(active),
			}

			err := nats.SendPoint(nc, c.ID, p, false)
			if err != nil {
				log.Println("Rule error sending point: ", err)
			}
		}
	}
	fmt.Println("CLIFF: allActive: ", allActive)

	if pointProcessed && allActive != r.Active {
		p := data.Point{
			Type:  data.PointTypeActive,
			Time:  time.Now(),
			Value: data.BoolToFloat(allActive),
		}

		err := nats.SendPoint(nc, r.ID, p, false)
		if err != nil {
			log.Println("Rule error sending point: ", err)
		}
	}

	if pointProcessed && allActive {
		return true, nil
	}

	return false, nil
}

// ruleRunActions runs rule actions
func ruleRunActions(r *data.Rule, nc *natsgo.Conn) error {
	for _, a := range r.Actions {
		switch a.Action {
		case data.PointValueActionSetValue:
			if a.NodeID == "" {
				log.Println("Error, node action nodeID must be set, action id: ", a.ID)
			}
			p := data.Point{
				Time:  time.Now(),
				Type:  a.PointType,
				Value: a.PointValue,
				Text:  a.PointTextValue,
			}
			err := nats.SendPoint(nc, a.NodeID, p, false)
			if err != nil {
				log.Println("Error sending rule action point: ", err)
			}
		case data.PointValueActionNotify:
			log.Println("Notify action not supported yet")
		default:
			log.Println("Uknown rule action: ", a.Action)
		}
	}
	return nil
}
