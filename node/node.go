package node

import (
	"bytes"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

// Manager is responsible for maintaining node state, running rules, etc
type Manager struct {
	nc              *natsgo.Conn
	appVersion      string
	modbusManager   *ModbusManager
	upstreamManager *UpstreamManager
	rootNodeID      string
}

// NewManger creates a new Manager
func NewManger(nc *natsgo.Conn, appVersion string) *Manager {
	return &Manager{
		nc:         nc,
		appVersion: appVersion,
	}
}

// Init initializes the tree root node and default admin if needed
func (m *Manager) Init() error {
	rootNodes, err := nats.GetNode(m.nc, "root", "")

	var rootNode data.NodeEdge

	if len(rootNodes) > 0 {
		rootNode = rootNodes[0]
	}

	if err != nil {
		log.Println("Error getting root node: ", err)
	} else {
		m.rootNodeID = rootNode.ID
	}

	if rootNode.ID == "" {
		// initialize root node and user
		log.Println("NODE: Initialize root node and admin user")
		rootNode.Points = data.Points{
			{
				Time: time.Now(),
				Type: data.PointTypeNodeType,
				Text: data.NodeTypeDevice,
			},
		}

		rootNode.ID = uuid.New().String()

		err := nats.SendNodePoints(m.nc, rootNode.ID, rootNode.Points, true)
		if err != nil {
			return fmt.Errorf("Error setting root node points: %v", err)
		}

		err = nats.SendEdgePoint(m.nc, rootNode.ID, "", data.Point{Type: data.PointTypeTombstone, Value: 0}, true)
		if err != nil {
			return fmt.Errorf("Error sending root node edges: %w", err)
		}

		// create admin user off root node
		admin := data.User{
			ID:        uuid.New().String(),
			FirstName: "admin",
			LastName:  "user",
			Email:     "admin@admin.com",
			Pass:      "admin",
		}

		points := admin.ToPoints()
		points = append(points, data.Point{Type: data.PointTypeNodeType,
			Text: data.NodeTypeUser})

		err = nats.SendNodePoints(m.nc, admin.ID, points, true)
		if err != nil {
			return fmt.Errorf("Error setting default user: %v", err)
		}

		m.rootNodeID = rootNode.ID

		err = nats.SendEdgePoint(m.nc, admin.ID, rootNode.ID, data.Point{Type: data.PointTypeTombstone, Value: 0}, true)
		if err != nil {
			return err
		}
	}

	// check if the SW version is current
	rootNodes, err = nats.GetNode(m.nc, "root", "")

	if len(rootNodes) > 0 {
		rootNode = rootNodes[0]
	}

	ver, ok := rootNode.Points.Find(data.PointTypeVersionApp, "")
	if !ok || ver.Text != m.appVersion {
		log.Println("Setting app version: ", m.appVersion)
		err := nats.SendNodePoint(m.nc, rootNode.ID, data.Point{
			Type: data.PointTypeVersionApp,
			Text: m.appVersion,
		}, true)

		if err != nil {
			log.Println("Error setting app version")
		}
	}

	m.modbusManager = NewModbusManager(m.nc, m.rootNodeID)
	m.upstreamManager = NewUpstreamManager(m.nc, m.rootNodeID)

	return nil
}

// Run manager
func (m *Manager) Run() {
	go func() {
		// TODO: this will not scale and needs to be made event driven
		// on the creation of new nodes
		for {
			if m.modbusManager != nil {
				m.modbusManager.Update()
			}
			if m.upstreamManager != nil {
				m.upstreamManager.Update()
			}
			time.Sleep(10 * time.Second)
		}
	}()

	select {}

	/* the following code needs redone, so commenting out for now
	for {
		// TODO: this will not scale and needs to be made event driven
		nodes, err := m.db.Nodes()
		if err != nil {
			log.Println("Error getting nodes: ", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, node := range nodes {
			// update node state
			state, changed := node.GetState()
			if changed {
				p := data.Point{
					Time: time.Now(),
					Type: data.PointTypeSysState,
					Text: state,
				}

				err := nats.SendNodePoint(m.nc, node.ID, p, false)
				if err != nil {
					log.Println("Error updating node state: ", err)
				}
			}
		}

		time.Sleep(30 * time.Second)
	}
	*/
}

type nodeTemplateData struct {
	ID          string
	Description string
	Ios         map[string]float64
}

func renderNotifyTemplate(node *data.Node, msgTemplate string) (string, error) {
	// build map of IO values so they are easy to reference by type or ID in template
	dtd := nodeTemplateData{
		ID:          node.ID,
		Description: node.Desc(),
		Ios:         make(map[string]float64),
	}

	for _, io := range node.Points {
		if io.Type != "" {
			dtd.Ios[io.Type] = io.Value
		}
	}

	buf := new(bytes.Buffer)

	tmpl, err := template.New("msg").Parse(msgTemplate)

	if err != nil {
		return "", err
	}

	err = tmpl.Execute(buf, dtd)

	if err != nil {
		return "", err
	}

	return buf.String(), nil

}
