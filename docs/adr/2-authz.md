# Authorization

- Author: Blake Miner
- Issue: https://github.com/simpleiot/simpleiot/issues/268
- PR / Discussion: https://github.com/simpleiot/simpleiot/pull/283
- Status: Brainstorming

## Problem

SIOT currently does not prevent unauthorized NATS clients from connecting and
publishing / subscribing. Presently, any NATS client with access to the NATS
server connection can read and write any data over the NATS connection.

## Discussion

This document describes a few mechanisms for how to implement authentication and
authorization mechanisms within Simple IoT.

### Current Authentication Mechanism

Currently, SIOT supports
[upstream connections](https://docs.simpleiot.org/docs/user/upstream.html)
through the use of upstream nodes. The connection to the upstream server can be
authenticated using a simple
[NATS auth token](https://docs.nats.io/using-nats/developer/connecting/token);
however, all NATS clients with knowledge of the auth token can read / write any
data over the NATS connection. This will not work well for a multi-tenant
application or applications where user access must be closely controlled.

Similarly, web browsers can access the NATS API using
[the WebSocket library](https://github.com/simpleiot/simpleiot/tree/master/frontend/lib),
but since they act as another NATS client, no additional security is provided;
browsers can read / write all data over the NATS connection.

## Proposal

NATS supports
[decentralized user authentication and authorization using NKeys and JSON Web Tokens (JWTs)](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt).
While robust, this authentication and authorization mechanism is rather complex
and confusing; a detailed explanation follows nonetheless. The end goal is to
dynamically add
[NATS accounts](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/accounts)
to the NATS server because publish / subscribe permissions of NATS subjects can
be tied to an account.

### Background

Each user node within SIOT will be linked to a dynamically created
[NATS account](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/accounts)
(on all [upstream nodes](https://docs.simpleiot.org/docs/user/upstream.html));
each account is generated when the user logs in. Only a single secret is stored
in the root node of the SIOT tree.

NATS has a public-key signature system based on
[Ed25519](https://en.wikipedia.org/wiki/EdDSA#Ed25519). These keypairs are
called NKeys. Put simply, NKeys allow one to cryptographically sign and verify
JWTs. An NKey not only consists of a Ed25519 private key / seed, but it also
contains information on the "role" of the key. In NATS, there are three primary
roles: operators, accounts, and users. In SIOT, there is one _operator_ for a
given NATS server, and there is one _account_ for each user node.

### Start-up

When the SIOT server starts, an NKey for the operator role is loaded from a
secret stored as a point in the root node of the tree. This point is always
stripped away when clients request the root node, so it's never transmitted over
a NATS connection. Once the NATS server is running, SIOT will start an internal
NATS client and connect to the local NATS server. This internal client will
authenticate to the NATS server with a superuser, whose account has full
permissions to publish and subscribe to all subjects. Unauthenticated NATS
clients _only_ have permission to publish to `auth`subject and listen for a
reply.

### Authentication / Login

External NATS clients (including web browsers over WebSockets) must first log
into the NATS server anonymously (using the auth token if needed) and send a
request to the `auth` subject with the username and password of a valid user
node. The default username is `admin`, and the default password is `admin`. The
internal NATS client will subscribe to requests on the `auth` subject, and if
the username / password is correct, it will respond with a user NKey and user
JWT Token, which are needed to login. The user JWT token will be issued and
signed by the account NKey, and the account NKey will be issued and signed by
the operator NKey. The NATS connection will then be re-established using the
user JWT and signing a server nonce with the user's NKey.

JWT expiration should be a configurable SIOT option and default to 1 hour.
Optionally, when the user JWT token is approaching its expiration, the NATS
client can request re-authenticate using the `auth` subject and reconnect using
the new user credentials.

### Storing NKeys

As discussed above, in the root node, we store the _seed_ needed to derive the
operator NKey. For user nodes, account and user NKeys are computed as-needed
from the node ID, the username, and the password.

### Authorization

An authenticated user will have publish / subscribe access to the subject space
of `$nodeID.>` where $nodeID is the node ID for the authenticated user. The
normal SIOT NATS API will work the same as normal with two notable exceptions:

- The API subjects are prepended with `$nodeID.`
- The "root" node is remapped to the set of parents of the logged in user node

### Examples

#### Example #1

Imagine the following a SIOT node tree:

- Root (Device 82ad…28ae)
- Power Users (Group b723…008d)
  - Temperature Sensor (Device 2820…abdc)
  - Humidity Sensor (Device a89f…eda9)
  - Blake (User ab12…ef22)
- Admin (User 920d…ab21)

In this case, logging in as "Blake" would reveal the following tree with a
single root node:

- Power Users (Group b723…008d)
  - Temperature Sensor (Device 2820…abdc)
  - Humidity Sensor (Device a89f…eda9)
  - Blake (User ab12…ef22)

To get points of the humidity sensor, one would send a request to this subject:
`ab12...ef22.p.a89f...eda9`.

#### Example #2

Imagine the following a SIOT node tree:

- Root (Device 82ad…28ae)
  - Temperature Sensor (Device 2820…abdc)
    - Blake (User ab12…ef22)
  - Humidity Sensor (Device a89f…eda9)
    - Blake (User ab12…ef22)
  - Admin (User 920d…ab21)

In this case, logging in as "Blake" would reveal the following tree with two
root nodes:

- Temperature Sensor (Device 2820…abdc)
  - Blake (User ab12…ef22)
- Humidity Sensor (Device a89f…eda9)
  - Blake (User ab12…ef22)

To get points of the humidity sensor, one would send a request to this subject:
`ab12...ef22.p.a89f...eda9`.

### Implementation Notes

```go
// Note: JWT issuer and subject must match an NKey public key
// Note: JWT issuer and subject must match roles depending on the claim NKeys

import (
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/nats-io/nats-server/v2/server"
)

// Example code to start NATS server
func StartNatsServer(o Options) {
	op, err := nkeys.CreateOperator()
	if err != nil {
		log.Fatal("Error creating NATS server: ", err)
	}
	pubKey, err := op.PublicKey()
	if err != nil {
		log.Fatal("Error creating NATS server: ", err)
	}
	acctResolver := server.MemAccResolver{}
	opts := server.Options{
		Port:                     o.Port,
		HTTPPort:                 o.HTTPPort,
		Authorization:            o.Auth,
		// First we trust all operators
		// Note: DO NOT USE conflicting `TrustedKeys` option
		TrustedOperators:         []{jwt.NewOperatorClaims(pubKey)},
		AccountResolver:          acctResolver,
	}
}

// Create an Account
acct, err := nkeys.CreateAccount()
if err != nil {
	log.Fatal("Error creating NATS account: ", err)
}
pubKey, err := acct.PublicKey()
if err != nil {
	log.Fatal("Error creating NATS account: ", err)
}
claims := jwt.NewAccountClaims{pubKey}
claims.DefaultPermissions = Permissions{
	// Note: subject `_INBOX.>` allowed for all NATS clients
	// Note: subject publish on `auth` allowed for all NATS clients
	Pub: Permission{
		Allow: StringList([]string{userNodeID+".>"}),
	},
	Sub: Permission{
		Allow: StringList([]string{userNodeID+".>"}),
	},
}
claims.Issuer = opPubKey
claims.Name = userNodeID
// Sign the JWT with the operator NKey
jwt, err := claims.Encode(op)
if err != nil {
	log.Fatal("Error creating NATS account: ", err)
}

acctResolver.Store(userNodeID, jwt)
```
