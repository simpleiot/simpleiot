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
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/nats"
)

// Manager is responsible for maintaining node state, running rules, etc
type Manager struct {
	db              *db.Db
	nc              *natsgo.Conn
	modbusManager   *ModbusManager
	upstreamManager *UpstreamManager
}

// NewManger creates a new Manager
func NewManger(db *db.Db, nc *natsgo.Conn) *Manager {
	return &Manager{
		db:              db,
		nc:              nc,
		modbusManager:   NewModbusManager(db, nc),
		upstreamManager: NewUpstreamManager(nc),
	}
}

// Init initializes the tree root node and default admin if needed
func (m *Manager) Init() error {
	rootID := m.db.RootNodeID()
	if rootID == "" {
		// initialize root node and user
		p := data.Point{
			Time: time.Now(),
			Type: data.PointTypeNodeType,
			Text: data.NodeTypeDevice,
		}

		id := uuid.New().String()

		err := nats.SendNodePoint(m.nc, id, p, false)
		if err != nil {
			return fmt.Errorf("Error setting root node: %v", err)
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

		err = nats.SendNodePoint(m.nc, id, p, false)
		if err != nil {
			return fmt.Errorf("Error setting default user: %v", err)
		}
	}

	return nil
}

// Run manager
func (m *Manager) Run() {
	go func() {
		// TODO: this will not scale and needs to be made event driven
		// on the creation of new nodes
		for {
			m.modbusManager.Update()
			m.upstreamManager.Update()
			time.Sleep(10 * time.Second)
		}
	}()

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
		if io.ID != "" {
			dtd.Ios[io.ID] = io.Value
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
