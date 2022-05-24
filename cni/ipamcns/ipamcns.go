// Copyright 2022 Microsoft. All rights reserved.
// MIT License

package ipamcns

import (
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/common"

	cniSkel "github.com/containernetworking/cni/pkg/skel"
)

// TODO
type plugin struct {
	*cni.Plugin
}

// TODO
func NewPlugin(name string, config *common.PluginConfig) (*plugin, error) {
	// TODO
	return &plugin{}, nil
}

// Starts the plugin.
func (p *plugin) Start(config *common.PluginConfig) error {
	// TODO
	return nil
}

// Stops the plugin.
func (p *plugin) Stop() {
	// TODO
}

// Configure parses and applies the given network configuration.
func (p *plugin) Configure(stdinData []byte) (*cni.NetworkConfig, error) {
	// TODO
	return nil, nil
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
