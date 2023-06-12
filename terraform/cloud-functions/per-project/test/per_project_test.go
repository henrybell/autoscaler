/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	scheduler "cloud.google.com/go/scheduler/apiv1beta1"
	schedulerpb "cloud.google.com/go/scheduler/apiv1beta1/schedulerpb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"

	logger "github.com/gruntwork-io/terratest/modules/logger"
	retry "github.com/gruntwork-io/terratest/modules/retry"
	terraform "github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"

	assert "github.com/stretchr/testify/assert"
)

const (
	preScalingProcessingUnits   = 100
	postScalingProcessingUnits  = 200
	spannerTestInstanceTfOutput = "spanner_test_instance_name"
	schedulerJobTfOutput        = "scheduler_job_id"
	projectId                   = "placeholder"
	region                      = "us-central1"
	zone                        = "us-central1-a"
)

func waitForSpannerProcessingUnits(t *testing.T, instanceAdmin *instance.InstanceAdminClient, instanceId string, targetProcessingUnits int32, retries int, sleepBetweenRetries time.Duration) {
	ctx := context.Background()
	status := fmt.Sprintf("Wait for instance to reach %d (PUs)...", targetProcessingUnits)
	message := retry.DoWithRetry(
		t,
		status,
		retries,
		sleepBetweenRetries,
		func() (string, error) {
			spannerInstanceReq := &instancepb.GetInstanceRequest{
				Name: instanceId,
			}
			spannerInstance, err := instanceAdmin.GetInstance(ctx, spannerInstanceReq)
			assert.Nil(t, err)
			assert.NotNil(t, spannerInstance)
			processingUnits := spannerInstance.GetProcessingUnits()
			if processingUnits != targetProcessingUnits {
				return "", fmt.Errorf("Currently %d PUs", processingUnits)
			}
			return "Spanner instance reached target PUs", nil
		},
	)
	logger.Log(t, message)
}

func TestPerProjectEndToEndDeployment(t *testing.T) {

	terraformDir := "../"

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := &terraform.Options{
			TerraformDir: terraformDir,
			Vars: map[string]interface{}{
				"project_id":             projectId,
				"region":                 region,
				"zone":                   zone,
				"terraform_spanner_test": "true",
				"terraform_spanner_test_processing_units": preScalingProcessingUnits,
			},
		}

		test_structure.SaveTerraformOptions(t, terraformDir, terraformOptions)
		terraform.Init(t, terraformOptions)
	})

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "import", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		terraformArgs := []string{"module.scheduler.google_app_engine_application.app", projectId}
		terraformArgsFormatted := append(terraform.FormatArgs(terraformOptions, "-input=false"), terraformArgs...)
		terraformCommand := append([]string{"import"}, terraformArgsFormatted...)
		terraform.RunTerraformCommand(t, terraformOptions, terraformCommand...)
	})

	test_structure.RunTestStage(t, "apply", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		terraform.ApplyAndIdempotent(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		ctx := context.Background()

		// Create Spanner Admin client
		spannerTestInstanceName := terraform.Output(t, terraformOptions, spannerTestInstanceTfOutput)
		instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, instanceAdmin)
		defer instanceAdmin.Close()

		// Create Scheduler client
		schedulerJobId := terraform.Output(t, terraformOptions, schedulerJobTfOutput)
		schedulerClient, err := scheduler.NewCloudSchedulerClient(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, schedulerClient)
		defer schedulerClient.Close()

		// Get Scheduler job
		schedulerJobReq := &schedulerpb.GetJobRequest{
			Name: schedulerJobId,
		}
		schedulerJob, err := schedulerClient.GetJob(ctx, schedulerJobReq)
		assert.Nil(t, err)
		assert.NotNil(t, schedulerJob)

		// Wait for Spanner to report initial processing units
		spannerTestInstanceId := fmt.Sprintf("projects/%s/instances/%s", projectId, spannerTestInstanceName)
		waitForSpannerProcessingUnits(t, instanceAdmin, spannerTestInstanceId, preScalingProcessingUnits, 10, time.Second*10)

		//fmt.Printf("%#v \n", schedulerJob)
	})
}
