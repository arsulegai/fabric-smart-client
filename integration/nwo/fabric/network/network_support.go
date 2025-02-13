/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package network

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/hyperledger/fabric/integration/runner"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
	"gopkg.in/yaml.v2"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/commands"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/fabricconfig"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/opts"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/registry"
)

func (n *Network) LogSpec() string {
	if len(n.Logging.Spec) == 0 {
		return "info:fscnode=debug"
	}
	return n.Logging.Spec
}

func (n *Network) LogFormat() string {
	if len(n.Logging.Format) == 0 {
		return "'%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'"
	}
	return n.Logging.Format
}

// AddOrg adds an organization to a network.
func (n *Network) AddOrg(o *topology.Organization, peers ...*topology.Peer) {
	for _, p := range peers {
		ports := registry.Ports{}
		for _, portName := range PeerPortNames() {
			ports[portName] = n.Registry.ReservePort()
		}
		n.Registry.PortsByPeerID[p.ID()] = ports
		n.Peers = append(n.Peers, p)
	}

	n.Organizations = append(n.Organizations, o)
	n.Consortiums[0].Organizations = append(n.Consortiums[0].Organizations, o.Name)
}

// ConfigTxPath returns the path to the generated configtxgen configuration
// file.
func (n *Network) ConfigTxConfigPath() string {
	return filepath.Join(n.Registry.RootDir, "configtx.yaml")
}

// CryptoPath returns the path to the directory where cryptogen will place its
// generated artifacts.
func (n *Network) CryptoPath() string {
	return filepath.Join(n.Registry.RootDir, "crypto")
}

// CryptoConfigPath returns the path to the generated cryptogen configuration
// file.
func (n *Network) CryptoConfigPath() string {
	return filepath.Join(n.Registry.RootDir, "crypto-config.yaml")
}

// OutputBlockPath returns the path to the genesis block for the named system
// channel.
func (n *Network) OutputBlockPath(channelName string) string {
	return filepath.Join(n.Registry.RootDir, fmt.Sprintf("%s_block.pb", channelName))
}

// CreateChannelTxPath returns the path to the create channel transaction for
// the named channel.
func (n *Network) CreateChannelTxPath(channelName string) string {
	return filepath.Join(n.Registry.RootDir, fmt.Sprintf("%s_tx.pb", channelName))
}

// OrdererDir returns the path to the configuration directory for the specified
// Orderer.
func (n *Network) OrdererDir(o *topology.Orderer) string {
	return filepath.Join(n.Registry.RootDir, "orderers", o.ID())
}

// OrdererConfigPath returns the path to the orderer configuration document for
// the specified Orderer.
func (n *Network) OrdererConfigPath(o *topology.Orderer) string {
	return filepath.Join(n.OrdererDir(o), "orderer.yaml")
}

// ReadOrdererConfig  unmarshals an orderer's orderer.yaml and returns an
// object approximating its contents.
func (n *Network) ReadOrdererConfig(o *topology.Orderer) *fabricconfig.Orderer {
	var orderer fabricconfig.Orderer
	ordererBytes, err := ioutil.ReadFile(n.OrdererConfigPath(o))
	Expect(err).NotTo(HaveOccurred())

	err = yaml.Unmarshal(ordererBytes, &orderer)
	Expect(err).NotTo(HaveOccurred())

	return &orderer
}

// WriteOrdererConfig serializes the provided configuration as the specified
// orderer's orderer.yaml document.
func (n *Network) WriteOrdererConfig(o *topology.Orderer, config *fabricconfig.Orderer) {
	ordererBytes, err := yaml.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(n.OrdererConfigPath(o), ordererBytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

// ReadConfigTxConfig  unmarshals the configtx.yaml and returns an
// object approximating its contents.
func (n *Network) ReadConfigTxConfig() *fabricconfig.ConfigTx {
	var configtx fabricconfig.ConfigTx
	configtxBytes, err := ioutil.ReadFile(n.ConfigTxConfigPath())
	Expect(err).NotTo(HaveOccurred())

	err = yaml.Unmarshal(configtxBytes, &configtx)
	Expect(err).NotTo(HaveOccurred())

	return &configtx
}

// WriteConfigTxConfig serializes the provided configuration to configtx.yaml.
func (n *Network) WriteConfigTxConfig(config *fabricconfig.ConfigTx) {
	configtxBytes, err := yaml.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(n.ConfigTxConfigPath(), configtxBytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

// PeerDir returns the path to the configuration directory for the specified
// Peer.
func (n *Network) PeerDir(p *topology.Peer) string {
	return filepath.Join(n.Registry.RootDir, "peers", p.ID())
}

// PeerConfigPath returns the path to the peer configuration document for the
// specified peer.
func (n *Network) PeerConfigPath(p *topology.Peer) string {
	return filepath.Join(n.PeerDir(p), "core.yaml")
}

// PeerLedgerDir returns the rwset root directory for the specified peer.
func (n *Network) PeerLedgerDir(p *topology.Peer) string {
	return filepath.Join(n.PeerDir(p), "filesystem/ledgersData")
}

// ReadPeerConfig unmarshals a peer's core.yaml and returns an object
// approximating its contents.
func (n *Network) ReadPeerConfig(p *topology.Peer) *fabricconfig.Core {
	var core fabricconfig.Core
	coreBytes, err := ioutil.ReadFile(n.PeerConfigPath(p))
	Expect(err).NotTo(HaveOccurred())

	err = yaml.Unmarshal(coreBytes, &core)
	Expect(err).NotTo(HaveOccurred())

	return &core
}

// WritePeerConfig serializes the provided configuration as the specified
// peer's core.yaml document.
func (n *Network) WritePeerConfig(p *topology.Peer, config *fabricconfig.Core) {
	coreBytes, err := yaml.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(n.PeerConfigPath(p), coreBytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

// peerUserCryptoDir returns the path to the directory containing the
// certificates and keys for the specified user of the peer.
func (n *Network) peerUserCryptoDir(p *topology.Peer, user, cryptoMaterialType string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return n.userCryptoDir(org, "peerOrganizations", user, cryptoMaterialType)
}

// ordererUserCryptoDir returns the path to the directory containing the
// certificates and keys for the specified user of the orderer.
func (n *Network) ordererUserCryptoDir(o *topology.Orderer, user, cryptoMaterialType string) string {
	org := n.Organization(o.Organization)
	Expect(org).NotTo(BeNil())

	return n.userCryptoDir(org, "ordererOrganizations", user, cryptoMaterialType)
}

// userCryptoDir returns the path to the folder with crypto materials for either peers or orderer organizations
// specific user
func (n *Network) userCryptoDir(org *topology.Organization, nodeOrganizationType, user, cryptoMaterialType string) string {
	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		nodeOrganizationType,
		org.Domain,
		"users",
		fmt.Sprintf("%s@%s", user, org.Domain),
		cryptoMaterialType,
	)
}

// PeerUserMSPDir returns the path to the MSP directory containing the
// certificates and keys for the specified user of the peer.
func (n *Network) PeerUserMSPDir(p *topology.Peer, user string) string {
	return n.peerUserCryptoDir(p, user, "msp")
}

func (n *Network) ViewNodeMSPDir(p *topology.Peer) string {
	switch {
	case n.topology.NodeOUs:
		switch p.Role {
		case "":
			return n.PeerUserMSPDir(p, p.Name)
		case "client":
			return n.PeerUserMSPDir(p, p.Name)
		default:
			return n.PeerLocalMSPDir(p)
		}
	default:
		return n.PeerLocalMSPDir(p)
	}
}

// FSCNodeLocalTLSDir returns the path to the local TLS directory for the peer.
func (n *Network) FSCNodeLocalTLSDir(p *topology.Peer) string {
	switch {
	case n.topology.NodeOUs:
		switch p.Role {
		case "":
			return n.peerUserCryptoDir(p, p.Name, "tls")
		case "client":
			return n.peerUserCryptoDir(p, p.Name, "tls")
		default:
			return n.peerLocalCryptoDir(p, "tls")
		}
	default:
		return n.peerLocalCryptoDir(p, "tls")
	}
}

// IdemixUserMSPDir returns the path to the MSP directory containing the
// idemix-related crypto material for the specified user of the organization.
func (n *Network) IdemixUserMSPDir(o *topology.Organization, user string) string {
	return n.userCryptoDir(o, "peerOrganizations", user, "")
}

// OrdererUserMSPDir returns the path to the MSP directory containing the
// certificates and keys for the specified user of the peer.
func (n *Network) OrdererUserMSPDir(o *topology.Orderer, user string) string {
	return n.ordererUserCryptoDir(o, user, "msp")
}

// PeerUserTLSDir returns the path to the TLS directory containing the
// certificates and keys for the specified user of the peer.
func (n *Network) PeerUserTLSDir(p *topology.Peer, user string) string {
	return n.peerUserCryptoDir(p, user, "tls")
}

// PeerUserCert returns the path to the certificate for the specified user in
// the peer organization.
func (n *Network) PeerUserCert(p *topology.Peer, user string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.PeerUserMSPDir(p, user),
		"signcerts",
		fmt.Sprintf("%s@%s-cert.pem", user, org.Domain),
	)
}

// OrdererUserCert returns the path to the certificate for the specified user in
// the orderer organization.
func (n *Network) OrdererUserCert(o *topology.Orderer, user string) string {
	org := n.Organization(o.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.OrdererUserMSPDir(o, user),
		"signcerts",
		fmt.Sprintf("%s@%s-cert.pem", user, org.Domain),
	)
}

// PeerUserKey returns the path to the private key for the specified user in
// the peer organization.
func (n *Network) PeerUserKey(p *topology.Peer, user string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.PeerUserMSPDir(p, user),
		"keystore",
		"priv_sk",
	)
}

func (n *Network) PeerKey(p *topology.Peer) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.PeerLocalMSPDir(p),
		"keystore",
		"priv_sk",
	)
}

// OrdererUserKey returns the path to the private key for the specified user in
// the orderer organization.
func (n *Network) OrdererUserKey(o *topology.Orderer, user string) string {
	org := n.Organization(o.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.OrdererUserMSPDir(o, user),
		"keystore",
		"priv_sk",
	)
}

// peerLocalCryptoDir returns the path to the local crypto directory for the peer.
func (n *Network) peerLocalCryptoDir(p *topology.Peer, cryptoType string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"peerOrganizations",
		org.Domain,
		"peers",
		fmt.Sprintf("%s.%s", p.Name, org.Domain),
		cryptoType,
	)
}

func (n *Network) peerUserLocalCryptoDir(p *topology.Peer, user, cryptoType string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"peerOrganizations",
		org.Domain,
		"users",
		fmt.Sprintf("%s@%s", user, org.Domain),
		cryptoType,
	)
}

// PeerLocalMSPDir returns the path to the local MSP directory for the peer.
func (n *Network) PeerLocalMSPDir(p *topology.Peer) string {
	return n.peerLocalCryptoDir(p, "msp")
}

func (n *Network) PeerLocalMSPIdentityCert(p *topology.Peer) string {
	return filepath.Join(
		n.peerLocalCryptoDir(p, "msp"),
		"signcerts",
		p.Name+"."+n.Organization(p.Organization).Domain+"-cert.pem",
	)
}

func (n *Network) PeerUserLocalMSPIdentityCert(p *topology.Peer, user string) string {
	return filepath.Join(
		n.peerUserLocalCryptoDir(p, user, "msp"),
		"signcerts",
		p.Name+"@"+n.Organization(p.Organization).Domain+"-cert.pem",
	)
}

func (n *Network) PeerLocalIdemixExtraIdentitiesDir(p *topology.Peer) string {
	return n.peerLocalCryptoDir(p, "extraids")
}

func (n *Network) PeerLocalExtraIdentityDir(p *topology.Peer, id string) string {
	index := 1
	for _, identity := range p.ExtraIdentities {
		if identity.ID == id {
			switch identity.MSPType {
			case "idemix":
				return n.peerLocalCryptoDir(p, filepath.Join("extraids", id))
			case "bccsp":
				return n.peerUserCryptoDir(p, identity.ID, "msp")
			}
		}
		if identity.MSPType == "bccsp" {
			index++
		}
	}
	panic("id not found")
}

// PeerLocalTLSDir returns the path to the local TLS directory for the peer.
func (n *Network) PeerLocalTLSDir(p *topology.Peer) string {
	return n.peerLocalCryptoDir(p, "tls")
}

// PeerCert returns the path to the peer's certificate.
func (n *Network) PeerCert(p *topology.Peer) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.PeerLocalMSPDir(p),
		"signcerts",
		fmt.Sprintf("%s.%s-cert.pem", p.Name, org.Domain),
	)
}

// PeerOrgMSPDir returns the path to the MSP directory of the Peer organization.
func (n *Network) PeerOrgMSPDir(org *topology.Organization) string {
	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"peerOrganizations",
		org.Domain,
		"msp",
	)
}

func (n *Network) Topology() *topology.Topology {
	return n.topology
}

func (n *Network) DefaultIdemixOrgMSPDir() string {
	for _, organization := range n.Organizations {
		if organization.MSPType == "idemix" {
			return filepath.Join(
				n.Registry.RootDir,
				"crypto",
				"peerOrganizations",
				organization.Domain,
			)
		}
	}
	return ""
}

func (n *Network) IdemixOrgMSPDir(org *topology.Organization) string {
	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"peerOrganizations",
		org.Domain,
	)
}

// OrdererOrgMSPDir returns the path to the MSP directory of the Orderer
// organization.
func (n *Network) OrdererOrgMSPDir(o *topology.Organization) string {
	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"ordererOrganizations",
		o.Domain,
		"msp",
	)
}

// OrdererLocalCryptoDir returns the path to the local crypto directory for the
// Orderer.
func (n *Network) OrdererLocalCryptoDir(o *topology.Orderer, cryptoType string) string {
	org := n.Organization(o.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"ordererOrganizations",
		org.Domain,
		"orderers",
		fmt.Sprintf("%s.%s", o.Name, org.Domain),
		cryptoType,
	)
}

// OrdererLocalMSPDir returns the path to the local MSP directory for the
// Orderer.
func (n *Network) OrdererLocalMSPDir(o *topology.Orderer) string {
	return n.OrdererLocalCryptoDir(o, "msp")
}

// OrdererLocalTLSDir returns the path to the local TLS directory for the
// Orderer.
func (n *Network) OrdererLocalTLSDir(o *topology.Orderer) string {
	return n.OrdererLocalCryptoDir(o, "tls")
}

// ProfileForChannel gets the configtxgen profile name associated with the
// specified channel.
func (n *Network) ProfileForChannel(channelName string) string {
	for _, ch := range n.Channels {
		if ch.Name == channelName {
			return ch.Profile
		}
	}
	return ""
}

// CACertsBundlePath returns the path to the bundle of CA certificates for the
// network. This bundle is used when connecting to peers.
func (n *Network) CACertsBundlePath() string {
	return filepath.Join(
		n.Registry.RootDir,
		"crypto",
		"ca-certs.pem",
	)
}

// bootstrapIdemix creates the idemix-related crypto material
func (n *Network) bootstrapIdemix() {
	for _, org := range n.IdemixOrgs() {
		output := n.IdemixOrgMSPDir(org)
		// - ca-keygen
		sess, err := n.Idemixgen(commands.CAKeyGen{
			Output: output,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}
}

func (n *Network) bootstrapExtraIdentities() {
	for i, peer := range n.Peers {
		for j, identity := range peer.ExtraIdentities {
			switch identity.MSPType {
			case "idemix":
				org := n.Organization(identity.Org)
				output := n.IdemixOrgMSPDir(org)
				userOutput := filepath.Join(n.PeerLocalIdemixExtraIdentitiesDir(peer), identity.ID)
				sess, err := n.Idemixgen(commands.SignerConfig{
					CAInput:          output,
					Output:           userOutput,
					OrgUnit:          org.Domain,
					EnrollmentID:     identity.EnrollmentID,
					RevocationHandle: fmt.Sprintf("1%d%d", i, j),
				})
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
			case "bccsp":
				// Nothing to do here cause the extra identities are generated by crypto gen.
			default:
				Expect(identity.MSPType).To(Equal("idemix"))
			}
		}
	}
}

func (n *Network) CheckTopology() {
	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	substring := "github.com/hyperledger-labs/fabric-smart-client"
	n.ExternalBuilders = []fabricconfig.ExternalBuilder{{
		Path: filepath.Join(
			cwd[:strings.Index(cwd, substring)],
			substring,
			"integration",
			"nwo",
			"fabric",
			"externalbuilders",
			"external",
		),
		Name:                 "external",
		PropagateEnvironment: []string{"GOPATH", "GOCACHE", "GOPROXY", "HOME", "PATH"},
	}}

	if n.Templates == nil {
		n.Templates = &topology.Templates{}
	}

	if n.Logging == nil {
		n.Logging = &topology.Logging{
			Spec:   "debug",
			Format: "'%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'",
		}
	}

	for i := 0; i < n.Consensus.Brokers; i++ {
		ports := registry.Ports{}
		for _, portName := range BrokerPortNames() {
			ports[portName] = n.Registry.ReservePort()
		}
		n.PortsByBrokerID[strconv.Itoa(i)] = ports
	}

	for _, o := range n.Orderers {
		ports := registry.Ports{}
		for _, portName := range OrdererPortNames() {
			ports[portName] = n.Registry.ReservePort()
		}
		n.PortsByOrdererID[o.ID()] = ports
	}

	fscTopology := n.Registry.TopologyByName("fsc").(*fsc.Topology)
	users := map[string]int{}
	userNames := map[string][]string{}
	for _, node := range fscTopology.Nodes {
		var extraIdentities []*topology.PeerIdentity

		po := node.PlatformOpts()
		opts := opts.Get(po)
		Expect(opts.Organization()).NotTo(BeEmpty())

		userNames[opts.Organization()] = append(userNames[opts.Organization()], node.Name)

		if opts.AnonymousIdentity() {
			extraIdentities = append(extraIdentities, &topology.PeerIdentity{
				ID:           "idemix",
				EnrollmentID: node.Name,
				MSPType:      "idemix",
				MSPID:        "IdemixOrgMSP",
				Org:          "IdemixOrg",
			})
			n.Registry.AddIdentityAlias(node.Name, "idemix")
		}
		for _, label := range opts.IdemixIdentities() {
			extraIdentities = append(extraIdentities, &topology.PeerIdentity{
				ID:           label,
				EnrollmentID: label,
				MSPType:      "idemix",
				MSPID:        "IdemixOrgMSP",
				Org:          "IdemixOrg",
			})
			n.Registry.AddIdentityAlias(node.Name, label)
		}
		for _, label := range opts.X509Identities() {
			extraIdentities = append(extraIdentities, &topology.PeerIdentity{
				ID:           label,
				MSPType:      "bccsp",
				EnrollmentID: label,
				MSPID:        opts.Organization() + "MSP",
				Org:          opts.Organization(),
			})
			users[opts.Organization()] = users[opts.Organization()] + 1
			userNames[opts.Organization()] = append(userNames[opts.Organization()], label)
			n.Registry.AddIdentityAlias(node.Name, label)
		}

		p := &topology.Peer{
			Name:            node.Name,
			Organization:    opts.Organization(),
			Type:            topology.ViewPeer,
			Role:            opts.Role(),
			Bootstrap:       node.Bootstrap,
			ExecutablePath:  node.ExecutablePath,
			ExtraIdentities: extraIdentities,
		}
		n.Peers = append(n.Peers, p)
		n.Registry.PortsByPeerID[p.ID()] = n.Registry.PortsByPeerID[node.Name]

		// Set paths
		po.Put("NodeLocalCertPath", n.ViewNodeLocalCertPath(p))
		po.Put("NodeLocalPrivateKeyPath", n.ViewNodeLocalPrivateKeyPath(p))
		po.Put("NodeLocalTLSDir", n.PeerLocalTLSDir(p))
	}

	for _, organization := range n.Organizations {
		organization.Users += users[organization.Name]
		organization.UserNames = append(userNames[organization.Name], "User1", "User2")
	}

	for _, p := range n.Peers {
		if p.Type == topology.ViewPeer {
			continue
		}
		ports := registry.Ports{}
		for _, portName := range PeerPortNames() {
			ports[portName] = n.Registry.ReservePort()
		}
		n.Registry.PortsByPeerID[p.ID()] = ports
	}
}

// ConcatenateTLSCACertificates concatenates all TLS CA certificates into a
// single file to be used by peer CLI.
func (n *Network) ConcatenateTLSCACertificates() {
	bundle := &bytes.Buffer{}
	for _, tlsCertPath := range n.listTLSCACertificates() {
		certBytes, err := ioutil.ReadFile(tlsCertPath)
		Expect(err).NotTo(HaveOccurred())
		bundle.Write(certBytes)
	}
	if len(bundle.Bytes()) == 0 {
		return
	}

	err := ioutil.WriteFile(n.CACertsBundlePath(), bundle.Bytes(), 0660)
	Expect(err).NotTo(HaveOccurred())
}

// listTLSCACertificates returns the paths of all TLS CA certificates in the
// network, across all organizations.
func (n *Network) listTLSCACertificates() []string {
	fileName2Path := make(map[string]string)
	filepath.Walk(filepath.Join(n.Registry.RootDir, "crypto"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// File starts with "tlsca" and has "-cert.pem" in it
		if strings.HasPrefix(info.Name(), "tlsca") && strings.Contains(info.Name(), "-cert.pem") {
			fileName2Path[info.Name()] = path
		}
		return nil
	})

	var tlsCACertificates []string
	for _, path := range fileName2Path {
		tlsCACertificates = append(tlsCACertificates, path)
	}
	return tlsCACertificates
}

// CreateAndJoinChannels will create all channels specified in the config that
// are referenced by peers. The referencing peers will then be joined to the
// channel(s).
//
// The network must be running before this is called.
func (n *Network) CreateAndJoinChannels(o *topology.Orderer) {
	for _, c := range n.Channels {
		n.CreateAndJoinChannel(o, c.Name)
	}
}

// CreateAndJoinChannel will create the specified channel. The referencing
// peers will then be joined to the channel.
//
// The network must be running before this is called.
func (n *Network) CreateAndJoinChannel(o *topology.Orderer, channelName string) {
	peers := n.PeersWithChannel(channelName)
	if len(peers) == 0 {
		return
	}

	n.CreateChannel(channelName, o, peers[0])
	n.JoinChannel(channelName, o, peers...)
}

// UpdateChannelAnchors determines the anchor peers for the specified channel,
// creates an anchor peer update transaction for each organization, and submits
// the update transactions to the orderer.
func (n *Network) UpdateChannelAnchors(o *topology.Orderer, channelName string) {
	tempFile, err := ioutil.TempFile("", "update-anchors")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	peersByOrg := map[string]*topology.Peer{}
	for _, p := range n.AnchorsForChannel(channelName) {
		peersByOrg[p.Organization] = p
	}

	for orgName, p := range peersByOrg {
		anchorUpdate := commands.OutputAnchorPeersUpdate{
			OutputAnchorPeersUpdate: tempFile.Name(),
			ChannelID:               channelName,
			Profile:                 n.ProfileForChannel(channelName),
			ConfigPath:              n.Registry.RootDir,
			AsOrg:                   orgName,
		}
		sess, err := n.ConfigTxGen(anchorUpdate)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

		sess, err = n.PeerAdminSession(p, commands.ChannelUpdate{
			ChannelID:  channelName,
			Orderer:    n.OrdererAddress(o, ListenPort),
			File:       tempFile.Name(),
			ClientAuth: n.ClientAuthRequired,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}
}

// VerifyMembership checks that each peer has discovered the expected
// peers in the network
func (n *Network) VerifyMembership(expectedPeers []*topology.Peer, channel string, chaincodes ...string) {
	// all peers currently include _lifecycle as an available chaincode
	chaincodes = append(chaincodes, "_lifecycle")
	expectedDiscoveredPeerMatchers := make([]types.GomegaMatcher, len(expectedPeers))
	for i, peer := range expectedPeers {
		expectedDiscoveredPeerMatchers[i] = n.DiscoveredPeerMatcher(peer, chaincodes...) //n.DiscoveredPeer(peer, chaincodes...)
	}
	for _, peer := range expectedPeers {
		Eventually(DiscoverPeers(n, peer, "User1", channel), n.EventuallyTimeout).Should(ConsistOf(expectedDiscoveredPeerMatchers))
	}
}

// CreateChannel will submit an existing create channel transaction to the
// specified orderer. The channel transaction must exist at the location
// returned by CreateChannelTxPath.  Optionally, additional signers may be
// included in the case where the channel creation tx modifies other
// aspects of the channel config for the new channel.
//
// The orderer must be running when this is called.
func (n *Network) CreateChannel(channelName string, o *topology.Orderer, p *topology.Peer, additionalSigners ...interface{}) {
	channelCreateTxPath := n.CreateChannelTxPath(channelName)
	n.signConfigTransaction(channelCreateTxPath, p, additionalSigners...)

	createChannel := func() int {
		sess, err := n.PeerAdminSession(p, commands.ChannelCreate{
			ChannelID:   channelName,
			Orderer:     n.OrdererAddress(o, ListenPort),
			File:        channelCreateTxPath,
			OutputBlock: "/dev/null",
			ClientAuth:  n.ClientAuthRequired,
		})
		Expect(err).NotTo(HaveOccurred())
		return sess.Wait(n.EventuallyTimeout).ExitCode()
	}
	Eventually(createChannel, n.EventuallyTimeout).Should(Equal(0))
}

// CreateChannelExitCode will submit an existing create channel transaction to
// the specified orderer, wait for the operation to complete, and return the
// exit status of the "peer channel create" command.
//
// The channel transaction must exist at the location returned by
// CreateChannelTxPath and the orderer must be running when this is called.
func (n *Network) CreateChannelExitCode(channelName string, o *topology.Orderer, p *topology.Peer, additionalSigners ...interface{}) int {
	channelCreateTxPath := n.CreateChannelTxPath(channelName)
	n.signConfigTransaction(channelCreateTxPath, p, additionalSigners...)

	sess, err := n.PeerAdminSession(p, commands.ChannelCreate{
		ChannelID:   channelName,
		Orderer:     n.OrdererAddress(o, ListenPort),
		File:        channelCreateTxPath,
		OutputBlock: "/dev/null",
		ClientAuth:  n.ClientAuthRequired,
	})
	Expect(err).NotTo(HaveOccurred())
	return sess.Wait(n.EventuallyTimeout).ExitCode()
}

func (n *Network) signConfigTransaction(channelTxPath string, submittingPeer *topology.Peer, signers ...interface{}) {
	for _, signer := range signers {
		switch signer := signer.(type) {
		case *topology.Peer:
			sess, err := n.PeerAdminSession(signer, commands.SignConfigTx{
				File:       channelTxPath,
				ClientAuth: n.ClientAuthRequired,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

		case *topology.Orderer:
			sess, err := n.OrdererAdminSession(signer, submittingPeer, commands.SignConfigTx{
				File:       channelTxPath,
				ClientAuth: n.ClientAuthRequired,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

		default:
			panic(fmt.Sprintf("unknown signer type %T, expect Peer or Orderer", signer))
		}
	}
}

// JoinChannel will join peers to the specified channel. The orderer is used to
// obtain the current configuration block for the channel.
//
// The orderer and listed peers must be running before this is called.
func (n *Network) JoinChannel(name string, o *topology.Orderer, peers ...*topology.Peer) {
	if len(peers) == 0 {
		return
	}

	tempFile, err := ioutil.TempFile("", "genesis-block")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	sess, err := n.PeerAdminSession(peers[0], commands.ChannelFetch{
		Block:      "0",
		ChannelID:  name,
		Orderer:    n.OrdererAddress(o, ListenPort),
		OutputFile: tempFile.Name(),
		ClientAuth: n.ClientAuthRequired,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

	for _, p := range peers {
		sess, err := n.PeerAdminSession(p, commands.ChannelJoin{
			BlockPath:  tempFile.Name(),
			ClientAuth: n.ClientAuthRequired,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}
}

// Cryptogen starts a gexec.Session for the provided cryptogen command.
func (n *Network) Cryptogen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(n.Components.Cryptogen(), command)
	return n.StartSession(cmd, command.SessionName())
}

// Idemixgen starts a gexec.Session for the provided idemixgen command.
func (n *Network) Idemixgen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(n.Components.Idemixgen(), command)
	return n.StartSession(cmd, command.SessionName())
}

// ConfigTxGen starts a gexec.Session for the provided configtxgen command.
func (n *Network) ConfigTxGen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(n.Components.ConfigTxGen(), command)
	return n.StartSession(cmd, command.SessionName())
}

// Discover starts a gexec.Session for the provided discover command.
func (n *Network) Discover(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(n.Components.Discover(), command)
	cmd.Args = append(cmd.Args, "--peerTLSCA", n.CACertsBundlePath())
	return n.StartSession(cmd, command.SessionName())
}

// ZooKeeperRunner returns a runner for a ZooKeeper instance.
func (n *Network) ZooKeeperRunner(idx int) *runner.ZooKeeper {
	colorCode := n.nextColor()
	name := fmt.Sprintf("zookeeper-%d-%s", idx, n.NetworkID)

	return &runner.ZooKeeper{
		ZooMyID:     idx + 1, //  IDs must be between 1 and 255
		Client:      n.DockerClient,
		Name:        name,
		NetworkName: n.NetworkID,
		OutputStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
		ErrorStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
	}
}

func (n *Network) minBrokersInSync() int {
	if n.Consensus.Brokers < 2 {
		return n.Consensus.Brokers
	}
	return 2
}

func (n *Network) defaultBrokerReplication() int {
	if n.Consensus.Brokers < 3 {
		return n.Consensus.Brokers
	}
	return 3
}

// BrokerRunner returns a runner for an kafka broker instance.
func (n *Network) BrokerRunner(id int, zookeepers []string) *runner.Kafka {
	colorCode := n.nextColor()
	name := fmt.Sprintf("kafka-%d-%s", id, n.NetworkID)

	return &runner.Kafka{
		BrokerID:                 id + 1,
		Client:                   n.DockerClient,
		AdvertisedListeners:      "127.0.0.1",
		HostPort:                 int(n.PortsByBrokerID[strconv.Itoa(id)][HostPort]),
		Name:                     name,
		NetworkName:              n.NetworkID,
		MinInsyncReplicas:        n.minBrokersInSync(),
		DefaultReplicationFactor: n.defaultBrokerReplication(),
		ZooKeeperConnect:         strings.Join(zookeepers, ","),
		OutputStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
		ErrorStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
	}
}

// BrokerGroupRunner returns a runner that manages the processes that make up
// the kafka broker network for fabric.
func (n *Network) BrokerGroupRunner() ifrit.Runner {
	members := grouper.Members{}
	zookeepers := []string{}

	for i := 0; i < n.Consensus.ZooKeepers; i++ {
		zk := n.ZooKeeperRunner(i)
		zookeepers = append(zookeepers, fmt.Sprintf("%s:2181", zk.Name))
		members = append(members, grouper.Member{Name: zk.Name, Runner: zk})
	}

	for i := 0; i < n.Consensus.Brokers; i++ {
		kafka := n.BrokerRunner(i, zookeepers)
		members = append(members, grouper.Member{Name: kafka.Name, Runner: kafka})
	}

	if len(members) == 0 {
		return nil
	}

	return grouper.NewOrdered(syscall.SIGTERM, members)
}

// OrdererRunner returns an ifrit.Runner for the specified orderer. The runner
// can be used to start and manage an orderer process.
func (n *Network) OrdererRunner(o *topology.Orderer) *ginkgomon.Runner {
	cmd := exec.Command(n.Components.Orderer())
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABRIC_CFG_PATH=%s", n.OrdererDir(o)))

	config := ginkgomon.Config{
		AnsiColorCode:     n.nextColor(),
		Name:              o.ID(),
		Command:           cmd,
		StartCheck:        "Beginning to serve requests",
		StartCheckTimeout: 1 * time.Minute,
	}

	// After consensus-type migration, the #brokers is >0, but the type is etcdraft
	if n.Consensus.Type == "kafka" && n.Consensus.Brokers != 0 {
		config.StartCheck = "Start phase completed successfully"
		config.StartCheckTimeout = 3 * time.Minute
	}

	return ginkgomon.New(config)
}

// OrdererGroupRunner returns a runner that can be used to start and stop all
// orderers in a network.
func (n *Network) OrdererGroupRunner() ifrit.Runner {
	members := grouper.Members{}
	for _, o := range n.Orderers {
		members = append(members, grouper.Member{Name: o.ID(), Runner: n.OrdererRunner(o)})
	}
	if len(members) == 0 {
		return nil
	}

	return grouper.NewParallel(syscall.SIGTERM, members)
}

// PeerRunner returns an ifrit.Runner for the specified peer. The runner can be
// used to start and manage a peer process.
func (n *Network) PeerRunner(p *topology.Peer, env ...string) *ginkgomon.Runner {
	cmd := n.peerCommand(
		p.ExecutablePath,
		commands.NodeStart{PeerID: p.ID(), DevMode: p.DevMode},
		"",
		fmt.Sprintf("FABRIC_CFG_PATH=%s", n.PeerDir(p)),
	)
	cmd.Env = append(cmd.Env, env...)

	return ginkgomon.New(ginkgomon.Config{
		AnsiColorCode:     n.nextColor(),
		Name:              p.ID(),
		Command:           cmd,
		StartCheck:        `Started peer with ID=.*, .*, address=`,
		StartCheckTimeout: 1 * time.Minute,
	})
}

// PeerGroupRunner returns a runner that can be used to start and stop all
// peers in a network.
func (n *Network) PeerGroupRunner() ifrit.Runner {
	members := grouper.Members{}
	for _, p := range n.Peers {
		switch {
		case p.Type == topology.FabricPeer:
			members = append(members, grouper.Member{Name: p.ID(), Runner: n.PeerRunner(p)})
		}
	}
	if len(members) == 0 {
		return nil
	}
	return grouper.NewParallel(syscall.SIGTERM, members)
}

func (n *Network) peerCommand(executablePath string, command common.Command, tlsDir string, env ...string) *exec.Cmd {
	cmd := common.NewCommand(n.Components.Peer(executablePath), command)
	cmd.Env = append(cmd.Env, env...)
	cmd.Env = append(cmd.Env, "FABRIC_LOGGING_SPEC="+n.Logging.Spec)

	if n.GRPCLogging {
		cmd.Env = append(cmd.Env, "GRPC_GO_LOG_VERBOSITY_LEVEL=2")
		cmd.Env = append(cmd.Env, "GRPC_GO_LOG_SEVERITY_LEVEL=debug")
	}

	if common.ConnectsToOrderer(command) {
		cmd.Args = append(cmd.Args, "--tls")
		cmd.Args = append(cmd.Args, "--cafile", n.CACertsBundlePath())
	}

	if common.ClientAuthEnabled(command) {
		certfilePath := filepath.Join(tlsDir, "client.crt")
		keyfilePath := filepath.Join(tlsDir, "client.key")

		cmd.Args = append(cmd.Args, "--certfile", certfilePath)
		cmd.Args = append(cmd.Args, "--keyfile", keyfilePath)
	}

	cmd.Args = append(cmd.Args, "--logging-level", n.Logging.Spec)

	// In case we have a peer invoke with multiple certificates,
	// we need to mimic the correct peer CLI usage,
	// so we count the number of --peerAddresses usages
	// we have, and add the same (concatenated TLS CA certificates file)
	// the same number of times to bypass the peer CLI sanity checks
	requiredPeerAddresses := flagCount("--peerAddresses", cmd.Args)
	for i := 0; i < requiredPeerAddresses; i++ {
		cmd.Args = append(cmd.Args, "--tlsRootCertFiles")
		cmd.Args = append(cmd.Args, n.CACertsBundlePath())
	}
	return cmd
}

func flagCount(flag string, args []string) int {
	var c int
	for _, arg := range args {
		if arg == flag {
			c++
		}
	}
	return c
}

// PeerAdminSession starts a gexec.Session as a peer admin for the provided
// peer command. This is intended to be used by short running peer cli commands
// that execute in the context of a peer configuration.
func (n *Network) PeerAdminSession(p *topology.Peer, command common.Command) (*gexec.Session, error) {
	return n.PeerUserSession(p, "Admin", command)
}

// PeerUserSession starts a gexec.Session as a peer user for the provided peer
// command. This is intended to be used by short running peer cli commands that
// execute in the context of a peer configuration.
func (n *Network) PeerUserSession(p *topology.Peer, user string, command common.Command) (*gexec.Session, error) {
	cmd := n.peerCommand(
		p.ExecutablePath,
		command,
		n.PeerUserTLSDir(p, user),
		fmt.Sprintf("FABRIC_CFG_PATH=%s", n.PeerDir(p)),
		fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", n.PeerUserMSPDir(p, user)),
	)
	return n.StartSession(cmd, command.SessionName())
}

// OrdererAdminSession starts a gexec.Session as an orderer admin user. This
// is used primarily to generate orderer configuration updates.
func (n *Network) OrdererAdminSession(o *topology.Orderer, p *topology.Peer, command common.Command) (*gexec.Session, error) {
	cmd := n.peerCommand(
		p.ExecutablePath,
		command,
		n.ordererUserCryptoDir(o, "Admin", "tls"),
		fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", n.Organization(o.Organization).MSPID),
		fmt.Sprintf("FABRIC_CFG_PATH=%s", n.PeerDir(p)),
		fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", n.OrdererUserMSPDir(o, "Admin")),
	)
	return n.StartSession(cmd, command.SessionName())
}

// Peer returns the information about the named Peer in the named organization.
func (n *Network) Peer(orgName, peerName string) *topology.Peer {
	for _, p := range n.PeersInOrg(orgName) {
		if p.Name == peerName {
			return p
		}
	}
	return nil
}

// DiscoveredPeer creates a new DiscoveredPeer from the peer and chaincodes
// passed as arguments.
func (n *Network) DiscoveredPeer(p *topology.Peer, chaincodes ...string) DiscoveredPeer {
	peerCert, err := ioutil.ReadFile(n.PeerCert(p))
	Expect(err).NotTo(HaveOccurred())

	return DiscoveredPeer{
		MSPID:      n.Organization(p.Organization).MSPID,
		Endpoint:   fmt.Sprintf("127.0.0.1:%d", n.PeerPort(p, ListenPort)),
		Identity:   string(peerCert),
		Chaincodes: chaincodes,
	}
}

func (n *Network) DiscoveredPeerMatcher(p *topology.Peer, chaincodes ...string) types.GomegaMatcher {
	peerCert, err := ioutil.ReadFile(n.PeerCert(p))
	Expect(err).NotTo(HaveOccurred())

	return MatchAllFields(Fields{
		"MSPID":      Equal(n.Organization(p.Organization).MSPID),
		"Endpoint":   Equal(fmt.Sprintf("127.0.0.1:%d", n.PeerPort(p, ListenPort))),
		"Identity":   Equal(string(peerCert)),
		"Chaincodes": containElements(chaincodes...),
	})
}

// containElements succeeds if a slice contains the passed in elements.
func containElements(elements ...string) types.GomegaMatcher {
	ms := make([]types.GomegaMatcher, 0, len(elements))
	for _, element := range elements {
		ms = append(ms, &matchers.ContainElementMatcher{
			Element: element,
		})
	}
	return &matchers.AndMatcher{
		Matchers: ms,
	}
}

// Orderer returns the information about the named Orderer.
func (n *Network) Orderer(name string) *topology.Orderer {
	for _, o := range n.Orderers {
		if o.Name == name {
			return o
		}
	}
	return nil
}

// Organization returns the information about the named Organization.
func (n *Network) Organization(orgName string) *topology.Organization {
	for _, org := range n.Organizations {
		if org.Name == orgName {
			return org
		}
	}
	return nil
}

// Consortium returns information about the named Consortium.
func (n *Network) Consortium(name string) *topology.Consortium {
	for _, c := range n.Consortiums {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// PeerOrgs returns all Organizations associated with at least one Peer.
func (n *Network) PeerOrgs() []*topology.Organization {
	orgsByName := map[string]*topology.Organization{}
	for _, p := range n.Peers {
		if n.Organization(p.Organization).MSPType != "idemix" {
			orgsByName[p.Organization] = n.Organization(p.Organization)
		}
	}

	var orgs []*topology.Organization
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

func (n *Network) PeerOrgsByPeers(peers []*topology.Peer) []*topology.Organization {
	orgsByName := map[string]*topology.Organization{}
	for _, p := range peers {
		if n.Organization(p.Organization).MSPType != "idemix" {
			orgsByName[p.Organization] = n.Organization(p.Organization)
		}
	}

	orgs := []*topology.Organization{}
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

// IdemixOrgs returns all Organizations of type idemix.
func (n *Network) IdemixOrgs() []*topology.Organization {
	orgs := []*topology.Organization{}
	for _, org := range n.Organizations {
		if org.MSPType == "idemix" {
			orgs = append(orgs, org)
		}
	}
	return orgs
}

// PeersWithChannel returns all Peer instances that have joined the named
// channel.
func (n *Network) PeersWithChannel(chanName string) []*topology.Peer {
	var peers []*topology.Peer
	for _, p := range n.Peers {
		switch {
		case p.Type == topology.FabricPeer:
			for _, c := range p.Channels {
				if c.Name == chanName {
					peers = append(peers, p)
				}
			}
		}
	}

	// This is a bit of a hack to make the output of this function deterministic.
	// When this function's output is supplied as input to functions such as ApproveChaincodeForMyOrg
	// it causes a different subset of peers to be picked, which can create flakiness in tests.
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].Organization < peers[j].Organization {
			return true
		}

		return peers[i].Organization == peers[j].Organization && peers[i].Name < peers[j].Name
	})
	return peers
}

// AnchorsForChannel returns all Peer instances that are anchors for the
// named channel.
func (n *Network) AnchorsForChannel(chanName string) []*topology.Peer {
	anchors := []*topology.Peer{}
	for _, p := range n.Peers {
		switch {
		case p.Type == topology.FabricPeer:
			for _, pc := range p.Channels {
				if pc.Name == chanName && pc.Anchor {
					anchors = append(anchors, p)
				}
			}
		}
	}
	return anchors
}

// AnchorsInOrg returns all peers that are an anchor for at least one channel
// in the named organization.
func (n *Network) AnchorsInOrg(orgName string) []*topology.Peer {
	anchors := []*topology.Peer{}
	for _, p := range n.PeersInOrg(orgName) {
		if p.Anchor() {
			anchors = append(anchors, p)
			break
		}
	}

	// No explicit anchor means all peers are anchors.
	if len(anchors) == 0 {
		anchors = n.PeersInOrg(orgName)
	}

	return anchors
}

// OrderersInOrg returns all Orderer instances owned by the named organaiztion.
func (n *Network) OrderersInOrg(orgName string) []*topology.Orderer {
	orderers := []*topology.Orderer{}
	for _, o := range n.Orderers {
		if o.Organization == orgName {
			orderers = append(orderers, o)
		}
	}
	return orderers
}

// OrgsForOrderers returns all Organization instances that own at least one of
// the named orderers.
func (n *Network) OrgsForOrderers(ordererNames []string) []*topology.Organization {
	orgsByName := map[string]*topology.Organization{}
	for _, name := range ordererNames {
		orgName := n.Orderer(name).Organization
		orgsByName[orgName] = n.Organization(orgName)
	}
	orgs := []*topology.Organization{}
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

// OrdererOrgs returns all Organization instances that own at least one
// orderer.
func (n *Network) OrdererOrgs() []*topology.Organization {
	orgsByName := map[string]*topology.Organization{}
	for _, o := range n.Orderers {
		orgsByName[o.Organization] = n.Organization(o.Organization)
	}

	orgs := []*topology.Organization{}
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

// PeersInOrg returns all Peer instances that are owned by the named
// organization.
func (n *Network) PeersInOrg(orgName string) []*topology.Peer {
	var peers []*topology.Peer
	for _, o := range n.Peers {
		if o.Organization == orgName {
			peers = append(peers, o)
		}
	}
	return peers
}

const (
	ChaincodePort  registry.PortName = "Chaincode"
	EventsPort     registry.PortName = "Events"
	HostPort       registry.PortName = "HostPort"
	ListenPort     registry.PortName = "Listen"
	ProfilePort    registry.PortName = "Profile"
	OperationsPort registry.PortName = "Operations"
	ViewPort       registry.PortName = "View"
	P2PPort        registry.PortName = "P2P"
	ClusterPort    registry.PortName = "Cluster"
)

// PeerPortNames returns the list of ports that need to be reserved for a Peer.
func PeerPortNames() []registry.PortName {
	return []registry.PortName{ListenPort, ChaincodePort, EventsPort, ProfilePort, OperationsPort, P2PPort}
}

// OrdererPortNames  returns the list of ports that need to be reserved for an
// Orderer.
func OrdererPortNames() []registry.PortName {
	return []registry.PortName{ListenPort, ProfilePort, OperationsPort, ClusterPort}
}

// BrokerPortNames returns the list of ports that need to be reserved for a
// Kafka broker.
func BrokerPortNames() []registry.PortName {
	return []registry.PortName{HostPort}
}

// BrokerAddresses returns the list of broker addresses for the network.
func (n *Network) BrokerAddresses(portName registry.PortName) []string {
	addresses := []string{}
	for _, ports := range n.PortsByBrokerID {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:%d", ports[portName]))
	}
	return addresses
}

// OrdererAddress returns the address (host and port) exposed by the Orderer
// for the named port. Commands line tools should use the returned address when
// connecting to the orderer.
//
// This assumes that the orderer is listening on 0.0.0.0 or 127.0.0.1 and is
// available on the loopback address.
func (n *Network) OrdererAddress(o *topology.Orderer, portName registry.PortName) string {
	return fmt.Sprintf("127.0.0.1:%d", n.OrdererPort(o, portName))
}

// OrdererPort returns the named port reserved for the Orderer instance.
func (n *Network) OrdererPort(o *topology.Orderer, portName registry.PortName) uint16 {
	ordererPorts := n.PortsByOrdererID[o.ID()]
	Expect(ordererPorts).NotTo(BeNil())
	return ordererPorts[portName]
}

// PeerAddress returns the address (host and port) exposed by the Peer for the
// named port. Commands line tools should use the returned address when
// connecting to a peer.
//
// This assumes that the peer is listening on 0.0.0.0 and is available on the
// loopback address.
func (n *Network) PeerAddress(p *topology.Peer, portName registry.PortName) string {
	return fmt.Sprintf("127.0.0.1:%d", n.PeerPort(p, portName))
}

func (n *Network) PeerAddressByName(p *topology.Peer, portName registry.PortName) string {
	return fmt.Sprintf("127.0.0.1:%d", n.PeerPortByName(p, portName))
}

// PeerPort returns the named port reserved for the Peer instance.
func (n *Network) PeerPort(p *topology.Peer, portName registry.PortName) uint16 {
	peerPorts := n.Registry.PortsByPeerID[p.ID()]
	if peerPorts == nil {
		fmt.Printf("PeerPort [%s,%s] not found", p.ID(), portName)
	}
	Expect(peerPorts).NotTo(BeNil(), "PeerPort [%s,%s] not found", p.ID(), portName)
	return peerPorts[portName]
}

func (n *Network) PeerPortByName(p *topology.Peer, portName registry.PortName) uint16 {
	peerPorts := n.Registry.PortsByPeerID[p.Name]
	Expect(peerPorts).NotTo(BeNil())
	return peerPorts[portName]
}

func (n *Network) BootstrapNode(me *topology.Peer) string {
	for _, p := range n.Peers {
		if p.Bootstrap {
			if p.Name == me.Name {
				return ""
			}
			return p.Name
		}
	}
	return ""
}

func (n *Network) nextColor() string {
	color := n.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	n.colorIndex++
	return fmt.Sprintf("%dm", color)
}

func (n *Network) NodeVaultDir(peer *topology.Peer) string {
	return filepath.Join(n.Registry.RootDir, "fscnodes", peer.ID(), "vault")
}

// StartSession executes a command session. This should be used to launch
// command line tools that are expected to run to completion.
func (n *Network) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
	ansiColorCode := n.nextColor()
	fmt.Fprintf(
		ginkgo.GinkgoWriter,
		"\x1b[33m[d]\x1b[%s[%s]\x1b[0m starting %s %s\n",
		ansiColorCode,
		name,
		filepath.Base(cmd.Args[0]),
		strings.Join(cmd.Args[1:], " "),
	)
	return gexec.Start(
		cmd,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
	)
}

func (n *Network) GenerateCryptoConfig() {
	crypto, err := os.Create(n.CryptoConfigPath())
	Expect(err).NotTo(HaveOccurred())
	defer crypto.Close()

	t, err := template.New("crypto").Parse(n.Templates.CryptoTemplate())
	Expect(err).NotTo(HaveOccurred())

	//pw := gexec.NewPrefixedWriter("[crypto-config.yaml] ", ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(crypto), n)
	Expect(err).NotTo(HaveOccurred())
}

func (n *Network) GenerateConfigTxConfig() {
	config, err := os.Create(n.ConfigTxConfigPath())
	Expect(err).NotTo(HaveOccurred())
	defer config.Close()

	t, err := template.New("configtx").Parse(n.Templates.ConfigTxTemplate())
	Expect(err).NotTo(HaveOccurred())

	//pw := gexec.NewPrefixedWriter("[configtx.yaml] ", ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(config), n)
	Expect(err).NotTo(HaveOccurred())
}

func (n *Network) GenerateOrdererConfig(o *topology.Orderer) {
	err := os.MkdirAll(n.OrdererDir(o), 0755)
	Expect(err).NotTo(HaveOccurred())

	orderer, err := os.Create(n.OrdererConfigPath(o))
	Expect(err).NotTo(HaveOccurred())
	defer orderer.Close()

	t, err := template.New("orderer").Funcs(template.FuncMap{
		"Orderer":    func() *topology.Orderer { return o },
		"ToLower":    func(s string) string { return strings.ToLower(s) },
		"ReplaceAll": func(s, old, new string) string { return strings.Replace(s, old, new, -1) },
	}).Parse(n.Templates.OrdererTemplate())
	Expect(err).NotTo(HaveOccurred())

	//pw := gexec.NewPrefixedWriter(fmt.Sprintf("[%s#orderer.yaml] ", o.ID()), ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(orderer), n)
	Expect(err).NotTo(HaveOccurred())
}

func (n *Network) GenerateCoreConfig(p *topology.Peer) {
	switch p.Type {
	case topology.FabricPeer:
		err := os.MkdirAll(n.PeerDir(p), 0755)
		Expect(err).NotTo(HaveOccurred())

		core, err := os.Create(n.PeerConfigPath(p))
		Expect(err).NotTo(HaveOccurred())
		defer core.Close()

		coreTemplate := n.Templates.CoreTemplate()

		t, err := template.New("peer").Funcs(template.FuncMap{
			"Peer":                      func() *topology.Peer { return p },
			"Orderer":                   func() *topology.Orderer { return n.Orderers[0] },
			"PeerLocalExtraIdentityDir": func(p *topology.Peer, id string) string { return n.PeerLocalExtraIdentityDir(p, id) },
			"ToLower":                   func(s string) string { return strings.ToLower(s) },
			"ReplaceAll":                func(s, old, new string) string { return strings.Replace(s, old, new, -1) },
		}).Parse(coreTemplate)
		Expect(err).NotTo(HaveOccurred())

		//pw := gexec.NewPrefixedWriter(fmt.Sprintf("[%s#core.yaml] ", p.ID()), ginkgo.GinkgoWriter)
		extension := bytes.NewBuffer([]byte{})
		err = t.Execute(io.MultiWriter(core, extension), n)
		n.Registry.AddExtension(p.ID(), registry.FabricExtension, extension.String())
		Expect(err).NotTo(HaveOccurred())
	case topology.ViewPeer:
		err := os.MkdirAll(n.PeerDir(p), 0755)
		Expect(err).NotTo(HaveOccurred())

		var refPeers []*topology.Peer
		coreTemplate := n.Templates.CoreTemplate()
		if p.Type == topology.ViewPeer {
			coreTemplate = n.Templates.ViewExtensionTemplate()
			peers := n.PeersInOrg(p.Organization)
			for _, peer := range peers {
				if peer.Type == topology.FabricPeer {
					refPeers = append(refPeers, peer)
				}
			}
		}

		t, err := template.New("peer").Funcs(template.FuncMap{
			"Peer":                      func() *topology.Peer { return p },
			"Orderers":                  func() []*topology.Orderer { return n.Orderers },
			"PeerLocalExtraIdentityDir": func(p *topology.Peer, id string) string { return n.PeerLocalExtraIdentityDir(p, id) },
			"ToLower":                   func(s string) string { return strings.ToLower(s) },
			"ReplaceAll":                func(s, old, new string) string { return strings.Replace(s, old, new, -1) },
			"Peers":                     func() []*topology.Peer { return refPeers },
			"OrdererAddress":            func(o *topology.Orderer, portName registry.PortName) string { return n.OrdererAddress(o, portName) },
			"PeerAddress":               func(o *topology.Peer, portName registry.PortName) string { return n.PeerAddress(o, portName) },
			"CACertsBundlePath":         func() string { return n.CACertsBundlePath() },
			"NodeVaultPath":             func() string { return n.NodeVaultDir(p) },
		}).Parse(coreTemplate)
		Expect(err).NotTo(HaveOccurred())

		//pw := gexec.NewPrefixedWriter(fmt.Sprintf("[%s#extension#core.yaml] ", p.ID()), ginkgo.GinkgoWriter)
		extension := bytes.NewBuffer([]byte{})
		err = t.Execute(io.MultiWriter(extension), n)
		Expect(err).NotTo(HaveOccurred())
		n.Registry.AddExtension(p.Name, registry.FabricExtension, extension.String())
	}
}

func (n *Network) PeersByName(names []string) []*topology.Peer {
	var peers []*topology.Peer
	for _, p := range n.Peers {
		for _, name := range names {
			if p.Name == name {
				peers = append(peers, p)
				break
			}
		}
	}
	return peers
}

func (n *Network) PeerByName(name string) *topology.Peer {
	for _, p := range n.Peers {
		if p.Name == name {
			return p
		}
	}
	return nil
}
