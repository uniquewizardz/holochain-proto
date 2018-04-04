// Copyright (C) 2013-2018, The MetaCurrency Project (Eric Harris-Braun, Arthur Brock, et. al.)
// Use of this source code is governed by GPLv3 found in the LICENSE file
//---------------------------------------------------------------------------------------
// implements managing and storing the world model for holochain nodes

package holochain

import (
	"errors"
	. "github.com/holochain/holochain-proto/hash"
	ic "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"sync"
)

// NodeRecord stores the necessary information about other nodes in the world model
type NodeRecord struct {
	PeerInfo  pstore.PeerInfo
	PubKey    ic.PubKey
	IsHolding map[Hash]bool
}

// World holds the data of a nodes' world model
type World struct {
	me          peer.ID
	nodes       map[peer.ID]*NodeRecord
	responsible map[Hash][]peer.ID
	ht          HashTable

	lk sync.RWMutex
}

var ErrNodeNotFound = errors.New("node not found")

// NewWorld creates and empty world model
func NewWorld(me peer.ID, ht HashTable) *World {
	world := World{me: me}
	world.nodes = make(map[peer.ID]*NodeRecord)
	world.responsible = make(map[Hash][]peer.ID)
	world.ht = ht
	return &world
}

// GetNodeRecord returns the peer's node record
// NOTE: do not modify the contents of the returned record! not thread safe
func (world *World) GetNodeRecord(ID peer.ID) (record *NodeRecord) {
	world.lk.RLock()
	defer world.lk.RUnlock()
	record = world.nodes[ID]
	return
}

// SetNodeHolding marks a node as holding a particular hash
func (world *World) SetNodeHolding(ID peer.ID, hash Hash) (err error) {
	//fmt.Printf("Setting Holding for %v of holding %v nodes:%v\n", ID, hash, world.nodes)
	world.lk.Lock()
	defer world.lk.Unlock()
	record := world.nodes[ID]
	if record == nil {
		err = ErrNodeNotFound
		return
	}
	record.IsHolding[hash] = true
	return
}

// IsHolding returns whether a node is holding a particular hash
func (world *World) IsHolding(ID peer.ID, hash Hash) (holding bool, err error) {
	world.lk.RLock()
	defer world.lk.RUnlock()
	//fmt.Printf("Looking to see if %v is holding %v\n", ID, hash)
	//fmt.Printf("NODES:%v\n", world.nodes)
	record := world.nodes[ID]
	if record == nil {
		err = ErrNodeNotFound
		return
	}
	holding = record.IsHolding[hash]
	return
}

// AllNodes returns a list of all the nodes in the world model.
func (world *World) AllNodes() (nodes []peer.ID, err error) {
	world.lk.RLock()
	defer world.lk.RUnlock()
	nodes, err = world.allNodes()
	return
}

func (world *World) allNodes() (nodes []peer.ID, err error) {
	nodes = make([]peer.ID, len(world.nodes))

	i := 0
	for k := range world.nodes {
		nodes[i] = k
		i++
	}
	return
}

// AddNode adds a node to the world model
func (world *World) AddNode(pi pstore.PeerInfo, pubKey ic.PubKey) (err error) {
	world.lk.Lock()
	defer world.lk.Unlock()
	rec := NodeRecord{PeerInfo: pi, PubKey: pubKey, IsHolding: make(map[Hash]bool)}
	world.nodes[pi.ID] = &rec
	return
}

// NodesByHash returns a sorted list of peers, including "me" by distance from a hash
func (world *World) nodesByHash(hash Hash) (nodes []peer.ID, err error) {
	nodes, err = world.allNodes()
	if err != nil {
		return
	}
	nodes = append(nodes, world.me)
	nodes = SortClosestPeers(nodes, hash)
	return
}

/*
func (world *World) NodeRecordsByHash(hash Hash) (records []*NodeRecord, err error) {

	records = make([]*NodeRecord, len(nodes))
	i := 0
	for _, id := range nodes {
		records[i] = world.nodes[id]
		i++
	}
	return
}*/

// UpdateResponsible calculates the list of nodes believed to be responsible for a given hash
// note that if redundancy is 0 the assumption is that all nodes are responsible
func (world *World) UpdateResponsible(hash Hash, redundancy int) (responsible bool, err error) {
	world.lk.Lock()
	defer world.lk.Unlock()
	var nodes []peer.ID
	if redundancy == 0 {
		world.responsible[hash] = nil
		responsible = true
	} else if redundancy > 1 {
		nodes, err = world.nodesByHash(hash)
		if err != nil {
			return
		}
		// TODO add in resilince calculations with uptime
		i := 0
		for i = 0; i < redundancy; i++ {
			if nodes[i] == world.me {
				responsible = true
				break
			}
		}
		// if me is included in the range of nodes that are close to the has
		// add this hash (and other nodes) to the responsible map
		// otherwise delete the item from the responsible map
		if responsible {
			// remove myself from the nodes list so I can add set the
			// responsible nodes
			nodes = append(nodes[:i], nodes[i+1:redundancy]...)
			world.responsible[hash] = nodes
		} else {
			delete(world.responsible, hash)
		}
	} else {
		panic("not implemented")
	}
	return
}

// Responsible returns a list of all the entries I'm responsible for holding
func (world *World) Responsible() (entries []Hash, err error) {
	world.lk.RLock()
	defer world.lk.RUnlock()
	entries = make([]Hash, len(world.responsible))

	i := 0
	for k := range world.responsible {
		entries[i] = k
		i++
	}
	return
}

// Overlap returns a list of all the nodes that overlap for a given hash
func (h *Holochain) Overlap(hash Hash) (overlap []peer.ID, err error) {
	h.world.lk.RLock()
	defer h.world.lk.RUnlock()
	if h.nucleus.dna.DHTConfig.RedundancyFactor == 0 {
		overlap, err = h.world.allNodes()
	} else {
		overlap = h.world.responsible[hash]
	}
	return
}

func HoldingTask(h *Holochain) {
	/*	h.dht.Iterate(func(hash Hash) bool {
		//TODO forget the hashes we are no longer responsible for
		//TODO this really shouldn't be called in the holding task
		//     but instead should be called with the Node list or hash list changes.
		h.world.UpdateResponsible(hash, h.RedundancyFactor())

		// TODO make this more efficient by collecting up a list of updates
		// per node rather than making the hold request over and over
		overlap, err := h.Overlap(hash)
		if err != nil {
			for _, nodeID := range overlap {

			}
		}
		return false
	})*/
}
