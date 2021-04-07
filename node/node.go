package node

import (
	"bytes"
	"log"
	"text/template"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
)

// Manager is responsible for maintaining node state, running rules, etc
type Manager struct {
	db            *genji.Db
	modbusManager *ModbusManager
	nc            *natsgo.Conn
}

// NewManger creates a new Manager
func NewManger(db *genji.Db, nc *natsgo.Conn) *Manager {
	return &Manager{
		db:            db,
		modbusManager: NewModbusManager(db, nc),
		nc:            nc,
	}
}

// Run manager
func (m *Manager) Run() {
	go func() {
		for {
			m.modbusManager.Update()
			time.Sleep(1 * time.Second)
		}
	}()

	for {
		nodes, err := m.db.Nodes()
		if err != nil {
			log.Println("Error getting nodes: ", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, node := range nodes {
			// update node state
			state, changed := node.UpdateState()
			if changed {
				// FIXME this needs modified to go through NATS
				err := m.db.NodeSetState(node.ID, state)
				if err != nil {
					log.Println("Error updating node state: ", err)
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func uniqueUsers(users []data.User) []data.User {
	found := make(map[string]bool)
	ret := []data.User{}
	for _, u := range users {
		if _, present := found[u.ID]; !present {
			ret = append(ret, u)
		}
	}

	return ret
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
