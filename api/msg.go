package api

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/msg"
)

// Msg handles user requests.
type Msg struct {
	db        *genji.Db
	validator RequestValidator
	messenger *msg.Messenger
}

// NewMsgHandler returns a new handler for sending messages.
func NewMsgHandler(db *genji.Db, v RequestValidator, messenger *msg.Messenger) Msg {
	return Msg{db: db, validator: v, messenger: messenger}
}

// ServeHTTP serves user requests.
func (m Msg) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validUser, userID := m.validator.Valid(req)
	if !validUser {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// only allow requests if user is part of root org
	isRoot, err := m.db.UserIsRoot(userUUID)

	if !isRoot {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch req.Method {
	case http.MethodPost:
		var point data.Point
		if err := decode(req.Body, &point); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if point.Text != "" {
			// send message to all users
			users, err := m.db.Users()
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}

			for _, u := range users {
				if u.Phone != "" {
					err := m.messenger.SendSMS(u.Phone, point.Text)
					if err != nil {
						log.Println("Error sending message: ", err)
					}
				}
			}
		}

	default:
		http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: ""})
}
