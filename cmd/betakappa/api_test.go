package betakappa_test

import (
	"testing"

	bk "github.com/ethersphere/beekeeper/cmd/betakappa"
)

func TestUsageExample(t *testing.T) {
	k8s := &bk.K8S{ /* config goes here */ }

	// set up cluster
	cluster := k8s.NewCluster()

	// boot nodes
	bnCfg := &bk.NodeGroupConfig{}
	boots := cluster.Add(bnCfg)

	bNode1Cfg := &bk.NodeConfig{}
	boot1 := boots.Add(bNode1Cfg)
	_ = boot1.Start()
	_ = boot1.RunningNode().Fund()

	bNode2Cfg := &bk.NodeConfig{}
	boot2 := boots.Add(bNode2Cfg)
	_ = boot2.Start()
	_ = boot2.RunningNode().Fund()

	// bees
	beeCfg := &bk.NodeGroupConfig{}
	bees := cluster.Add(beeCfg)

	bee1Cfg := &bk.NodeConfig{}
	bee1 := bees.Add(bee1Cfg)
	_ = bee1.Start()
	_ = bee1.RunningNode().Fund()

	bee2Cfg := &bk.NodeConfig{}
	bee2 := bees.Add(bee2Cfg)
	_ = bee2.Start()
	_ = bee2.RunningNode().Fund()

	// light bees
	lightBeeCfg := &bk.NodeGroupConfig{}
	lightNodes := cluster.Add(lightBeeCfg)

	lightBee1Cfg := &bk.NodeConfig{}
	light1 := lightNodes.Add(lightBee1Cfg)
	_ = light1.Start()
	_ = light1.RunningNode().Fund()

	lightBee2Cfg := &bk.NodeConfig{}
	light2 := lightNodes.Add(lightBee2Cfg)
	_ = light2.Start()
	_ = light2.RunningNode().Fund()

	// start
	cluster.ForEachNode(func(n *bk.Node) error {
		return n.Start()
	})

	// tests
	lightBeeNodes := lightNodes.RunningNodes()

	// tests
	lightBee := lightBeeNodes.Bees().PickRandom()

	var file struct{}

	_ = lightBee.Upload(file)

	var fullBees bk.RunningNodes
	fullBees = append(fullBees, boots.RunningNodes()...)
	fullBees = append(fullBees, bees.RunningNodes()...)

	bk.ExpectToHaveFile(file, fullBees...)

	// clean up
	cluster.ForEachNode(func(n *bk.Node) error {
		_ = n.RunningNode().Stop()
		return n.Remove()
	})

	_ = cluster.ShutDown()
}
