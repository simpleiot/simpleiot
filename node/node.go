package node

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/system"
)

// Manager is responsible for maintaining node state, running rules, etc
type Manager struct {
	nc             *nats.Conn
	appVersion     string
	osVersionField string
	modbusManager  *ModbusManager
	rootNodeID     string
	oneWireManager *oneWireManager
	chStop         chan struct{}
}

// NewManger creates a new Manager
func NewManger(nc *nats.Conn, appVersion, osVersionField string) *Manager {
	return &Manager{
		nc:             nc,
		appVersion:     appVersion,
		osVersionField: osVersionField,
		chStop:         make(chan struct{}),
	}
}

// Init initializes some node managers
func (m *Manager) init() error {
	var rootNode data.NodeEdge

	rootNodes, err := client.GetNodes(m.nc, "root", "all", "", false)

	if err != nil {
		return fmt.Errorf("error getting root node: %v", err)
	}

	if len(rootNodes) > 0 {
		rootNode = rootNodes[0]
	} else {
		return fmt.Errorf("no nodes returned for root node")
	}

	m.rootNodeID = rootNode.ID

	appVer, ok := rootNode.Points.Find(data.PointTypeVersionApp, "")
	if !ok || appVer.Text != m.appVersion {
		log.Println("Setting app version:", m.appVersion)
		err := client.SendNodePoint(m.nc, rootNode.ID, data.Point{
			Type: data.PointTypeVersionApp,
			Text: m.appVersion,
		}, true)

		if err != nil {
			log.Println("Error setting app version")
		}
	}

	// check if OS version is current
	osVer, err := system.ReadOSVersion(m.osVersionField)
	if err != nil {
		log.Println("Error reading OS version:", err)
	} else {
		log.Println("OS version:", osVer)
		osVerStored, ok := rootNode.Points.Find(data.PointTypeVersionOS, "")
		if !ok || osVer.String() != osVerStored.Text {
			log.Println("Setting os version:", osVer)
			err := client.SendNodePoint(m.nc, rootNode.ID, data.Point{
				Type: data.PointTypeVersionOS,
				Text: osVer.String(),
			}, true)

			if err != nil {
				log.Println("Error setting OS version")
			}
		}

	}

	m.modbusManager = NewModbusManager(m.nc, m.rootNodeID)
	m.oneWireManager = newOneWireManager(m.nc, m.rootNodeID)

	return nil
}

// Start manager
func (m *Manager) Start() error {
	if err := m.init(); err != nil {
		return fmt.Errorf("error initializing nodes: %v", err)
	}

	t := time.NewTimer(time.Millisecond)

	// TODO: this will not scale and needs to be made event driven
	// on the creation of new nodes
	for {
		select {
		case <-m.chStop:
			return errors.New("node manager stopping")
		case <-t.C:
			if m.modbusManager != nil {
				_ = m.modbusManager.Update()
			}
			if m.oneWireManager != nil {
				_ = m.oneWireManager.update()
			}
			t.Reset(time.Second * 20)
		}
	}

	/* the following code needs redone, so commenting out for now
	for {
		// TODO: this will not scale and needs to be made event driven
		nodes, err := m.db.Nodes()
		if err != nil {
			log.Println("Error getting nodes:", err)
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

				err := client.SendNodePoint(m.nc, node.ID, p, false)
				if err != nil {
					log.Println("Error updating node state:", err)
				}
			}
		}

		time.Sleep(30 * time.Second)
	}
	*/
}

// Stop manager
func (m *Manager) Stop(_ error) {
	close(m.chStop)
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
