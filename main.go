package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	dacranepdk "github.com/SIOS-Technology-Inc/dacrane-pdk"
)

func main() {
	dacranepdk.ExecPluginJob(dacranepdk.Plugin{
		Config:    dacranepdk.NewDefaultPluginConfig(),
		Resources: buildTerraformResource,
		Data:      buildTerraformData,
	})
}

func buildTerraformResource(name string) (dacranepdk.Resource, bool) {
	providerName := strings.Split(name, "_")[0]
	resourceType := name

	var Create = func(parameter any, meta dacranepdk.PluginMeta) (any, error) {
		parameters := parameter.(map[string]any)

		resourceName := "main"
		argument := parameters["resource"].(map[string]any)
		provider := parameters["provider"].(map[string]any)

		mainTf := map[string]any{
			"provider": map[string]any{
				providerName: provider,
			},
			"resource": map[string]any{
				resourceType: map[string]any{
					resourceName: argument,
				},
			},
		}

		byteData, err := json.MarshalIndent(mainTf, "", "  ")
		if err != nil {
			meta.Log("Error marshalling to JSON")
			return nil, err
		}

		// Write Terraform File (JSON)
		filename := "main.tf.json"
		dir := meta.CustomStateDir
		filePath := filepath.Join(dir, filename)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("error creating directory: %v", err)
			}
		}

		if err := os.WriteFile(filePath, byteData, 0644); err != nil {
			return nil, fmt.Errorf("error writing JSON file: %v", err)
		}

		if err := ApplyTerraform(filePath, meta); err != nil {
			return nil, err
		}

		// Get Terraform State
		bytes, err := os.ReadFile(dir + "/terraform.tfstate")
		if err != nil {
			return nil, err
		}

		var state map[string]any
		err = json.Unmarshal(bytes, &state)
		if err != nil {
			return nil, err
		}

		resource := Find(state["resources"].([]any), func(r any) bool {
			return r.(map[string]any)["mode"] == "managed" &&
				r.(map[string]any)["type"] == resourceType &&
				r.(map[string]any)["name"] == resourceName
		})

		instances := resource.(map[string]any)["instances"]
		instance := instances.([]any)[0]
		attributes := instance.(map[string]any)["attributes"]
		return attributes.(map[string]any), nil
	}

	return dacranepdk.Resource{
		Create: Create,
		Update: func(current, _ any, meta dacranepdk.PluginMeta) (any, error) {
			return Create(current, meta)
		},
		Delete: func(_ any, meta dacranepdk.PluginMeta) error {
			dir := meta.CustomStateDir
			// terraform destroy
			cmd := exec.Command("terraform", "destroy", "-auto-approve")
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to execute terraform destroy: %v, output: %s", err, output)
			}

			err = os.RemoveAll(dir)
			if err != nil {
				return err
			}

			meta.Log("Terraform destroy executed successfully.")
			return nil
		},
	}, true
}

func buildTerraformData(name string) (dacranepdk.Data, bool) {
	providerName := strings.Split(name, "_")[0]
	resourceType := name

	return dacranepdk.Data{
		Get: func(parameter any, meta dacranepdk.PluginMeta) (any, error) {
			parameters := parameter.(map[string]any)
			resourceName := "main"
			argument := parameters["data"].(map[string]any)
			provider := parameters["provider"].(map[string]any)

			mainTf := map[string]any{
				"provider": map[string]any{
					providerName: provider,
				},
				"data": map[string]any{
					resourceType: map[string]any{
						resourceName: argument,
					},
				},
			}

			byteData, err := json.MarshalIndent(mainTf, "", "  ")
			if err != nil {
				meta.Log("Error marshalling to JSON")
				return nil, err
			}

			// Write Terraform File (JSON)
			filename := "main.tf.json"
			dir := meta.CustomStateDir
			filePath := filepath.Join(dir, filename)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, fmt.Errorf("error creating directory: %v", err)
				}
			}

			if err := os.WriteFile(filePath, byteData, 0644); err != nil {
				return nil, fmt.Errorf("error writing JSON file: %v", err)
			}

			if err := ApplyTerraform(filePath, meta); err != nil {
				return nil, err
			}

			// Get Terraform State
			bytes, err := os.ReadFile(dir + "/terraform.tfstate")
			if err != nil {
				return nil, err
			}

			var state map[string]any
			err = json.Unmarshal(bytes, &state)
			if err != nil {
				return nil, err
			}

			resource := Find(state["resources"].([]any), func(r any) bool {
				return r.(map[string]any)["mode"] == "managed" &&
					r.(map[string]any)["type"] == resourceType &&
					r.(map[string]any)["name"] == resourceName
			})

			instances := resource.(map[string]any)["instances"]
			instance := instances.([]any)[0]
			attributes := instance.(map[string]any)["attributes"]
			return attributes.(map[string]any), nil
		},
	}, true
}

func ApplyTerraform(filePath string, meta dacranepdk.PluginMeta) error {
	// Terraform init
	dir := filepath.Dir(filePath)

	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = dir
	if output, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run terraform init: %s, %s", err, output)
	}

	// Terraform apply
	applyCmd := exec.Command("terraform", "apply", "-auto-approve")
	applyCmd.Dir = dir
	if output, err := applyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run terraform apply: %s, %s", err, output)
	}

	meta.Log("Terraform apply complete")
	return nil
}

func Find[T any](array []T, f func(T) bool) (result T) {
	for _, value := range array {
		if f(value) {
			return value
		}
	}
	return
}
