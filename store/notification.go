package store

import (
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/msg"
)

// TODO this code is currently not used and needs to be moved to a client

func (st *Store) handleNotification(msg *nats.Msg) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 2 {
		log.Println("Error in message subject: ", msg.Subject)
		return
	}

	nodeID := chunks[1]

	not, err := data.PbDecodeNotification(msg.Data)

	if err != nil {
		log.Println("Error decoding Pb notification: ", err)
		return
	}

	userNodes := []data.NodeEdge{}

	/*
		var findUsers func(id string)

		findUsers = func(id string) {
			nodes, err := st.db.getNodes(nil, "all", id, data.NodeTypeUser, false)
			if err != nil {
				log.Println("Error find user nodes: ", err)
				return
			}

			for _, n := range nodes {
				userNodes = append(userNodes, n)
			}


			// now process upstream nodes
			upIDs := st.db.edgeUp(id, false)
			if err != nil {
				log.Println("Error getting upstream nodes: ", err)
				return
			}

			for _, id := range upIDs {
				findUsers(id.Up)
			}
		}

			node, err := st.db.node(nodeID)

			if err != nil {
				log.Println("Error getting node: ", nodeID)
				return
			}

			if node.Type == data.NodeTypeUser {
				// if we notify a user node, we only want to message this node, and not walk up the tree
				nodeEdge := node.ToNodeEdge(data.Edge{Up: not.Parent})
				userNodes = append(userNodes, nodeEdge)
			} else {
				findUsers(nodeID)
			}
	*/

	for _, userNode := range userNodes {
		user, err := data.NodeToUser(userNode.ToNode())

		if err != nil {
			log.Println("Error converting node to user: ", err)
			continue
		}

		if user.Email != "" || user.Phone != "" {
			msg := data.Message{
				ID:             uuid.New().String(),
				UserID:         user.ID,
				ParentID:       userNode.Parent,
				NotificationID: nodeID,
				Email:          user.Email,
				Phone:          user.Phone,
				Subject:        not.Subject,
				Message:        not.Message,
			}

			data, err := msg.ToPb()

			if err != nil {
				log.Println("Error serializing msg to protobuf: ", err)
				continue
			}

			err = st.nc.Publish("node."+user.ID+".msg", data)

			if err != nil {
				log.Println("Error publishing message: ", err)
				continue
			}
		}
	}
}

func (st *Store) handleMessage(natsMsg *nats.Msg) {
	chunks := strings.Split(natsMsg.Subject, ".")
	if len(chunks) < 2 {
		log.Println("Error in message subject: ", natsMsg.Subject)
		return
	}

	nodeID := chunks[1]

	message, err := data.PbDecodeMessage(natsMsg.Data)

	if err != nil {
		log.Println("Error decoding Pb message: ", err)
		return
	}

	svcNodes := []data.NodeEdge{}

	var findSvcNodes func(string)

	level := 0

	findSvcNodes = func(id string) {
		nodes, err := st.db.getNodes(nil, "all", id, data.NodeTypeMsgService, false)
		if err != nil {
			log.Println("Error getting svc descendents: ", err)
			return
		}

		svcNodes = append(svcNodes, nodes...)

		// now process upstream nodes
		// if we are at the first level, only process the msg user parent, instead
		// of all user parents. This eliminates duplicate messages when a user is a
		// member of multiple groups which may have different notification services.

		var upIDs []*data.Edge

		/* FIXME this needs to be moved to client

		if level == 0 {
			upIDs = []*data.Edge{&data.Edge{Up: message.ParentID}}
		} else {
			upIDs = st.db.edgeUp(id, false)
			if err != nil {
				log.Println("Error getting upstream nodes: ", err)
				return
			}
		}
		*/

		level++

		for _, id := range upIDs {
			findSvcNodes(id.Up)
		}
	}

	findSvcNodes(nodeID)

	svcNodes = data.RemoveDuplicateNodesID(svcNodes)

	for _, svcNode := range svcNodes {
		svc, err := data.NodeToMsgService(svcNode.ToNode())
		if err != nil {
			log.Println("Error converting node to msg service: ", err)
			continue
		}

		if svc.Service == data.PointValueTwilio &&
			message.Phone != "" {
			twilio := msg.NewTwilio(svc.SID, svc.AuthToken, svc.From)

			err := twilio.SendSMS(message.Phone, message.Message)

			if err != nil {
				log.Printf("Error sending SMS to: %v: %v\n",
					message.Phone, err)
			}
		}
	}
}
