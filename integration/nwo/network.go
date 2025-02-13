/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nwo

import (
	"syscall"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/runner"
)

var logger = flogging.MustGetLogger("fsc.integration")

type Platform interface {
	Name() string

	GenerateConfigTree()
	GenerateArtifacts()
	Load()

	Members() []grouper.Member
	PostRun()
	Cleanup()
}

type Network struct {
	Processes []ifrit.Process
	Members   grouper.Members

	Platforms              []Platform
	StartEventuallyTimeout time.Duration
	StopEventuallyTimeout  time.Duration
	ViewMembers            grouper.Members
}

func New(platforms ...Platform) *Network {
	return &Network{
		Platforms:              platforms,
		StartEventuallyTimeout: 10 * time.Minute,
		StopEventuallyTimeout:  time.Minute,
	}
}

func (n *Network) Generate() {
	logger.Infof("Generate Configuration...")
	for _, platform := range n.Platforms {
		platform.GenerateConfigTree()
	}

	for _, platform := range n.Platforms {
		platform.GenerateArtifacts()
	}
	logger.Infof("Generate Configuration...done!")
}

func (n *Network) Load() {
	logger.Infof("Load Configuration...")
	for _, platform := range n.Platforms {
		platform.Load()
	}
	logger.Infof("Load Configuration...done")
}

func (n *Network) Start() {
	logger.Infof("Starting...")

	logger.Infof("Collect members...")
	members := grouper.Members{}

	fscMembers := grouper.Members{}
	for _, platform := range n.Platforms {
		logger.Infof("From [%s]...", platform.Name())
		m := platform.Members()
		if m == nil {
			continue
		}
		for _, member := range m {
			logger.Infof("Adding member [%s]", member.Name)
		}

		if platform.Name() == "fsc" {
			fscMembers = append(fscMembers, m...)
		} else {
			members = append(members, m...)
		}

	}
	n.Members = members
	n.ViewMembers = fscMembers

	logger.Infof("Run nodes...")

	// Execute members on their own stuff...
	Runner := runner.NewOrdered(syscall.SIGTERM, members)
	process := ifrit.Invoke(Runner)
	n.Processes = append(n.Processes, process)
	Eventually(process.Ready(), n.StartEventuallyTimeout).Should(BeClosed())

	// Execute the fsc members in isolation so can be stopped and restarted as needed
	for _, member := range fscMembers {
		runner := runner.NewOrdered(syscall.SIGTERM, []grouper.Member{member})
		process := ifrit.Invoke(runner)
		Eventually(process.Ready(), n.StartEventuallyTimeout).Should(BeClosed())
		n.Processes = append(n.Processes, process)
	}

	logger.Infof("Post execution...")
	for _, platform := range n.Platforms {
		platform.PostRun()
	}
}

func (n *Network) Stop() {
	logger.Infof("Stopping...")
	if len(n.Processes) != 0 {
		logger.Infof("Sending sigtem signal...")
		for _, process := range n.Processes {
			process.Signal(syscall.SIGTERM)
			Eventually(process.Wait(), n.StopEventuallyTimeout).Should(Receive())
		}
	}

	logger.Infof("Cleanup...")
	for _, platform := range n.Platforms {
		platform.Cleanup()
	}
	logger.Infof("Stopping...done!")
}

func (n *Network) StopViewNode(id string) {
	logger.Infof("Stopping fsc node [%s]...", id)
	for _, member := range n.ViewMembers {
		if member.Name == id {
			member.Runner.(*runner.Runner).Stop()
			logger.Infof("Stopping fsc node [%s]...done", id)
			return
		}
	}
	logger.Errorf("Stopping fsc node [%s]...not found", id)
}

func (n *Network) StartViewNode(id string) {
	logger.Infof("Starting fsc node [%s]...", id)
	for _, member := range n.ViewMembers {
		if member.Name == id {
			runner := runner.NewOrdered(syscall.SIGTERM, []grouper.Member{{
				Name: id, Runner: member.Runner.(*runner.Runner).Clone(),
			}})
			member.Runner = runner
			process := ifrit.Invoke(runner)
			Eventually(process.Ready(), n.StartEventuallyTimeout).Should(BeClosed())
			n.Processes = append(n.Processes, process)
			logger.Infof("Starting fsc node [%s]...done", id)
			return
		}
	}
	logger.Errorf("Starting fsc node [%s]...not found", id)
}
