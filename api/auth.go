package api

import (
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// Auth handles user authentication requests.
type Auth struct {
	nc *nats.Conn
	// authTokenSet says whether the instance has an auth token; the reply
	// tells the UI when it has none.
	authTokenSet bool
}

// NewAuthHandler returns a new authentication handler.
func NewAuthHandler(nc *nats.Conn, authTokenSet bool) Auth {
	return Auth{nc: nc, authTokenSet: authTokenSet}
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
	if err != nil && err.Error() == data.ErrDefaultPassword.Error() {
		// the reason is safe to give: the password was right, and it is
		// refused all the same
		log.Printf("HTTP auth: default password refused for %q from %v", email, req.RemoteAddr)
		http.Error(res, err.Error(), http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(nodes) == 0 {
		log.Printf("HTTP auth: sign-in failed for %q from %v", email, req.RemoteAddr)
		http.Error(res, "invalid login", http.StatusForbidden)
		return
	}

	var token string

	for _, n := range nodes {
		if n.Type == data.NodeTypeJWT {
			p, ok := n.Points.Find(data.PointTypeToken, "")
			if ok {
				token = p.Txt()
			}
		}
	}

	err = encode(res, data.Auth{
		Token: token,
		Email: email,
		DefaultPassword: email == data.DefaultUserEmail &&
			password == data.DefaultUserPass,
		NoAuthToken: !auth.authTokenSet,
	})

	if err != nil {
		log.Println("Error encoding:", err)
	}
}
