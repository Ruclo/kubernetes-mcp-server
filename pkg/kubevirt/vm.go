package kubevirt

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// RunStrategy represents the run strategy for a VirtualMachine
type RunStrategy string

const (
	RunStrategyAlways RunStrategy = "Always"
	RunStrategyHalted RunStrategy = "Halted"
)

// GetVirtualMachine retrieves a VirtualMachine by namespace and name
func GetVirtualMachine(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(VirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetVMRunStrategy retrieves the current runStrategy from a VirtualMachine
// Returns the strategy, whether it was found, and any error
func GetVMRunStrategy(vm *unstructured.Unstructured) (RunStrategy, bool, error) {
	strategy, found, err := unstructured.NestedString(vm.Object, "spec", "runStrategy")
	if err != nil {
		return "", false, fmt.Errorf("failed to read runStrategy: %w", err)
	}

	return RunStrategy(strategy), found, nil
}

// SetVMRunStrategy sets the runStrategy on a VirtualMachine
func SetVMRunStrategy(vm *unstructured.Unstructured, strategy RunStrategy) error {
	return unstructured.SetNestedField(vm.Object, string(strategy), "spec", "runStrategy")
}

// UpdateVirtualMachine updates a VirtualMachine in the cluster
func UpdateVirtualMachine(ctx context.Context, client dynamic.Interface, vm *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return client.Resource(VirtualMachineGVR).
		Namespace(vm.GetNamespace()).
		Update(ctx, vm, metav1.UpdateOptions{})
}

// StartVM starts a VirtualMachine by updating its runStrategy to Always
// Returns the updated VM and true if the VM was started, false if it was already running
func StartVM(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) (*unstructured.Unstructured, bool, error) {
	// Get the current VirtualMachine
	vm, err := GetVirtualMachine(ctx, dynamicClient, namespace, name)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get VirtualMachine: %w", err)
	}

	currentStrategy, found, err := GetVMRunStrategy(vm)
	if err != nil {
		return nil, false, err
	}

	// Check if already running
	if found && currentStrategy == RunStrategyAlways {
		return vm, false, nil
	}

	// Update runStrategy to Always
	if err := SetVMRunStrategy(vm, RunStrategyAlways); err != nil {
		return nil, false, fmt.Errorf("failed to set runStrategy: %w", err)
	}

	// Update the VM in the cluster
	updatedVM, err := UpdateVirtualMachine(ctx, dynamicClient, vm)
	if err != nil {
		return nil, false, fmt.Errorf("failed to start VirtualMachine: %w", err)
	}

	return updatedVM, true, nil
}

// StopVM stops a VirtualMachine by updating its runStrategy to Halted
// Returns the updated VM and true if the VM was stopped, false if it was already stopped
func StopVM(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) (*unstructured.Unstructured, bool, error) {
	// Get the current VirtualMachine
	vm, err := GetVirtualMachine(ctx, dynamicClient, namespace, name)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get VirtualMachine: %w", err)
	}

	currentStrategy, found, err := GetVMRunStrategy(vm)
	if err != nil {
		return nil, false, err
	}

	// Check if already stopped
	if found && currentStrategy == RunStrategyHalted {
		return vm, false, nil
	}

	// Update runStrategy to Halted
	if err := SetVMRunStrategy(vm, RunStrategyHalted); err != nil {
		return nil, false, fmt.Errorf("failed to set runStrategy: %w", err)
	}

	// Update the VM in the cluster
	updatedVM, err := UpdateVirtualMachine(ctx, dynamicClient, vm)
	if err != nil {
		return nil, false, fmt.Errorf("failed to stop VirtualMachine: %w", err)
	}

	return updatedVM, true, nil
}

// CloneVM creates a VirtualMachineClone CR to clone a source VM to a target VM
func CloneVM(ctx context.Context, dynamicClient dynamic.Interface, namespace, sourceName, targetName string) (*unstructured.Unstructured, error) {
	clone := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "clone.kubevirt.io/v1beta1",
			"kind":       "VirtualMachineClone",
			"metadata": map[string]any{
				"namespace":    namespace,
				"generateName": sourceName + "-clone-",
			},
			"spec": map[string]any{
				"source": map[string]any{
					"apiGroup": "kubevirt.io",
					"kind":     "VirtualMachine",
					"name":     sourceName,
				},
				"target": map[string]any{
					"apiGroup": "kubevirt.io",
					"kind":     "VirtualMachine",
					"name":     targetName,
				},
			},
		},
	}

	result, err := dynamicClient.Resource(VirtualMachineCloneGVR).Namespace(namespace).Create(ctx, clone, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create VirtualMachineClone: %w", err)
	}

	return result, nil
}

// DiskType represents the type of disk to attach to a VirtualMachine
type DiskType string

const (
	DiskTypeDisk DiskType = "disk"
	DiskTypeLun  DiskType = "lun"
)

// ValidateDiskType validates the disk type and bus combination, returning the resolved bus.
// If bus is empty, it returns the default bus for the given disk type.
func ValidateDiskType(diskType DiskType, bus string) (string, error) {
	switch diskType {
	case DiskTypeDisk:
		if bus == "" {
			return "virtio", nil
		}
		if bus != "virtio" && bus != "scsi" {
			return "", fmt.Errorf("invalid bus '%s' for disk type 'disk': must be 'virtio' or 'scsi'", bus)
		}
		return bus, nil
	case DiskTypeLun:
		if bus == "" {
			return "scsi", nil
		}
		if bus != "scsi" {
			return "", fmt.Errorf("invalid bus '%s' for disk type 'lun': must be 'scsi'", bus)
		}
		return bus, nil
	default:
		return "", fmt.Errorf("invalid disk type '%s': must be 'disk' or 'lun'", diskType)
	}
}

// HotplugPVC adds a hotpluggable PersistentVolumeClaim volume and disk to a VirtualMachine
func HotplugPVC(ctx context.Context, dynamicClient dynamic.Interface, namespace, vmName, diskName, claimName string, diskType DiskType, bus string) (*unstructured.Unstructured, error) {
	volume := map[string]any{
		"name": diskName,
		"persistentVolumeClaim": map[string]any{
			"claimName":    claimName,
			"hotpluggable": true,
		},
	}
	return hotplugVolume(ctx, dynamicClient, namespace, vmName, diskName, diskType, bus, volume)
}

// HotplugDataVolume adds a hotpluggable DataVolume volume and disk to a VirtualMachine
func HotplugDataVolume(ctx context.Context, dynamicClient dynamic.Interface, namespace, vmName, diskName, dataVolumeName string, diskType DiskType, bus string) (*unstructured.Unstructured, error) {
	volume := map[string]any{
		"name": diskName,
		"dataVolume": map[string]any{
			"name":         dataVolumeName,
			"hotpluggable": true,
		},
	}
	return hotplugVolume(ctx, dynamicClient, namespace, vmName, diskName, diskType, bus, volume)
}

func hotplugVolume(ctx context.Context, dynamicClient dynamic.Interface, namespace, vmName, diskName string, diskType DiskType, bus string, volume map[string]any) (*unstructured.Unstructured, error) {
	vm, err := GetVirtualMachine(ctx, dynamicClient, namespace, vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualMachine: %w", err)
	}

	// Get existing disks or initialize empty slice
	disks, _, _ := unstructured.NestedSlice(vm.Object, "spec", "template", "spec", "domain", "devices", "disks")

	// Check for duplicate disk name
	for _, d := range disks {
		if disk, ok := d.(map[string]any); ok {
			if name, _ := disk["name"].(string); name == diskName {
				return nil, fmt.Errorf("disk '%s' already exists in VirtualMachine '%s'", diskName, vmName)
			}
		}
	}

	// Validate disk type and bus combination
	resolvedBus, err := ValidateDiskType(diskType, bus)
	if err != nil {
		return nil, err
	}

	// Add new disk
	newDisk := map[string]any{
		"name": diskName,
		string(diskType): map[string]any{
			"bus": resolvedBus,
		},
	}
	disks = append(disks, newDisk)
	if err := unstructured.SetNestedSlice(vm.Object, disks, "spec", "template", "spec", "domain", "devices", "disks"); err != nil {
		return nil, fmt.Errorf("failed to set disks: %w", err)
	}

	// Get existing volumes or initialize empty slice
	volumes, _, _ := unstructured.NestedSlice(vm.Object, "spec", "template", "spec", "volumes")

	// Add new volume
	volumes = append(volumes, volume)
	if err := unstructured.SetNestedSlice(vm.Object, volumes, "spec", "template", "spec", "volumes"); err != nil {
		return nil, fmt.Errorf("failed to set volumes: %w", err)
	}

	updatedVM, err := UpdateVirtualMachine(ctx, dynamicClient, vm)
	if err != nil {
		return nil, fmt.Errorf("failed to update VirtualMachine: %w", err)
	}

	return updatedVM, nil
}

// RestartVM restarts a VirtualMachine by temporarily setting runStrategy to Halted then back to Always
func RestartVM(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	// Get the current VirtualMachine
	vm, err := GetVirtualMachine(ctx, dynamicClient, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualMachine: %w", err)
	}

	// Stop the VM first
	if err := SetVMRunStrategy(vm, RunStrategyHalted); err != nil {
		return nil, fmt.Errorf("failed to set runStrategy to Halted: %w", err)
	}

	vm, err = UpdateVirtualMachine(ctx, dynamicClient, vm)
	if err != nil {
		return nil, fmt.Errorf("failed to stop VirtualMachine: %w", err)
	}

	// Start the VM again
	if err := SetVMRunStrategy(vm, RunStrategyAlways); err != nil {
		return nil, fmt.Errorf("failed to set runStrategy to Always: %w", err)
	}

	updatedVM, err := UpdateVirtualMachine(ctx, dynamicClient, vm)
	if err != nil {
		return nil, fmt.Errorf("failed to start VirtualMachine: %w", err)
	}

	return updatedVM, nil
}
