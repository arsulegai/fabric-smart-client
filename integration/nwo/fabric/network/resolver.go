/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package network

import (
	"fmt"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/registry"
)

type ResolverIdentity struct {
	ID      string
	MSPType string
	MSPID   string
	Path    string
}

type Resolver struct {
	Name      string
	Domain    string
	Identity  ResolverIdentity
	Addresses map[registry.PortName]string
	Port      int
	Aliases   []string
}

// ResolverMapPath returns the path to the generated resolver map configuration
// file.
func (n *Network) ResolverMapPath(p *topology.Peer) string {
	return filepath.Join(n.PeerDir(p), "resolver.json")
}

func (n *Network) GenerateResolverMap() {
	n.Resolvers = []*Resolver{}
	for _, peer := range n.Peers {
		org := n.Organization(peer.Organization)

		var addresses map[registry.PortName]string
		var path string
		if peer.Type == topology.ViewPeer {
			addresses = map[registry.PortName]string{
				ViewPort:   fmt.Sprintf("127.0.0.1:%d", n.Registry.PortsByPeerID[peer.Name][ListenPort]),
				ListenPort: fmt.Sprintf("127.0.0.1:%d", n.Registry.PortsByPeerID[peer.Name][ListenPort]),
				P2PPort:    fmt.Sprintf("127.0.0.1:%d", n.Registry.PortsByPeerID[peer.Name][P2PPort]),
			}
			if n.topology.NodeOUs {
				switch peer.Role {
				case "":
					path = n.PeerUserLocalMSPIdentityCert(peer, peer.Name)
				case "client":
					path = n.PeerUserLocalMSPIdentityCert(peer, peer.Name)
				default:
					path = n.PeerLocalMSPIdentityCert(peer)
				}
			} else {
				path = n.PeerLocalMSPIdentityCert(peer)
			}
		} else {
			addresses = map[registry.PortName]string{
				ViewPort:   fmt.Sprintf("127.0.0.1:%d", n.Registry.PortsByPeerID[peer.ID()][ListenPort]),
				ListenPort: fmt.Sprintf("127.0.0.1:%d", n.Registry.PortsByPeerID[peer.ID()][ListenPort]),
				P2PPort:    fmt.Sprintf("127.0.0.1:%d", n.Registry.PortsByPeerID[peer.ID()][P2PPort]),
			}
			path = n.PeerLocalMSPIdentityCert(peer)
		}

		var aliases []string
		for _, eid := range peer.ExtraIdentities {
			if len(eid.EnrollmentID) != 0 {
				aliases = append(aliases, eid.EnrollmentID)
			}
		}

		n.Resolvers = append(n.Resolvers, &Resolver{
			Name: peer.Name,
			Identity: ResolverIdentity{
				ID:      peer.Name,
				MSPType: "bccsp",
				MSPID:   org.MSPID,
				Path:    path,
			},
			Domain:    org.Domain,
			Addresses: addresses,
			Aliases:   aliases,
		})
	}
}

func (n *Network) ViewNodeLocalCertPath(peer *topology.Peer) string {
	if n.topology.NodeOUs {
		switch peer.Role {
		case "":
			return n.PeerUserLocalMSPIdentityCert(peer, peer.Name)
		case "client":
			return n.PeerUserLocalMSPIdentityCert(peer, peer.Name)
		default:
			return n.PeerLocalMSPIdentityCert(peer)
		}
	} else {
		return n.PeerLocalMSPIdentityCert(peer)
	}
}

func (n *Network) ViewNodeLocalPrivateKeyPath(peer *topology.Peer) string {
	if n.topology.NodeOUs {
		switch peer.Role {
		case "":
			return n.PeerUserKey(peer, peer.Name)
		case "client":
			return n.PeerUserKey(peer, peer.Name)
		default:
			return n.PeerKey(peer)
		}
	} else {
		return n.PeerKey(peer)
	}
}
