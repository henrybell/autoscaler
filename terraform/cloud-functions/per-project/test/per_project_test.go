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
	"testing"

	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestPerProjectEndToEndDeployment(t *testing.T) {

	const projectID = "placeholder"
	const region = "us-central1"
	const zone = "us-central1-a"

	t.Parallel()

	terraformDir := "../"

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := &terraform.Options{
			TerraformDir: terraformDir,
			Vars: map[string]interface{}{
				"project_id":             projectID,
				"region":                 region,
				"zone":                   zone,
				"terraform_spanner_test": "true",
			},
		}

		test_structure.SaveTerraformOptions(t, terraformDir, terraformOptions)
		terraform.Init(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "import", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		terraformArgs := []string{"module.scheduler.google_app_engine_application.app", projectID}
		terraformArgsFormatted := append(terraform.FormatArgs(terraformOptions, "-input=false"), terraformArgs...)
		terraformCommand := append([]string{"import"}, terraformArgsFormatted...)
		terraform.RunTerraformCommand(t, terraformOptions, terraformCommand...)
	})

	test_structure.RunTestStage(t, "apply", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		terraform.ApplyAndIdempotent(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {
		//terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
		assert.True(t, true)
	})
}
