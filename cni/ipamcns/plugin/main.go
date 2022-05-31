// Copyright 2022 Microsoft. All rights reserved.
// MIT License

package main

import (
	"fmt"
	"os"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cni/ipamcns"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/pkg/errors"
)

const name = "azure-cns-ipam"

// Version is populated by make during build.
var version string

// Entrypoint for the IPAM plugin that uses CNS.
func main() {
	if err := executePlugin(); err != nil {
		fmt.Printf("Error executing CNS IPAM plugin: %s\n", err)
		os.Exit(1)
	}
}

func executePlugin() error {
	var config common.PluginConfig
	config.Version = version

	logDirectory := "" // Sets the current location as log directory
	log.SetName(name)
	log.SetLevel(log.LevelInfo)
	if err := log.SetTargetLogDirectory(log.TargetLogfile, logDirectory); err != nil {
		return errors.Wrapf(err, "Failed to setup cni logging")
	}
	defer log.Close()

	ipamPlugin, err := ipamcns.NewPlugin(name, &config)
	if err != nil {
		return errors.Wrapf(err, "Failed to create CNS IPAM plugin")
	}

	if err := ipamPlugin.Start(&config); err != nil {
		return errors.Wrapf(err, "Failed to start CNS IPAM plugin")
	}

	defer ipamPlugin.Stop()

	if err := ipamPlugin.Execute(cni.PluginApi(ipamPlugin)); err != nil {
		return errors.Wrapf(err, "Failed to execute CNS IPAM plugin")
	}

	return nil
}
