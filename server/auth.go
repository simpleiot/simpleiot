package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// Device authentication modes, selected with --deviceAuth or
// SIOT_DEVICE_AUTH.
const (
	// DeviceAuthOptional accepts the shared token from anywhere, as well as
	// device credentials.
	DeviceAuthOptional = "optional"
	// DeviceAuthRequired accepts the shared token only from loopback
	// connections (the server's own client and the CLI on the same host),
	// so every remote connection has to present a device credential.
	DeviceAuthRequired = "required"
)

// authReconcilePeriod is how often the authorizer compares the live
// connections with its index: connections whose credential is gone are
// closed, and the connected status of each credential is brought up to
// date.
const authReconcilePeriod = 5 * time.Second

// credEntry is what the authorizer knows about one credential.
type credEntry struct {
	credID   string
	deviceID string
	// disabled is set when the credential is disabled, or when it does not
	// authorize anything: it has no live parent, more than one, or the
	// device it sits under has been deleted. The entry stays in the index
	// so a change can be told from a credential the tree never had.
	disabled bool
	// connected tracks the connected point on the credential node so the
	// authorizer sends it only when it changes.
	connected bool
}

// enrollEntry is what the authorizer knows about one enrollment token.
type enrollEntry struct {
	nodeID      string
	disabled    bool
	autoApprove bool
	expires     int64
}

// enrollUserPrefix names an enrollment-token connection in the server's
// connection list, so enforce can find it when the token is revoked.
const enrollUserPrefix = "enroll:"

// authConn records what a device connection was granted, so a later
// change to the index or to the streams in its boundary can be enforced by
// closing it.
type authConn struct {
	pubKey string
	grant  string
}

// authorizer authenticates every connection to the embedded NATS server.
// The shared token grants full access, as it always has. An NKey whose
// public key is in a deviceCred node grants exactly the subjects the device
// under that node needs to sync (see devicePermissions). The index is held
// in memory so Check, which runs inside the server's connection handling,
// never waits on a NATS request.
type authorizer struct {
	token      string
	deviceAuth string

	mu sync.RWMutex
	// ready is set once the index has been loaded from the store; until
	// then only the shared token is accepted and devices retry.
	ready  bool
	rootID string
	// creds indexes credentials by public key.
	creds map[string]*credEntry
	// credIDs maps every credential node ID seen to its public key, which
	// is blank until the key is set, so a point on a credential node can be
	// tied back to the entry.
	credIDs map[string]string
	// devices maps a device ID to the credential IDs under it.
	devices map[string]map[string]bool
	// enroll indexes enrollment tokens by hash, and enrollIDs maps the
	// node ID back to the hash.
	enroll    map[string]*enrollEntry
	enrollIDs map[string]string
	// origins lists, for each boundary, the origins that have a stream on
	// this instance. Permissions for the pull side are enumerated from it.
	origins map[string]map[string]bool
	// conns records live device connections by connection ID.
	conns map[uint64]authConn

	nc  *nats.Conn
	ns  *server.Server
	sub *nats.Subscription
	adv *nats.Subscription
	// work carries index maintenance out of the NATS callbacks.
	work chan func()
	// connects carries successful device authentications out of Check.
	connects chan string
	stop     chan struct{}
	done     chan struct{}
}

func newAuthorizer(token, deviceAuth string) *authorizer {
	if deviceAuth == "" {
		deviceAuth = DeviceAuthOptional
	}

	return &authorizer{
		token:      token,
		deviceAuth: deviceAuth,
		creds:      make(map[string]*credEntry),
		credIDs:    make(map[string]string),
		devices:    make(map[string]map[string]bool),
		enroll:     make(map[string]*enrollEntry),
		enrollIDs:  make(map[string]string),
		origins:    make(map[string]map[string]bool),
		conns:      make(map[uint64]authConn),
		work:       make(chan func(), 1024),
		connects:   make(chan string, 256),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Check implements server.Authentication.
func (a *authorizer) Check(c server.ClientAuthentication) bool {
	opts := c.GetOpts()

	if opts.Nkey != "" {
		return a.checkNkey(c, opts)
	}

	return a.checkToken(c, opts)
}

// checkToken accepts the shared token, or anything at all when no token is
// configured, which is how an instance has always run. An enrollment token
// is accepted with permission to ask for a credential and nothing else.
func (a *authorizer) checkToken(c server.ClientAuthentication, opts *server.ClientOpts) bool {
	if a.token == "" {
		return true
	}

	if opts.Token != a.token {
		if id, ok := a.enrollNode(opts.Token); ok {
			c.RegisterUser(&server.User{
				Username:    enrollUserPrefix + id,
				Permissions: enrollPermissions(),
			})
			return true
		}
		return false
	}

	if a.deviceAuth == DeviceAuthRequired && !isLoopback(c.RemoteAddress()) {
		log.Printf("NATS auth: refusing shared token from %v, device auth is required",
			c.RemoteAddress())
		return false
	}

	return true
}

// checkNkey verifies the nonce signature, looks the public key up, and
// registers the device's permissions with the connection.
func (a *authorizer) checkNkey(c server.ClientAuthentication, opts *server.ClientOpts) bool {
	if !verifyNonce(opts.Nkey, opts.Sig, c.GetNonce()) {
		log.Printf("NATS auth: bad signature for %v", opts.Nkey)
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.ready {
		if a.token == "" {
			return true
		}
		log.Printf("NATS auth: refusing %v, credentials not loaded yet", opts.Nkey)
		return false
	}

	e, ok := a.creds[opts.Nkey]
	if !ok && a.token == "" {
		// an open instance accepts everyone; a key it does not know is
		// no different from no credentials at all
		return true
	}
	if !ok || e.disabled {
		log.Printf("NATS auth: refusing unknown or disabled credential %v", opts.Nkey)
		return false
	}

	origins := a.originsFor(e.deviceID)
	perms := devicePermissions(e.deviceID, a.rootID, origins)

	c.RegisterUser(&server.User{Username: opts.Nkey, Permissions: perms})

	a.conns[c.GetID()] = authConn{
		pubKey: opts.Nkey,
		grant:  strings.Join(origins, ","),
	}

	select {
	case a.connects <- opts.Nkey:
	default:
	}

	return true
}

// root is this instance's root node ID.
func (a *authorizer) root() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.rootID
}

// enrollNode returns the node ID of the live enrollment token a value
// matches, if any.
func (a *authorizer) enrollNode(token string) (string, bool) {
	if token == "" {
		return "", false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	e, ok := a.enroll[client.HashEnrollToken(token)]
	if !a.ready || !ok || !e.live() {
		return "", false
	}

	return e.nodeID, true
}

func (e *enrollEntry) live() bool {
	return !e.disabled && (e.expires == 0 || time.Now().Unix() < e.expires)
}

// EnrollToken reports whether an enrollment token is live and whether
// credentials it enrolls are approved straight away.
func (a *authorizer) EnrollToken(token string) (autoApprove, ok bool) {
	id, ok := a.enrollNode(token)
	if !ok {
		return false, false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.enroll[a.enrollIDs[id]].autoApprove, true
}

// enrollPermissions is all an enrollment token allows: one request.
func enrollPermissions() *server.Permissions {
	return &server.Permissions{
		Publish:   &server.SubjectPermission{Allow: []string{client.SubjectEnrollRequest}},
		Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
	}
}

// Device implements api.DeviceAuthorizer: the device a public key is
// enrolled for, when it is and its credential is live.
func (a *authorizer) Device(pubKey string) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	e, ok := a.creds[pubKey]
	if !a.ready || !ok || e.disabled {
		return "", false
	}

	return e.deviceID, true
}

// originsFor lists the origins a device may pull from, sorted so the list
// doubles as a fingerprint of the grant. Lock must be held.
func (a *authorizer) originsFor(deviceID string) []string {
	set := map[string]bool{a.rootID: true}
	for o := range a.origins[deviceID] {
		set[o] = true
	}
	delete(set, deviceID)

	origins := make([]string, 0, len(set))
	for o := range set {
		origins = append(origins, o)
	}
	sort.Strings(origins)

	return origins
}

// verifyNonce checks that the signature was made over the connection nonce
// by the holder of the public key's seed. The signature is encoded the way
// nats.go sends it.
func verifyNonce(pubKey, sig string, nonce []byte) bool {
	if sig == "" || len(nonce) == 0 {
		return false
	}

	raw, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return false
		}
	}

	kp, err := nkeys.FromPublicKey(pubKey)
	if err != nil {
		return false
	}

	return kp.Verify(nonce, raw) == nil
}

func isLoopback(addr net.Addr) bool {
	if addr == nil {
		return false
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		host = addr.String()
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// devicePermissions is everything a device with root X needs to sync with
// this instance (root R), and nothing else:
//
//   - find the upstream root and check whether it is adopted
//   - announce itself under the root
//   - push its origin stream into the replica stream inst_X_X
//   - discover and pull the streams other origins write into its
//     boundary, which are enumerated because a stream name is one subject
//     token and cannot be matched by prefix
//   - receive replies
//
// A device never needs p.>, up.>, auth.*, admin.*, or any other instance's
// streams.
func devicePermissions(deviceID, rootID string, origins []string) *server.Permissions {
	X := deviceID
	own := fmt.Sprintf("inst_%v_%v", X, X)

	pub := []string{
		"nodes.root.all",
		"nodes.all." + X,
		fmt.Sprintf("ep.%v.%v", X, rootID),
		fmt.Sprintf("inst.%v.%v.>", X, X),
		"$JS.API.STREAM.INFO." + own,
		"$JS.API.STREAM.CREATE." + own,
		"$JS.API.STREAM.NAMES",
	}

	for _, o := range origins {
		if o == X {
			continue
		}
		s := fmt.Sprintf("inst_%v_%v", X, o)
		pub = append(pub,
			"$JS.API.STREAM.INFO."+s,
			"$JS.API.CONSUMER.CREATE."+s+".>",
			"$JS.API.CONSUMER.INFO."+s+".*",
			"$JS.API.CONSUMER.MSG.NEXT."+s+".*",
			"$JS.ACK."+s+".>",
		)
	}

	return &server.Permissions{
		Publish:   &server.SubjectPermission{Allow: pub},
		Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
	}
}

// start loads the index from the store and begins maintaining it. It is
// called once the store is serving requests; device connections that
// arrive before then are refused and retry.
func (a *authorizer) start(nc *nats.Conn, ns *server.Server) error {
	a.nc = nc
	a.ns = ns

	root, err := client.GetRootNode(nc)
	if err != nil {
		return fmt.Errorf("error getting root node: %w", err)
	}

	a.mu.Lock()
	a.rootID = root.ID
	a.mu.Unlock()

	// subscribe before loading so nothing is missed in between
	a.sub, err = nc.Subscribe("up.root.>", a.handleUp)
	if err != nil {
		return fmt.Errorf("error subscribing to tree changes: %w", err)
	}

	a.adv, err = nc.Subscribe("$JS.EVENT.ADVISORY.STREAM.*.*", func(_ *nats.Msg) {
		a.enqueue(a.loadOrigins)
	})
	if err != nil {
		return fmt.Errorf("error subscribing to stream advisories: %w", err)
	}

	a.loadOrigins()

	creds, err := client.GetNodes(nc, "all", "all", data.NodeTypeDeviceCred, true)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return fmt.Errorf("error loading credentials: %w", err)
	}

	seen := map[string]bool{}
	for _, c := range creds {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		a.refreshCred(c.ID)
	}

	tokens, err := client.GetNodes(nc, "all", "all", data.NodeTypeEnrollToken, true)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return fmt.Errorf("error loading enrollment tokens: %w", err)
	}

	seen = map[string]bool{}
	for _, t := range tokens {
		if seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		a.refreshEnroll(t.ID)
	}

	a.mu.Lock()
	a.ready = true
	n := len(a.creds)
	a.mu.Unlock()

	log.Printf("NATS auth: %v device credential(s) loaded, device auth %v",
		n, a.deviceAuth)

	go a.run()

	return nil
}

// stopIndex stops maintaining the index. The server refuses device
// connections from then on, which only matters during shutdown.
func (a *authorizer) stopIndex() {
	if a.sub != nil {
		_ = a.sub.Unsubscribe()
	}
	if a.adv != nil {
		_ = a.adv.Unsubscribe()
	}

	a.mu.Lock()
	a.ready = false
	started := a.nc != nil
	a.mu.Unlock()

	if started {
		close(a.stop)
		<-a.done
	}
}

func (a *authorizer) enqueue(f func()) {
	select {
	case a.work <- f:
	default:
		log.Println("NATS auth: index maintenance queue full, dropping update")
	}
}

// handleUp watches every point in the tree for the few that concern
// credentials: a point on a credential node, a new deviceCred node, and an
// edge change on a device that has credentials.
func (a *authorizer) handleUp(msg *nats.Msg) {
	tok := strings.Split(msg.Subject, ".")

	// up.root.<node>.<type>.<key> is a node point;
	// up.root.<node>.<parent>.<type>.<key> is an edge point
	if len(tok) != 5 && len(tok) != 6 {
		return
	}

	nodeID := tok[2]

	a.mu.RLock()
	_, isCred := a.credIDs[nodeID]
	_, isDevice := a.devices[nodeID]
	_, isEnroll := a.enrollIDs[nodeID]
	a.mu.RUnlock()

	switch {
	case isCred:
		a.enqueue(func() { a.refreshCred(nodeID) })
	case isEnroll:
		a.enqueue(func() { a.refreshEnroll(nodeID) })
	case isDevice:
		a.enqueue(func() { a.refreshDevice(nodeID) })
	case len(tok) == 6 && tok[4] == data.PointTypeNodeType:
		pts, err := data.DecodePoints(msg.Data)
		if err != nil {
			return
		}
		for _, p := range pts {
			switch p.Txt() {
			case data.NodeTypeDeviceCred:
				a.enqueue(func() { a.refreshCred(nodeID) })
				return
			case data.NodeTypeEnrollToken:
				a.enqueue(func() { a.refreshEnroll(nodeID) })
				return
			}
		}
	}
}

// refreshEnroll rereads one enrollment token from the store.
func (a *authorizer) refreshEnroll(nodeID string) {
	edges, err := client.GetNodes(a.nc, "all", nodeID, "", true)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		log.Printf("NATS auth: error reading enrollment token %v: %v", nodeID, err)
		return
	}

	var t client.EnrollToken
	found, live := false, false
	for _, e := range edges {
		if e.Type != data.NodeTypeEnrollToken {
			continue
		}
		found = true
		if err := data.Decode(data.NodeEdgeChildren{NodeEdge: e}, &t); err != nil {
			log.Printf("NATS auth: error decoding enrollment token %v: %v", nodeID, err)
			return
		}
		if ts, _ := e.IsTombstone(); !ts {
			live = true
		}
	}

	if !found {
		return
	}

	a.mu.Lock()
	if old, ok := a.enrollIDs[nodeID]; ok && old != t.TokenHash {
		delete(a.enroll, old)
	}
	a.enrollIDs[nodeID] = t.TokenHash
	if t.TokenHash != "" {
		a.enroll[t.TokenHash] = &enrollEntry{
			nodeID:      nodeID,
			disabled:    t.Disabled || !live,
			autoApprove: t.AutoApprove,
			expires:     int64(t.Expires),
		}
	}
	a.mu.Unlock()

	a.enforce()
}

// run serializes index maintenance, status updates, and enforcement.
func (a *authorizer) run() {
	defer close(a.done)

	ticker := time.NewTicker(authReconcilePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-a.stop:
			return
		case f := <-a.work:
			f()
		case pub := <-a.connects:
			a.noteConnect(pub)
		case <-ticker.C:
			a.reconcile()
		}
	}
}

// refreshCred rereads one credential from the store and updates the index.
// Every change to a credential comes through here, so the rules for what
// authorizes are in one place: a credential authorizes the one live device
// it sits under, as long as that device is itself live in the tree.
func (a *authorizer) refreshCred(credID string) {
	edges, err := client.GetNodes(a.nc, "all", credID, "", true)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		log.Printf("NATS auth: error reading credential %v: %v", credID, err)
		return
	}

	var pubKey string
	var disabled, connected, found bool
	var parents []string

	for _, e := range edges {
		if e.Type != data.NodeTypeDeviceCred {
			continue
		}
		found = true
		pubKey, _ = e.Points.Text(data.PointTypePubKey, "")
		disabled, _ = e.Points.ValueBool(data.PointTypeDisabled, "")
		connected, _ = e.Points.ValueBool(data.PointTypeConnected, "")
		// a credential a device enrolled itself with authorizes nothing
		// until an operator approves it
		if pending, _ := e.Points.ValueBool(data.PointTypePending, ""); pending {
			disabled = true
		}
		if ts, _ := e.IsTombstone(); !ts {
			parents = append(parents, e.Parent)
		}
	}

	if !found {
		return
	}

	var deviceID string
	switch len(parents) {
	case 1:
		deviceID = parents[0]
		if !a.deviceLive(credID, deviceID) {
			disabled = true
		}
	case 0:
		disabled = true
	default:
		log.Printf("NATS auth: credential %v has %v parents, authorizing nothing",
			credID, len(parents))
		disabled = true
	}

	a.mu.Lock()
	if old, ok := a.credIDs[credID]; ok && old != "" && old != pubKey {
		delete(a.creds, old)
	}
	a.credIDs[credID] = pubKey

	for d, ids := range a.devices {
		delete(ids, credID)
		if len(ids) == 0 {
			delete(a.devices, d)
		}
	}
	if deviceID != "" {
		if a.devices[deviceID] == nil {
			a.devices[deviceID] = make(map[string]bool)
		}
		a.devices[deviceID][credID] = true
	}

	if pubKey != "" {
		e := a.creds[pubKey]
		if e == nil {
			// the stored status is the starting point; from here on
			// the authorizer is the authority on it
			e = &credEntry{connected: connected}
			a.creds[pubKey] = e
		}
		e.credID = credID
		e.deviceID = deviceID
		e.disabled = disabled
	}
	a.mu.Unlock()

	a.enforce()
}

// deviceLive reports whether the node a credential sits under is a live
// device node other than this instance's own root. A detached device must
// not keep writing, and a credential under anything else would grant that
// node's subjects: under the root, that is this instance's own origin
// stream.
func (a *authorizer) deviceLive(credID, deviceID string) bool {
	if deviceID == a.root() {
		log.Printf("NATS auth: credential %v is under this instance's root node, authorizing nothing",
			credID)
		return false
	}

	edges, err := client.GetNodes(a.nc, "all", deviceID, "", false)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		log.Printf("NATS auth: error reading device %v: %v", deviceID, err)
		return false
	}

	if len(edges) == 0 {
		return false
	}

	if edges[0].Type != data.NodeTypeDevice {
		log.Printf("NATS auth: credential %v is under a %v node, not a device, authorizing nothing",
			credID, edges[0].Type)
		return false
	}

	return true
}

// refreshDevice rereads every credential under a device.
func (a *authorizer) refreshDevice(deviceID string) {
	a.mu.RLock()
	var ids []string
	for id := range a.devices[deviceID] {
		ids = append(ids, id)
	}
	a.mu.RUnlock()

	for _, id := range ids {
		a.refreshCred(id)
	}
}

// loadOrigins lists the boundary-origin streams on this instance and
// records, per boundary, which origins write into it. A device whose grant
// no longer matches is reconnected by enforce so it picks the new streams
// up.
func (a *authorizer) loadOrigins() {
	js, err := jetstream.New(a.nc)
	if err != nil {
		log.Println("NATS auth: error creating JetStream context:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	origins := make(map[string]map[string]bool)
	lister := js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))
	for si := range lister.Info() {
		b, o, ok := client.StreamBoundaryOrigin(si.Config)
		if !ok || b == o {
			continue
		}
		if origins[b] == nil {
			origins[b] = make(map[string]bool)
		}
		origins[b][o] = true
	}
	if err := lister.Err(); err != nil {
		log.Println("NATS auth: error listing streams:", err)
		return
	}

	a.mu.Lock()
	a.origins = origins
	a.mu.Unlock()

	a.enforce()
}

// noteConnect records a successful device connection on its credential
// node.
func (a *authorizer) noteConnect(pubKey string) {
	a.mu.Lock()
	e, ok := a.creds[pubKey]
	var credID string
	if ok {
		credID = e.credID
		e.connected = true
	}
	a.mu.Unlock()

	if !ok {
		return
	}

	pts := data.Points{
		data.NewPointFloat(data.PointTypeLastConnect, "", float64(time.Now().Unix())),
		data.NewPointFloat(data.PointTypeConnected, "", 1),
	}
	if err := client.SendNodePoints(a.nc, credID, pts, false); err != nil {
		log.Printf("NATS auth: error recording connection on %v: %v", credID, err)
	}
}

// liveConns lists the device connections the server currently holds.
func (a *authorizer) liveConns() map[uint64]string {
	live := make(map[uint64]string)
	if a.ns == nil {
		return live
	}

	for offset := 0; ; {
		cz, err := a.ns.Connz(&server.ConnzOptions{Username: true, Offset: offset, Limit: 1024})
		if err != nil {
			log.Println("NATS auth: error listing connections:", err)
			return live
		}
		for _, c := range cz.Conns {
			if nkeys.IsValidPublicUserKey(c.AuthorizedUser) ||
				strings.HasPrefix(c.AuthorizedUser, enrollUserPrefix) {
				live[c.Cid] = c.AuthorizedUser
			}
		}
		offset += len(cz.Conns)
		if offset >= cz.Total || len(cz.Conns) == 0 {
			break
		}
	}

	return live
}

// enforce closes every device connection whose credential no longer
// authorizes what it was granted: the credential was disabled, deleted, or
// re-keyed, its device was detached, or its boundary gained a stream it
// should now pull.
func (a *authorizer) enforce() {
	live := a.liveConns()

	a.mu.Lock()
	var closeIDs []uint64
	for cid, pub := range live {
		if id, isEnroll := strings.CutPrefix(pub, enrollUserPrefix); isEnroll {
			e, ok := a.enroll[a.enrollIDs[id]]
			if !ok || !e.live() {
				closeIDs = append(closeIDs, cid)
			}
			continue
		}
		e, ok := a.creds[pub]
		if !ok || e.disabled {
			closeIDs = append(closeIDs, cid)
			continue
		}
		if ac, ok := a.conns[cid]; ok &&
			ac.grant != strings.Join(a.originsFor(e.deviceID), ",") {
			closeIDs = append(closeIDs, cid)
		}
	}
	for cid := range a.conns {
		if _, ok := live[cid]; !ok {
			delete(a.conns, cid)
		}
	}
	a.mu.Unlock()

	for _, cid := range closeIDs {
		log.Printf("NATS auth: closing connection %v, credential %v no longer authorizes it",
			cid, live[cid])
		if err := a.ns.DisconnectClientByID(cid); err != nil {
			log.Printf("NATS auth: error closing connection %v: %v", cid, err)
		}
	}
}

// reconcile enforces the index against live connections and brings the
// connected point on each credential up to date.
func (a *authorizer) reconcile() {
	a.enforce()

	live := a.liveConns()
	connected := make(map[string]bool)
	for _, pub := range live {
		connected[pub] = true
	}

	var disconnected []string
	a.mu.Lock()
	for pub, e := range a.creds {
		if e.connected && !connected[pub] {
			e.connected = false
			disconnected = append(disconnected, e.credID)
		}
	}
	a.mu.Unlock()

	for _, credID := range disconnected {
		p := data.NewPointFloat(data.PointTypeConnected, "", 0)
		if err := client.SendNodePoint(a.nc, credID, p, false); err != nil {
			log.Printf("NATS auth: error recording disconnect on %v: %v", credID, err)
		}
	}
}
