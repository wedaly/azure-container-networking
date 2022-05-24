// Copyright 2022 Microsoft. All rights reserved.
// MIT License

package ipamcns

import (
	"context"
	"net"
	"time"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cns"
	cnsclient "github.com/Azure/azure-container-networking/cns/client"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/network"
	"github.com/pkg/errors"

	cniSkel "github.com/containernetworking/cni/pkg/skel"
)

const (
	cnsBaseUrl    = "" // fallback to default http://localhost:10090
	cnsReqTimeout = 15 * time.Second
)

// TODO
type plugin struct {
	*cni.Plugin
	cnsClient *cnsclient.Client
}

// TODO
func NewPlugin(name string, config *common.PluginConfig) (*plugin, error) {
	basePlugin, err := cni.NewPlugin(name, config.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "Create base plugin")
	}

	cnsClient, err := cnsclient.New(cnsBaseUrl, cnsReqTimeout)
	if err != nil {
		return nil, errors.Wrapf(err, "Initializing CNS client")
	}

	p := &plugin{
		Plugin:    basePlugin,
		cnsClient: cnsClient,
	}

	return p, nil
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
	ctx := context.TODO() // explain this, set timeout?

	// TODO: explain this...
	endpointId := network.GetEndpointID(args)

	cnsReq := cns.IPConfigRequest{
		PodInterfaceID:      endpointId,
		InfraContainerID:    args.ContainerID,
		OrchestratorContext: "", // TODO
	}

	resp, err := p.cnsClient.RequestIPAddress(ctx, cnsReq)
	if err != nil {
		log.Printf("Failed to get IP address from CNS with error %s, response: %v", err, resp)
		return errors.Wrapf(err, "CNS client RequestIPAddress")
	}

	ipnet, gw, err := interpretCNSResponse(resp)
	if err != nil {
		return errors.Wrapf(err, "Could not interpret CNS response")
	}

	// TODO: worry about locking...


	// TODO: need to output something, right?
	cniResult := &cniTypesCurr.Result{
		IPs: []*cniTypesCurr.IPConfig{
			Version: "4",
			Address: net.IPNet{
				IP:   ip,
				Mask: ncipnet,
			},
			Gateway: "", // TODO
		},
		Routes: []*cniTypes.Route{
			{
				Dst: "", // TODO
				GW:  ncwg,
			},
		},
	}

	res, err := result.GetAsVersion(nwCfg.CNIVersion) // TODO: need the version...
	if err != nil {
		log.Printf("TODO")
		return errors.Wrapf(err, "TODO")
	}

	res.Print()
	return nil
}

func interpretCNSResponse(resp *cns.IPConfigResponse) (net.IPNet, net.IP, error) {
	podCIDR := fmt.Sprintf(
		"%s/%s",
		resp.PodIpInfo.PodIPConfig.IPAddress,
		resp.PodIpInfo.NetworkContainerPrimaryIPConfig.IPSubnet.PrefixLength,
	)
	_, podIPNet, err := net.ParseCIDR(podCIDR)
	if err != nil {
		return errors.Wrapf(err, "CNS returned invalid pod CIDR %q", podCIDR)
	}

	ncGatewayIPAddress := resp.PodIpInfo.NetworkContainerPrimaryIPConfig.GatewayIPAddress
	gwIP := net.ParseIP(ncGatewayIPAddress)
	if gwIP == nil {
		return net.IPNet{}, nil, fmt.Errorf("CNS returned an invalid gateway address: %s", ncGatewayIPAddress)
	}

	return podIPNet, gwIP, nil
}

// Get handles CNI Get commands.
func (p *plugin) Get(args *cniSkel.CmdArgs) error {
	return nil
}

// Delete handles CNI delete commands.
func (p *plugin) Delete(args *cniSkel.CmdArgs) error {
	// TODO
	// instantiate cns client, make the req
	// worry about locking...
	return nil
}

// Update handles CNI update command.
func (p *plugin) Update(args *cniSkel.CmdArgs) error {
	return nil
}
