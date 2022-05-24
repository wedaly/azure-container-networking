// Copyright 2022 Microsoft. All rights reserved.
// MIT License

package ipamcns

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Azure/azure-container-networking/cni"
	cninetwork "github.com/Azure/azure-container-networking/cni/network"
	"github.com/Azure/azure-container-networking/cns"
	cnsclient "github.com/Azure/azure-container-networking/cns/client"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/network"
	"github.com/pkg/errors"

	cniSkel "github.com/containernetworking/cni/pkg/skel"
	cniTypes "github.com/containernetworking/cni/pkg/types"
	cniTypesCurr "github.com/containernetworking/cni/pkg/types/current"
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

	orchestratorContext, err := cnsOrchestratorContext(args)
	if err != nil {
		// TODO
		return err
	}

	// TODO: explain this...
	endpointId := cninetwork.GetEndpointID(args)

	cnsReq := cns.IPConfigRequest{
		PodInterfaceID:      endpointId,
		InfraContainerID:    args.ContainerID,
		OrchestratorContext: orchestratorContext,
	}

	resp, err := p.cnsClient.RequestIPAddress(ctx, cnsReq)
	if err != nil {
		log.Printf("Failed to get IP address from CNS with error %s, response: %v", err, resp)
		return errors.Wrapf(err, "CNS client RequestIPAddress")
	}

	podIPNet, gwIP, err := interpretIPConfigResponse(resp)
	if err != nil {
		return errors.Wrapf(err, "Could not interpret CNS IPConfigResponse")
	}

	// TODO: worry about locking...
	nwCfg, err := cni.ParseNetworkConfig(args.StdinData)
	if err != nil {
		// TODO
		return err
	}

	// TODO: need to output something, right?
	cniResult := &cniTypesCurr.Result{
		IPs: []*cniTypesCurr.IPConfig{
			{
				Version: "4",
				Address: *podIPNet,
				Gateway: gwIP,
			},
		},
		Routes: []*cniTypes.Route{
			{
				Dst: network.Ipv4DefaultRouteDstPrefix,
				GW:  gwIP,
			},
		},
	}

	versionedCniResult, err := cniResult.GetAsVersion(nwCfg.CNIVersion) // TODO: need the version...
	if err != nil {
		log.Printf("TODO")
		return errors.Wrapf(err, "TODO")
	}

	versionedCniResult.Print()
	return nil
}

func cnsOrchestratorContext(args *cniSkel.CmdArgs) (json.RawMessage, error) {
	podCfg, err := cni.ParseCniArgs(args.Args)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not parse CNI args")
	}

	podInfo := cns.KubernetesPodInfo{
		PodName:      string(podCfg.K8S_POD_NAME),
		PodNamespace: string(podCfg.K8S_POD_NAMESPACE),
	}

	jsonMsg, err := json.Marshal(podInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not marshal podInfo to JSON")
	}

	return jsonMsg, nil
}

func interpretIPConfigResponse(resp *cns.IPConfigResponse) (*net.IPNet, net.IP, error) {
	podCIDR := fmt.Sprintf(
		"%s/%s",
		resp.PodIpInfo.PodIPConfig.IPAddress,
		resp.PodIpInfo.NetworkContainerPrimaryIPConfig.IPSubnet.PrefixLength,
	)
	_, podIPNet, err := net.ParseCIDR(podCIDR)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "CNS returned invalid pod CIDR %q", podCIDR)
	}

	ncGatewayIPAddress := resp.PodIpInfo.NetworkContainerPrimaryIPConfig.GatewayIPAddress
	gwIP := net.ParseIP(ncGatewayIPAddress)
	if gwIP == nil {
		return nil, nil, fmt.Errorf("CNS returned an invalid gateway address: %s", ncGatewayIPAddress)
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
