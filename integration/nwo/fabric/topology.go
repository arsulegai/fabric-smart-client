/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"

var (
	ImplicitMetaReaders              = &topology.Policy{Name: "Readers", Type: "ImplicitMeta", Rule: "ANY Readers"}
	ImplicitMetaWriters              = &topology.Policy{Name: "Writers", Type: "ImplicitMeta", Rule: "ANY Writers"}
	ImplicitMetaAdmins               = &topology.Policy{Name: "Admins", Type: "ImplicitMeta", Rule: "ANY Admins"}
	ImplicitMetaLifecycleEndorsement = &topology.Policy{Name: "LifecycleEndorsement", Type: "ImplicitMeta", Rule: "MAJORITY Endorsement"}
	ImplicitMetaEndorsement          = &topology.Policy{Name: "Endorsement", Type: "ImplicitMeta", Rule: "ANY Endorsement"}
)

const (
	TopologyName = "fabric"
)

// NewDefaultTopology is a configuration with two organizations and one peer per org.
func NewDefaultTopology() *topology.Topology {
	return &topology.Topology{
		TopologyName: "fabric",
		Logging: &topology.Logging{
			Spec:   "grpc=error:chaincode=debug:endorser=debug:info",
			Format: "'%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'",
		},
		Organizations: []*topology.Organization{{
			Name:          "OrdererOrg",
			MSPID:         "OrdererMSP",
			Domain:        "example.com",
			EnableNodeOUs: false,
			Users:         0,
			CA:            &topology.CA{Hostname: "ca"},
		}},
		Consortiums: []*topology.Consortium{{
			Name: "SampleConsortium",
		}},
		Consensus: &topology.Consensus{
			Type: "solo",
		},
		SystemChannel: &topology.SystemChannel{
			Name:    "systemchannel",
			Profile: "OrgsOrdererGenesis",
		},
		Orderers: []*topology.Orderer{
			{Name: "orderer", Organization: "OrdererOrg"},
		},
		Channels: []*topology.Channel{
			{Name: "testchannel", Profile: "OrgsChannel", Default: true},
		},
		Profiles: []*topology.Profile{{
			Name:     "OrgsOrdererGenesis",
			Orderers: []string{"orderer"},
		}, {
			Name:       "OrgsChannel",
			Consortium: "SampleConsortium",
			Policies: []*topology.Policy{
				ImplicitMetaReaders,
				ImplicitMetaWriters,
				ImplicitMetaAdmins,
				ImplicitMetaLifecycleEndorsement,
				ImplicitMetaEndorsement,
			},
		}},
		ChaincodeMode: "net",
	}
}
