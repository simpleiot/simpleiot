package node

import (
	"bytes"
	"fmt"
	"log"
	"text/template"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/msg"
)

// Manager is responsible for maintaining node state, running rules, etc
type Manager struct {
	db            *genji.Db
	messenger     *msg.Messenger
	modbusManager *ModbusManager
	nc            *natsgo.Conn
}

// NewManger creates a new Manager
func NewManger(db *genji.Db, messenger *msg.Messenger, nc *natsgo.Conn) *Manager {
	return &Manager{
		db:            db,
		messenger:     messenger,
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

func (m *Manager) notify(node *data.Node, ruleDesc, msgTemplate string, groups []string) error {
	// find users for the groups
	var users []data.User
	for _, gID := range groups {
		us, err := m.db.UsersForGroup(gID)
		if err != nil {
			log.Printf("Error getting users for group %v: %v\n", gID, err)
			continue
		}
		users = append(users, us...)
	}

	uniqueUsers := uniqueUsers(users)

	// send notification to all users
	var msg string
	if msgTemplate == "" {
		msg = fmt.Sprintf("Notification: %v at %v fired", ruleDesc, node.Desc())
	} else {
		var err error
		msg, err = renderNotifyTemplate(node, msgTemplate)
		if err != nil {
			log.Printf("Error rendering template %v: %v\n",
				msgTemplate, err)
			msg = fmt.Sprintf("Notification: %v at %v fired", ruleDesc, node.Desc())
		}
	}

	for _, u := range uniqueUsers {
		if u.Phone != "" {
			if m.messenger != nil {
				log.Printf("Sending SMS to %v %v: %v\n", u.FirstName, u.LastName, msg)
				err := m.messenger.SendSMS(u.Phone, msg)
				if err != nil {
					log.Printf("Error sending SMS to %v: %v\n", u.Phone, err)
				}
			}
		}
	}

	return nil
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
