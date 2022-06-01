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

// plugin is an IPAM plugin that uses CNS to manage IP addresses.
type plugin struct {
	*cni.Plugin
	cnsClient *cnsclient.Client
}

// NewPlugin constructs a new IPAM plugin.
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

// Start initializes the plugin.
func (p *plugin) Start(config *common.PluginConfig) error {
	if err := p.Initialize(config); err != nil {
		return errors.Wrapf(err, "Initialize base plugin")
	}
	log.Printf("[cni-ipam] Plugin started")
	return nil
}

// Stop uninitializes the plugin.
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
	req, err := cnsIPConfigRequest(args)
	if err != nil {
		return err
	}

	// cnsClient sets a request timeout.
	ctx := context.TODO()
	resp, err := p.cnsClient.RequestIPAddress(ctx, req)
	if err != nil {
		log.Printf("Failed to get IP address from CNS with error %s, response: %v", err, resp)
		return errors.Wrapf(err, "CNS client RequestIPAddress")
	}

	podIPNet, gwIP, err := interpretRequestIPResp(resp)
	if err != nil {
		return errors.Wrapf(err, "Could not interpret CNS IPConfigResponse")
	}

	nwCfg, err := cni.ParseNetworkConfig(args.StdinData)
	if err != nil {
		log.Printf("Could not parse CNI network config: %s\n", err)
		return errors.Wrapf(err, "Could not parse CNI network config")
	}

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

	versionedCniResult, err := cniResult.GetAsVersion(nwCfg.CNIVersion)
	if err != nil {
		log.Printf("Could not interpret CNI result as version %s: %s", nwCfg.CNIVersion, err)
		return errors.Wrapf(err, "Could not interpret CNI result as version %s", nwCfg.CNIVersion)
	}

	versionedCniResult.Print()
	return nil
}

// Get handles CNI Get commands.
func (p *plugin) Get(args *cniSkel.CmdArgs) error {
	// Not used for delegated IPAM plugins.
	return nil
}

// Delete handles CNI delete commands.
func (p *plugin) Delete(args *cniSkel.CmdArgs) error {
	req, err := cnsIPConfigRequest(args)
	if err != nil {
		return err
	}

	// cnsClient sets a request timeout.
	// If the IP address has already been released, CNS will do nothing and return success.
	ctx := context.TODO()
	if err := p.cnsClient.ReleaseIPAddress(ctx, req); err != nil {
		return p.RetriableError(fmt.Errorf("failed to release address: %w", err))
	}

	return nil
}

// Update handles CNI update command.
func (p *plugin) Update(args *cniSkel.CmdArgs) error {
	// This isn't part of the CNI spec, so do nothing.
	return nil
}

func cnsIPConfigRequest(args *cniSkel.CmdArgs) (cns.IPConfigRequest, error) {
	podCfg, err := cni.ParseCniArgs(args.Args)
	if err != nil {
		return cns.IPConfigRequest{}, errors.Wrapf(err, "Could not parse CNI args")
	}

	podInfo := cns.KubernetesPodInfo{
		PodName:      string(podCfg.K8S_POD_NAME),
		PodNamespace: string(podCfg.K8S_POD_NAMESPACE),
	}

	orchestratorContext, err := json.Marshal(podInfo)
	if err != nil {
		return cns.IPConfigRequest{}, errors.Wrapf(err, "Could not marshal podInfo to JSON")
	}

	req := cns.IPConfigRequest{
		PodInterfaceID:      cninetwork.GetEndpointID(args),
		InfraContainerID:    args.ContainerID,
		OrchestratorContext: orchestratorContext,
	}

	return req, nil
}

func interpretRequestIPResp(resp *cns.IPConfigResponse) (*net.IPNet, net.IP, error) {
	podCIDR := fmt.Sprintf(
		"%s/%d",
		resp.PodIpInfo.PodIPConfig.IPAddress,
		resp.PodIpInfo.NetworkContainerPrimaryIPConfig.IPSubnet.PrefixLength,
	)
	podIP, podIPNet, err := net.ParseCIDR(podCIDR)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "CNS returned invalid pod CIDR %q", podCIDR)
	}

	resultIPNet := &net.IPNet{
		IP:   podIP,
		Mask: podIPNet.Mask,
	}

	ncGatewayIPAddress := resp.PodIpInfo.NetworkContainerPrimaryIPConfig.GatewayIPAddress
	gwIP := net.ParseIP(ncGatewayIPAddress)
	if gwIP == nil {
		return nil, nil, fmt.Errorf("CNS returned an invalid gateway address: %s", ncGatewayIPAddress)
	}

	return resultIPNet, gwIP, nil
}
