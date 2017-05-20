// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package plugin contains general interfaces and types relevant to plugins.
// It also provides the methods for registering plugins.
package plugin

import (
	"sync"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/framework/runpluginutil"
	"github.com/aws/amazon-ssm-agent/agent/plugins/configurecontainers"
	"github.com/aws/amazon-ssm-agent/agent/plugins/configurepackage"
	"github.com/aws/amazon-ssm-agent/agent/plugins/dockercontainer"
	"github.com/aws/amazon-ssm-agent/agent/plugins/inventory"
	"github.com/aws/amazon-ssm-agent/agent/plugins/lrpminvoker"
	"github.com/aws/amazon-ssm-agent/agent/plugins/pluginutil"
	"github.com/aws/amazon-ssm-agent/agent/plugins/refreshassociation"
	"github.com/aws/amazon-ssm-agent/agent/plugins/runscript"
	"github.com/aws/amazon-ssm-agent/agent/plugins/updatessmagent"
)

// allPlugins is the list of all known plugins.
// This allows us to differentiate between the case where a document asks for a plugin that exists but isn't supported on this platform
// and the case where a plugin name isn't known at all to this version of the agent (and the user should probably upgrade their agent)
var allPlugins = map[string]struct{}{
	appconfig.PluginNameAwsAgentUpdate:         {},
	appconfig.PluginNameAwsApplications:        {},
	appconfig.PluginNameAwsConfigureDaemon:     {},
	appconfig.PluginNameAwsConfigurePackage:    {},
	appconfig.PluginNameAwsPowerShellModule:    {},
	appconfig.PluginNameAwsRunPowerShellScript: {},
	appconfig.PluginNameAwsRunShellScript:      {},
	appconfig.PluginNameAwsSoftwareInventory:   {},
	appconfig.PluginNameCloudWatch:             {},
	appconfig.PluginNameConfigureDocker:        {},
	appconfig.PluginNameDockerContainer:        {},
	appconfig.PluginNameDomainJoin:             {},
	appconfig.PluginEC2ConfigUpdate:            {},
	appconfig.PluginNameRefreshAssociation:     {},
}

// registeredExecuters stores the registered plugins.
var registeredExecuters, registeredLongRunningPlugins *runpluginutil.PluginRegistry

// RegisteredWorkerPlugins returns all registered core modules.
func RegisteredWorkerPlugins(context context.T) runpluginutil.PluginRegistry {
	if !isLoaded() {
		cache(loadWorkerPlugins(context), loadLongRunningPlugins(context))
	}
	return getCachedWorkerPlugins()
}

// LongRunningPlugins returns a map of long running plugins and their respective handlers
func RegisteredLongRunningPlugins(context context.T) runpluginutil.PluginRegistry {
	if !isLoaded() {
		cache(loadWorkerPlugins(context), loadLongRunningPlugins(context))
	}
	return getCachedLongRunningPlugins()
}

var lock sync.RWMutex

func isLoaded() bool {
	lock.RLock()
	defer lock.RUnlock()
	return registeredExecuters != nil
}

func cache(workerPlugins, longRunningPlugins runpluginutil.PluginRegistry) {
	lock.Lock()
	defer lock.Unlock()
	registeredExecuters = &workerPlugins
	registeredLongRunningPlugins = &longRunningPlugins
}

func getCachedWorkerPlugins() runpluginutil.PluginRegistry {
	lock.RLock()
	defer lock.RUnlock()
	return *registeredExecuters
}

func getCachedLongRunningPlugins() runpluginutil.PluginRegistry {
	lock.RLock()
	defer lock.RUnlock()
	return *registeredLongRunningPlugins
}

// loadLongRunningPlugins loads all long running plugins
func loadLongRunningPlugins(context context.T) runpluginutil.PluginRegistry {
	log := context.Log()
	var longRunningPlugins = runpluginutil.PluginRegistry{}

	//Long running plugins are handled by lrpm. lrpminvoker is a worker plugin that can communicate with lrpm.
	//that's why all long running plugins are first handled by lrpminvoker - which then hands off the work to lrpm.

	//NOTE: register all long running plugins here (one instance of lrpminvoker per long running plugin)
	if handler, err := lrpminvoker.NewPlugin(pluginutil.DefaultPluginConfig(), appconfig.PluginNameCloudWatch); err != nil {
		log.Errorf("Failed to load lrpminvoker that will handle all long running plugins - %v", err)
	} else {
		//registering handler for aws:cloudWatch plugin
		longRunningPlugins[appconfig.PluginNameCloudWatch] = handler
	}

	return longRunningPlugins
}

// loadWorkerPlugins loads all plugins
func loadWorkerPlugins(context context.T) runpluginutil.PluginRegistry {
	var workerPlugins = runpluginutil.PluginRegistry{}

	for key, value := range loadPlatformIndependentPlugins(context) {
		workerPlugins[key] = value
	}

	for key, value := range loadPlatformDependentPlugins(context) {
		workerPlugins[key] = value
	}

	return workerPlugins
}

// loadPlatformIndependentPlugins registers plugins common to all platforms
func loadPlatformIndependentPlugins(context context.T) runpluginutil.PluginRegistry {
	log := context.Log()
	var workerPlugins = runpluginutil.PluginRegistry{}

	inventoryPluginName := inventory.Name()
	if inventoryPlugin, err := inventory.NewPlugin(context, pluginutil.DefaultPluginConfig()); err != nil {
		log.Errorf("failed to create plugin %s %v", inventoryPluginName, err)
	} else {
		workerPlugins[inventoryPluginName] = inventoryPlugin
	}

	// registering aws:runPowerShellScript plugin
	powershellPlugin, err := runscript.NewRunPowerShellPlugin(pluginutil.DefaultPluginConfig())
	powershellPluginName := powershellPlugin.Name
	if err != nil {
		log.Errorf("failed to create plugin %s %v", powershellPluginName, err)
	} else {
		workerPlugins[powershellPluginName] = powershellPlugin
	}

	// registering aws:updateSsmAgent plugin
	updateAgentPluginName := updatessmagent.Name()
	updateAgentPlugin, err := updatessmagent.NewPlugin(updatessmagent.GetUpdatePluginConfig(context))
	if err != nil {
		log.Errorf("failed to create plugin %s %v", updateAgentPluginName, err)
	} else {
		workerPlugins[updateAgentPluginName] = updateAgentPlugin
	}

	// registering aws:configureContainers plugin
	configureContainersPluginName := configurecontainers.Name()
	configureContainersPlugin, err := configurecontainers.NewPlugin(pluginutil.DefaultPluginConfig())
	if err != nil {
		log.Errorf("failed to create plugin %s %v", configureContainersPluginName, err)
	} else {
		workerPlugins[configureContainersPluginName] = configureContainersPlugin
	}

	// registering aws:runDockerAction plugin
	runDockerPluginName := dockercontainer.Name()
	runDockerPlugin, err := dockercontainer.NewPlugin(pluginutil.DefaultPluginConfig())
	if err != nil {
		log.Errorf("failed to create plugin %s %v", runDockerPluginName, err)
	} else {
		workerPlugins[runDockerPluginName] = runDockerPlugin
	}

	// registering aws:refreshAssociation plugin
	refreshAssociationPluginName := refreshassociation.Name()
	refreshAssociationPlugin, err := refreshassociation.NewPlugin(pluginutil.DefaultPluginConfig())
	if err != nil {
		log.Errorf("failed to create plugin %s %v", refreshAssociationPluginName, err)
	} else {
		workerPlugins[refreshAssociationPluginName] = refreshAssociationPlugin
	}

	// registering aws:configurePackage
	configurePackagePluginName := configurepackage.Name()
	configurePackagePlugin, err := configurepackage.NewPlugin(pluginutil.DefaultPluginConfig())
	if err != nil {
		log.Errorf("failed to create plugin %s %v", configurePackagePluginName, err)
	} else {
		workerPlugins[configurePackagePluginName] = configurePackagePlugin
	}

	return workerPlugins
}
