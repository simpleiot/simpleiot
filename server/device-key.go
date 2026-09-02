package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// deviceKeyFile is where an instance keeps its device key, relative to
// SIOT_DATA. It is generated on first start, so an image must not ship one.
const deviceKeyFile = "device.nkey"

// deviceKey is this instance's identity when it connects to an upstream: an
// NKey user seed kept in a file, never in the store, since every point on
// this instance replicates upstream. The public key is put on the sync
// nodes so it can be read off the device and enrolled.
type deviceKey struct {
	path string

	mu     sync.Mutex
	seed   string
	pubKey string

	nc   *nats.Conn
	subs []*nats.Subscription
}

func newDeviceKey(dataDir string) *deviceKey {
	return &deviceKey{path: filepath.Join(dataDir, deviceKeyFile)}
}

// load reads the key file, generating one the first time.
func (k *deviceKey) load() error {
	b, err := os.ReadFile(k.path)
	if errors.Is(err, os.ErrNotExist) {
		seed, _, err := client.GenerateDeviceKey()
		if err != nil {
			return err
		}
		if err := k.write(seed); err != nil {
			return err
		}
		b = []byte(seed)
		log.Println("Device key generated:", k.path)
	} else if err != nil {
		return fmt.Errorf("error reading device key: %w", err)
	}

	seed := strings.TrimSpace(string(b))
	pubKey, err := client.ParseDeviceKey(seed)
	if err != nil {
		return fmt.Errorf("error in %v: %w", k.path, err)
	}

	k.mu.Lock()
	k.seed = seed
	k.pubKey = pubKey
	k.mu.Unlock()

	log.Println("Device key:", pubKey)

	return nil
}

func (k *deviceKey) write(seed string) error {
	if err := os.MkdirAll(filepath.Dir(k.path), 0o755); err != nil {
		return fmt.Errorf("error creating data dir: %w", err)
	}
	if err := os.WriteFile(k.path, []byte(seed+"\n"), 0o600); err != nil {
		return fmt.Errorf("error writing device key: %w", err)
	}
	return nil
}

// start answers key requests and makes sure the sync nodes show the key.
func (k *deviceKey) start(nc *nats.Conn) error {
	k.nc = nc

	sub, err := nc.Subscribe(client.SubjectDeviceKey, k.handleGet)
	if err != nil {
		return err
	}
	k.subs = append(k.subs, sub)

	k.publish()

	return nil
}

func (k *deviceKey) stop() {
	for _, s := range k.subs {
		_ = s.Unsubscribe()
	}
}

func (k *deviceKey) reply(msg *nats.Msg, r client.DeviceKey) {
	b, _ := json.Marshal(r)
	if err := k.nc.Publish(msg.Reply, b); err != nil {
		log.Println("Error replying to device key request:", err)
	}
}

func (k *deviceKey) handleGet(msg *nats.Msg) {
	k.mu.Lock()
	r := client.DeviceKey{Seed: k.seed, PubKey: k.pubKey}
	k.mu.Unlock()

	k.reply(msg, r)
}

// publish puts the public key on every sync node under the root that does
// not already show it.
func (k *deviceKey) publish() {
	k.mu.Lock()
	pubKey := k.pubKey
	k.mu.Unlock()

	root, err := client.GetRootNode(k.nc)
	if err != nil {
		log.Println("Device key: error getting root node:", err)
		return
	}

	syncs, err := client.GetNodesType[client.Sync](k.nc, root.ID, "all")
	if err != nil {
		log.Println("Device key: error getting sync nodes:", err)
		return
	}

	for _, s := range syncs {
		if s.PubKey == pubKey {
			continue
		}
		p := data.NewPointString(data.PointTypePubKey, "", pubKey)
		// an origin so the sync client is told about a point it did
		// not write itself
		p.Origin = "server"
		if err := client.SendNodePoint(k.nc, s.ID, p, true); err != nil {
			log.Println("Device key: error updating sync node:", err)
		}
	}
}
