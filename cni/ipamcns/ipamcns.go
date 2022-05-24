// Copyright 2022 Microsoft. All rights reserved.
// MIT License

package ipamcns

import (
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/pkg/errors"

	cniSkel "github.com/containernetworking/cni/pkg/skel"
)

// TODO
type plugin struct {
	*cni.Plugin
}

// TODO
func NewPlugin(name string, config *common.PluginConfig) (*plugin, error) {
	basePlugin, err := cni.NewPlugin(name, config.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "Create base plugin")
	}
	return &plugin{Plugin: basePlugin}, nil
}

// Starts the plugin.
func (p *plugin) Start(config *common.PluginConfig) error {
	if err := p.Initialize(config); err != nil {
		return errors.Wrapf(err, "Initialize base plugin")
	}
	log.Printf("[cni-ipam] Plugin started")
	return nil
}

// Stops the plugin.
func (p *plugin) Stop() {
	p.Uninitialize()
	log.Printf("[cni-ipam] Plugin stopped")
}

//
// CNI implementation
// https://github.com/containernetworking/cni/blob/master/SPEC.md
//

// Add handles CNI add commands.
func (p *plugin) Add(args *cniSkel.CmdArgs) error {
	// TODO
	return nil
}

// Get handles CNI Get commands.
func (p *plugin) Get(args *cniSkel.CmdArgs) error {
	return nil
}

// Delete handles CNI delete commands.
func (p *plugin) Delete(args *cniSkel.CmdArgs) error {
	// TODO
	return nil
}

// Update handles CNI update command.
func (p *plugin) Update(args *cniSkel.CmdArgs) error {
	return nil
}
