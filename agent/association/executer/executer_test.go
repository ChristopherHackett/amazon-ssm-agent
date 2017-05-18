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

// Package executer allows execute Pending association and InProgress association
package executer

import (
	"fmt"
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/contracts"
	"github.com/stretchr/testify/assert"
)

func TestOutputBuilderWithMultiplePlugins(t *testing.T) {
	results := make(map[string]*contracts.PluginRuntimeStatus)

	results["pluginA"] = &contracts.PluginRuntimeStatus{
		Status: contracts.ResultStatusPassedAndReboot,
	}
	results["pluginB"] = &contracts.PluginRuntimeStatus{
		Status: contracts.ResultStatusSuccess,
	}
	results["pluginC"] = &contracts.PluginRuntimeStatus{
		Status: contracts.ResultStatusFailed,
	}
	results["pluginD"] = &contracts.PluginRuntimeStatus{
		Status: contracts.ResultStatusSkipped,
	}

	output, _ := buildOutput(results, 5)

	fmt.Println(output)
	assert.NotNil(t, output)
	assert.Equal(t, output, "4 out of 5 plugins processed, 2 success, 1 failed, 0 timedout, 1 skipped")
}

func TestOutputBuilderWithSinglePlugin(t *testing.T) {
	results := make(map[string]*contracts.PluginRuntimeStatus)

	results["pluginA"] = &contracts.PluginRuntimeStatus{
		Status: contracts.ResultStatusFailed,
	}

	output, _ := buildOutput(results, 1)

	fmt.Println(output)
	assert.NotNil(t, output)
	assert.Equal(t, output, "1 out of 1 plugin processed, 0 success, 1 failed, 0 timedout, 0 skipped")
}

func TestOutputBuilderWithSinglePluginWithSkippedStatus(t *testing.T) {
	results := make(map[string]*contracts.PluginRuntimeStatus)

	results["pluginA"] = &contracts.PluginRuntimeStatus{
		Status: contracts.ResultStatusSkipped,
	}

	output, _ := buildOutput(results, 1)

	fmt.Println(output)
	assert.NotNil(t, output)
	assert.Equal(t, output, "1 out of 1 plugin processed, 0 success, 0 failed, 0 timedout, 1 skipped")
}
