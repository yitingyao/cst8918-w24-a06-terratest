package test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

var subscriptionID = "d0508ffd-65b4-44c3-ab85-331a0c6b77df"

func TestAzureLinuxVMCreation(t *testing.T) {
    // Hard-code the Azure CLI path
    azureCliPath := `C:\Program Files\Microsoft SDKs\Azure\CLI2\wbin`
    currentPath := os.Getenv("PATH")
    newPath := azureCliPath + ";" + currentPath

    // Also set PATH in your environment so we can see it in logs
    os.Setenv("PATH", newPath)

    // Print the final PATH for debugging
    fmt.Println("Test environment PATH:", os.Getenv("PATH"))

    // Terratest config
    terraformOptions := &terraform.Options{
        TerraformDir: "../",
        Vars: map[string]interface{}{
            "labelPrefix": "yao00043",
        },
        EnvVars: map[string]string{
            "PATH": newPath, // <-- This ensures Terraform + azure module uses your custom PATH
        },
    }

    // Clean up at the end
    defer terraform.Destroy(t, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)

    // Gather outputs
    vmName := terraform.Output(t, terraformOptions, "vm_name")
    resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
    nicName := terraform.Output(t, terraformOptions, "nic_name")

    // 1. Confirm the VM exists
    assert.True(t, azure.VirtualMachineExists(t, vmName, resourceGroupName, subscriptionID),
        "Expected VM '%s' to exist in Resource Group '%s'.", vmName, resourceGroupName)

    // 2. Confirm the NIC exists
    assert.True(t, azure.NetworkInterfaceExists(t, nicName, resourceGroupName, subscriptionID),
        "Expected NIC '%s' to exist in Resource Group '%s'.", nicName, resourceGroupName)

    // 3. Confirm that NIC is attached to the VM
    vm := azure.GetVirtualMachine(t, vmName, resourceGroupName, subscriptionID)
    if vm.NetworkProfile == nil || vm.NetworkProfile.NetworkInterfaces == nil {
        t.Fatalf("VM '%s' has no NetworkInterfaces field!", vmName)
    }

    nicAttached := false
    for _, nicRef := range *vm.NetworkProfile.NetworkInterfaces {
        // nicRef.ID is something like:
        // "/subscriptions/<sub>/resourceGroups/myRG/providers/Microsoft.Network/networkInterfaces/myNicName"
        if strings.Contains(*nicRef.ID, nicName) {
            nicAttached = true
            break
        }
    }
    assert.True(t, nicAttached, "Expected NIC '%s' to be attached to VM '%s'.", nicName, vmName)

    // 4. Confirm the VM is running the correct Ubuntu version (checking the image reference)
    if vm.StorageProfile == nil || vm.StorageProfile.ImageReference == nil {
        t.Fatalf("VM '%s' is missing StorageProfile/ImageReference!", vmName)
    }
    imageRef := vm.StorageProfile.ImageReference

    // These values come from your Terraform:
    // publisher = "Canonical"
    // offer     = "0001-com-ubuntu-server-jammy"
    // sku       = "22_04-lts-gen2"
    // version   = "latest"

    assert.NotNil(t, imageRef.Publisher, "ImageReference publisher should not be nil.")
    assert.Equal(t, "Canonical", *imageRef.Publisher, "Unexpected publisher.")
    assert.NotNil(t, imageRef.Offer, "ImageReference offer should not be nil.")
    assert.Equal(t, "0001-com-ubuntu-server-jammy", *imageRef.Offer, "Unexpected offer.")
    assert.NotNil(t, imageRef.Sku, "ImageReference sku should not be nil.")
    assert.Equal(t, "22_04-lts-gen2", *imageRef.Sku, "Unexpected SKU.")
    // If you want to check version == "latest", you can:
    //   assert.Equal(t, "latest", *imageRef.Version, "Unexpected version.")
}
