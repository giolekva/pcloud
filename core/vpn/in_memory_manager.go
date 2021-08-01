package vpn

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/giolekva/pcloud/core/vpn/types"
)

func errorDeviceNotFound(pubKey types.PublicKey) error {
	return fmt.Errorf("Device not found: %s", pubKey)
}

func errorGroupNotFound(id types.GroupID) error {
	return fmt.Errorf("Group not found: %s", id)
}

type InMemoryManager struct {
	lock           sync.Mutex
	devices        []*types.DeviceInfo
	keyToDevices   map[types.PublicKey]*types.DeviceInfo
	currGroupID    int64
	groups         map[types.GroupID]*types.Group
	deviceToGroups map[types.PublicKey][]*types.Group
	callbacks      map[types.PublicKey][]NetworkMapChangeCallback
	ipm            IPManager
}

func NewInMemoryManager(ipm IPManager) Manager {
	return &InMemoryManager{
		devices:        make([]*types.DeviceInfo, 0),
		keyToDevices:   make(map[types.PublicKey]*types.DeviceInfo),
		callbacks:      make(map[types.PublicKey][]NetworkMapChangeCallback),
		currGroupID:    0,
		groups:         make(map[types.GroupID]*types.Group),
		deviceToGroups: make(map[types.PublicKey][]*types.Group),
		ipm:            ipm,
	}
}

func (m *InMemoryManager) RegisterDevice(d types.DeviceInfo) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.keyToDevices[d.PublicKey]; ok {
		return fmt.Errorf("Device with given public key is already registered: %s", d.PublicKey)
	}
	if _, err := m.ipm.New(d.PublicKey); err != nil {
		return err
	}
	m.keyToDevices[d.PublicKey] = &d
	m.devices = append(m.devices, &d)
	m.callbacks[d.PublicKey] = make([]NetworkMapChangeCallback, 0)
	m.deviceToGroups[d.PublicKey] = make([]*types.Group, 0)
	return nil
}

func (m *InMemoryManager) RemoveDevice(pubKey types.PublicKey) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.keyToDevices[pubKey]; !ok {
		return errorDeviceNotFound(pubKey)
	}
	for _, g := range m.deviceToGroups[pubKey] {
		m.removeDeviceFromGroupNoLock(pubKey, g.ID)
	}
	delete(m.deviceToGroups, pubKey)
	delete(m.callbacks, pubKey)
	found := false
	for i, peer := range m.devices {
		if peer.PublicKey == pubKey {
			m.devices[i] = m.devices[len(m.devices)-1]
			m.devices = m.devices[:len(m.devices)-1]
			found = true
			break
		}
	}
	if !found {
		panic("MUST not happen, device not found")
	}
	delete(m.keyToDevices, pubKey) // TODO(giolekva): maybe mark as deleted?
	return nil
}

func (m *InMemoryManager) CreateGroup(name string) (types.GroupID, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	id := types.GroupID(strconv.FormatInt(m.currGroupID, 10))
	m.groups[id] = &types.Group{
		ID:    id,
		Name:  name,
		Peers: make([]*types.DeviceInfo, 0),
	}
	m.currGroupID++
	return id, nil
}

func (m *InMemoryManager) DeleteGroup(id types.GroupID) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	g, ok := m.groups[id]
	if !ok {
		return errorGroupNotFound(id)
	}
	// TODO(giolekva): optimize, current implementation calls callbacks group size squared times.
	for _, peer := range g.Peers {
		if _, err := m.removeDeviceFromGroupNoLock(peer.PublicKey, id); err != nil {
			return err
		}
	}
	return nil
}

func (m *InMemoryManager) AddDeviceToGroup(pubKey types.PublicKey, id types.GroupID) (*types.NetworkMap, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	d, ok := m.keyToDevices[pubKey]
	if !ok {
		return nil, errorDeviceNotFound(pubKey)
	}
	g, ok := m.groups[id]
	if !ok {
		return nil, errorGroupNotFound(id)
	}
	groups, ok := m.deviceToGroups[pubKey]
	if !ok {
		groups = make([]*types.Group, 1)
	}
	// TODO(giolekva): Check if device is already in the group and return error if so.
	g.Peers = append(g.Peers, d)
	groups = append(groups, g)
	m.deviceToGroups[pubKey] = groups
	ret, err := m.genNetworkMap(d)
	m.notifyPeers(d, g)
	return ret, err
}

func (m *InMemoryManager) RemoveDeviceFromGroup(pubKey types.PublicKey, id types.GroupID) (*types.NetworkMap, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.removeDeviceFromGroupNoLock(pubKey, id)
}

func (m *InMemoryManager) removeDeviceFromGroupNoLock(pubKey types.PublicKey, id types.GroupID) (*types.NetworkMap, error) {
	d, ok := m.keyToDevices[pubKey]
	if !ok {
		return nil, errorDeviceNotFound(pubKey)
	}
	g, ok := m.groups[id]
	if !ok {
		return nil, errorGroupNotFound(id)
	}
	groups := m.deviceToGroups[pubKey]
	found := false
	for i, group := range groups {
		if id == group.ID {
			groups[i] = groups[len(groups)-1]
			groups = groups[:len(groups)-1]
			m.deviceToGroups[pubKey] = groups
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Device %s is not part of the group %s", pubKey, id)
	}
	found = false
	for i, peer := range g.Peers {
		if pubKey == peer.PublicKey {
			g.Peers[i] = g.Peers[len(g.Peers)-1]
			g.Peers = g.Peers[:len(g.Peers)-1]
			found = true
		}
	}
	if !found {
		panic("Should not reach")
	}
	m.notifyPeers(d, g)
	return m.genNetworkMap(d)
}

func (m *InMemoryManager) GetNetworkMap(pubKey types.PublicKey) (*types.NetworkMap, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if d, ok := m.keyToDevices[pubKey]; ok {
		return m.genNetworkMap(d)
	}
	return nil, errorDeviceNotFound(pubKey)
}

func (m *InMemoryManager) AddNetworkMapChangeCallback(pubKey types.PublicKey, cb NetworkMapChangeCallback) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.keyToDevices[pubKey]; ok {
		m.callbacks[pubKey] = append(m.callbacks[pubKey], cb)
	}
	return errorDeviceNotFound(pubKey)
}

func (m *InMemoryManager) notifyPeers(d *types.DeviceInfo, g *types.Group) {
	// TODO(giolekva): maybe run this in a goroutine?
	for _, peer := range g.Peers {
		if peer.PublicKey != d.PublicKey {
			netMap, err := m.genNetworkMap(peer)
			if err != nil {
				panic(err) // TODO(giolekva): handle properly
			}
			for _, cb := range m.callbacks[peer.PublicKey] {
				cb(netMap)
			}
		}
	}
}

func (m *InMemoryManager) genNetworkMap(d *types.DeviceInfo) (*types.NetworkMap, error) {
	vpnIP, err := m.ipm.Get(d.PublicKey)
	// NOTE(giolekva): Should not happen as devices must have been already registered and assigned IP address.
	// Maybe should return error anyways instead of panic?
	if err != nil {
		return nil, err
	}
	ret := types.NetworkMap{
		Self: types.Node{
			PublicKey: d.PublicKey,
			DiscoKey:  d.DiscoKey,
			IPPort:    d.IPPort,
			VPNIP:     vpnIP,
		},
	}
	for _, group := range m.deviceToGroups[d.PublicKey] {
		for _, peer := range group.Peers {
			if d.PublicKey == peer.PublicKey {
				continue
			}
			vpnIP, err := m.ipm.Get(peer.PublicKey)
			if err != nil {
				panic(err)
			}
			ret.Peers = append(ret.Peers, types.Node{
				PublicKey: peer.PublicKey,
				DiscoKey:  peer.DiscoKey,
				IPPort:    peer.IPPort,
				VPNIP:     vpnIP,
			})
		}
	}
	return &ret, nil
}
