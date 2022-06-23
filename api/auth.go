package api

import (
	"net/http"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// Auth handles user authentication requests.
type Auth struct {
	nc *natsgo.Conn
}

// NewAuthHandler returns a new authentication handler using the given key.
func NewAuthHandler(nc *natsgo.Conn) Auth {
	return Auth{nc: nc}
}

// ServeHTTP serves requests to authenticate.
func (auth Auth) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	email := req.FormValue("email")
	password := req.FormValue("password")

	nodes, err := client.UserCheck(auth.nc, email, password)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(nodes) < 0 {
		http.Error(res, "invalid login", http.StatusForbidden)
		return
	}

	var token string

	for _, n := range nodes {
		if n.Type == data.NodeTypeJWT {
			p, ok := n.Points.Find(data.PointTypeToken, "")
			if ok {
				token = p.Text
			}
		}
	}

	encode(res, data.Auth{
		Token: token,
		Email: email,
	})
}
