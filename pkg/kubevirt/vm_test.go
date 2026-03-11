package kubevirt

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

// createTestVM creates a test VirtualMachine with the given name, namespace, and runStrategy
func createTestVM(name, namespace string, runStrategy RunStrategy) *unstructured.Unstructured {
	vm := &unstructured.Unstructured{}
	vm.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kubevirt.io/v1",
		"kind":       "VirtualMachine",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"runStrategy": string(runStrategy),
		},
	})
	return vm
}

func TestStartVM(t *testing.T) {
	tests := []struct {
		name          string
		initialVM     *unstructured.Unstructured
		wantStarted   bool
		wantError     bool
		errorContains string
	}{
		{
			name:        "Start VM that is Halted",
			initialVM:   createTestVM("test-vm", "default", RunStrategyHalted),
			wantStarted: true,
			wantError:   false,
		},
		{
			name:        "Start VM that is already running (Always)",
			initialVM:   createTestVM("test-vm", "default", RunStrategyAlways),
			wantStarted: false,
			wantError:   false,
		},
		{
			name: "Start VM without runStrategy",
			initialVM: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kubevirt.io/v1",
					"kind":       "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":      "test-vm",
						"namespace": "default",
					},
					"spec": map[string]interface{}{},
				},
			},
			wantStarted: true,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewSimpleDynamicClient(scheme, tt.initialVM)
			ctx := context.Background()

			vm, wasStarted, err := StartVM(ctx, client, tt.initialVM.GetNamespace(), tt.initialVM.GetName())

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error = %v, want to contain %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if vm == nil {
				t.Errorf("Expected non-nil VM, got nil")
				return
			}

			if wasStarted != tt.wantStarted {
				t.Errorf("wasStarted = %v, want %v", wasStarted, tt.wantStarted)
			}

			// Verify the VM's runStrategy is Always
			strategy, found, err := GetVMRunStrategy(vm)
			if err != nil {
				t.Errorf("Failed to get runStrategy: %v", err)
				return
			}
			if !found {
				t.Errorf("runStrategy not found")
				return
			}
			if strategy != RunStrategyAlways {
				t.Errorf("Strategy = %q, want %q", strategy, RunStrategyAlways)
			}
		})
	}
}

func TestStartVMNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	ctx := context.Background()

	_, _, err := StartVM(ctx, client, "default", "non-existent-vm")
	if err == nil {
		t.Errorf("Expected error for non-existent VM, got nil")
		return
	}
	if !strings.Contains(err.Error(), "failed to get VirtualMachine") {
		t.Errorf("Error = %v, want to contain 'failed to get VirtualMachine'", err)
	}
}

func TestStopVM(t *testing.T) {
	tests := []struct {
		name          string
		initialVM     *unstructured.Unstructured
		wantStopped   bool
		wantError     bool
		errorContains string
	}{
		{
			name:        "Stop VM that is running (Always)",
			initialVM:   createTestVM("test-vm", "default", RunStrategyAlways),
			wantStopped: true,
			wantError:   false,
		},
		{
			name:        "Stop VM that is already stopped (Halted)",
			initialVM:   createTestVM("test-vm", "default", RunStrategyHalted),
			wantStopped: false,
			wantError:   false,
		},
		{
			name: "Stop VM without runStrategy",
			initialVM: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kubevirt.io/v1",
					"kind":       "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":      "test-vm",
						"namespace": "default",
					},
					"spec": map[string]interface{}{},
				},
			},
			wantStopped: true,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewSimpleDynamicClient(scheme, tt.initialVM)
			ctx := context.Background()

			vm, wasStopped, err := StopVM(ctx, client, tt.initialVM.GetNamespace(), tt.initialVM.GetName())

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error = %v, want to contain %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if vm == nil {
				t.Errorf("Expected non-nil VM, got nil")
				return
			}

			if wasStopped != tt.wantStopped {
				t.Errorf("wasStopped = %v, want %v", wasStopped, tt.wantStopped)
			}

			// Verify the VM's runStrategy is Halted
			strategy, found, err := GetVMRunStrategy(vm)
			if err != nil {
				t.Errorf("Failed to get runStrategy: %v", err)
				return
			}
			if !found {
				t.Errorf("runStrategy not found")
				return
			}
			if strategy != RunStrategyHalted {
				t.Errorf("Strategy = %q, want %q", strategy, RunStrategyHalted)
			}
		})
	}
}

func TestStopVMNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	ctx := context.Background()

	_, _, err := StopVM(ctx, client, "default", "non-existent-vm")
	if err == nil {
		t.Errorf("Expected error for non-existent VM, got nil")
		return
	}
	if !strings.Contains(err.Error(), "failed to get VirtualMachine") {
		t.Errorf("Error = %v, want to contain 'failed to get VirtualMachine'", err)
	}
}

func TestRestartVM(t *testing.T) {
	tests := []struct {
		name          string
		initialVM     *unstructured.Unstructured
		wantError     bool
		errorContains string
	}{
		{
			name:      "Restart VM that is running (Always)",
			initialVM: createTestVM("test-vm", "default", RunStrategyAlways),
			wantError: false,
		},
		{
			name:      "Restart VM that is stopped (Halted)",
			initialVM: createTestVM("test-vm", "default", RunStrategyHalted),
			wantError: false,
		},
		{
			name: "Restart VM without runStrategy",
			initialVM: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kubevirt.io/v1",
					"kind":       "VirtualMachine",
					"metadata": map[string]interface{}{
						"name":      "test-vm",
						"namespace": "default",
					},
					"spec": map[string]interface{}{},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewSimpleDynamicClient(scheme, tt.initialVM)
			ctx := context.Background()

			vm, err := RestartVM(ctx, client, tt.initialVM.GetNamespace(), tt.initialVM.GetName())

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error = %v, want to contain %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if vm == nil {
				t.Errorf("Expected non-nil VM, got nil")
				return
			}

			// Verify the VM's runStrategy is Always (after restart)
			strategy, found, err := GetVMRunStrategy(vm)
			if err != nil {
				t.Errorf("Failed to get runStrategy: %v", err)
				return
			}
			if !found {
				t.Errorf("runStrategy not found")
				return
			}
			if strategy != RunStrategyAlways {
				t.Errorf("Strategy = %q, want %q after restart", strategy, RunStrategyAlways)
			}
		})
	}
}

func TestRestartVMNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	ctx := context.Background()

	_, err := RestartVM(ctx, client, "default", "non-existent-vm")
	if err == nil {
		t.Errorf("Expected error for non-existent VM, got nil")
		return
	}
	if !strings.Contains(err.Error(), "failed to get VirtualMachine") {
		t.Errorf("Error = %v, want to contain 'failed to get VirtualMachine'", err)
	}
}

func withDisk(vm *unstructured.Unstructured) *unstructured.Unstructured {
	vm.Object["spec"].(map[string]interface{})["template"] = map[string]interface{}{
		"spec": map[string]interface{}{
			"domain": map[string]interface{}{
				"devices": map[string]interface{}{
					"disks": []interface{}{
						map[string]interface{}{"name": "rootdisk", "disk": map[string]interface{}{"bus": "virtio"}},
					},
				},
			},
			"volumes": []interface{}{
				map[string]interface{}{"name": "rootdisk", "containerDisk": map[string]interface{}{"image": "fedora:latest"}},
			},
		},
	}
	return vm
}

func TestHotplugPVC(t *testing.T) {
	t.Run("adds PVC with hotpluggable true", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), createTestVM("test-vm", "default", RunStrategyAlways))

		result, err := HotplugPVC(context.Background(), client, "default", "test-vm", "hp-disk", "my-pvc", DiskTypeDisk, "virtio")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		volumes, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "volumes")
		if len(volumes) != 1 {
			t.Fatalf("Expected 1 volume, got %d", len(volumes))
		}
		pvc := volumes[0].(map[string]interface{})["persistentVolumeClaim"].(map[string]interface{})
		if pvc["claimName"] != "my-pvc" {
			t.Errorf("Expected claimName 'my-pvc', got %v", pvc["claimName"])
		}
		if pvc["hotpluggable"] != true {
			t.Errorf("Expected hotpluggable true, got %v", pvc["hotpluggable"])
		}
	})

	t.Run("appends to existing disks", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), withDisk(createTestVM("test-vm", "default", RunStrategyAlways)))

		result, err := HotplugPVC(context.Background(), client, "default", "test-vm", "hp-disk", "my-pvc", DiskTypeDisk, "scsi")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		disks, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "domain", "devices", "disks")
		if len(disks) != 2 {
			t.Fatalf("Expected 2 disks, got %d", len(disks))
		}
		if disks[1].(map[string]interface{})["disk"].(map[string]interface{})["bus"] != "scsi" {
			t.Error("Expected bus 'scsi'")
		}
	})

	t.Run("duplicate disk name", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), withDisk(createTestVM("test-vm", "default", RunStrategyAlways)))

		_, err := HotplugPVC(context.Background(), client, "default", "test-vm", "rootdisk", "my-pvc", DiskTypeDisk, "virtio")
		if err == nil {
			t.Fatal("Expected error for duplicate disk name")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("VM not found", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme())
		_, err := HotplugPVC(context.Background(), client, "default", "missing", "d1", "pvc1", DiskTypeDisk, "virtio")
		if err == nil {
			t.Fatal("Expected error for non-existent VM")
		}
		if !strings.Contains(err.Error(), "failed to get VirtualMachine") {
			t.Errorf("Expected 'failed to get VirtualMachine' error, got: %v", err)
		}
	})

	t.Run("lun disk type uses scsi bus", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), createTestVM("test-vm", "default", RunStrategyAlways))

		result, err := HotplugPVC(context.Background(), client, "default", "test-vm", "hp-lun", "my-pvc", DiskTypeLun, "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		disks, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "domain", "devices", "disks")
		if len(disks) != 1 {
			t.Fatalf("Expected 1 disk, got %d", len(disks))
		}
		disk := disks[0].(map[string]interface{})
		lun := disk["lun"].(map[string]interface{})
		if lun["bus"] != "scsi" {
			t.Errorf("Expected bus 'scsi' for lun, got %v", lun["bus"])
		}
	})

	t.Run("invalid bus for lun disk type", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), createTestVM("test-vm", "default", RunStrategyAlways))

		_, err := HotplugPVC(context.Background(), client, "default", "test-vm", "hp-lun", "my-pvc", DiskTypeLun, "virtio")
		if err == nil {
			t.Fatal("Expected error for invalid bus")
		}
		if !strings.Contains(err.Error(), "invalid bus") {
			t.Errorf("Expected 'invalid bus' error, got: %v", err)
		}
	})

	t.Run("invalid disk type", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), createTestVM("test-vm", "default", RunStrategyAlways))

		_, err := HotplugPVC(context.Background(), client, "default", "test-vm", "hp-disk", "my-pvc", DiskType("invalid"), "virtio")
		if err == nil {
			t.Fatal("Expected error for invalid disk type")
		}
		if !strings.Contains(err.Error(), "invalid disk type") {
			t.Errorf("Expected 'invalid disk type' error, got: %v", err)
		}
	})

	t.Run("default bus for disk type", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), createTestVM("test-vm", "default", RunStrategyAlways))

		result, err := HotplugPVC(context.Background(), client, "default", "test-vm", "hp-disk", "my-pvc", DiskTypeDisk, "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		disks, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "domain", "devices", "disks")
		if len(disks) != 1 {
			t.Fatalf("Expected 1 disk, got %d", len(disks))
		}
		disk := disks[0].(map[string]interface{})
		d := disk["disk"].(map[string]interface{})
		if d["bus"] != "virtio" {
			t.Errorf("Expected default bus 'virtio' for disk type, got %v", d["bus"])
		}
	})
}

func TestHotplugDataVolume(t *testing.T) {
	t.Run("adds DataVolume with hotpluggable true", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), createTestVM("test-vm", "default", RunStrategyAlways))

		result, err := HotplugDataVolume(context.Background(), client, "default", "test-vm", "hp-disk", "my-dv", DiskTypeDisk, "virtio")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		volumes, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "volumes")
		if len(volumes) != 1 {
			t.Fatalf("Expected 1 volume, got %d", len(volumes))
		}
		dv := volumes[0].(map[string]interface{})["dataVolume"].(map[string]interface{})
		if dv["name"] != "my-dv" {
			t.Errorf("Expected dataVolume name 'my-dv', got %v", dv["name"])
		}
		if dv["hotpluggable"] != true {
			t.Errorf("Expected hotpluggable true, got %v", dv["hotpluggable"])
		}
	})

	t.Run("appends to existing disks", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme(), withDisk(createTestVM("test-vm", "default", RunStrategyAlways)))

		result, err := HotplugDataVolume(context.Background(), client, "default", "test-vm", "hp-disk", "my-dv", DiskTypeDisk, "virtio")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		disks, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "domain", "devices", "disks")
		if len(disks) != 2 {
			t.Fatalf("Expected 2 disks, got %d", len(disks))
		}
		volumes, _, _ := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "volumes")
		if len(volumes) != 2 {
			t.Fatalf("Expected 2 volumes, got %d", len(volumes))
		}
	})

	t.Run("VM not found", func(t *testing.T) {
		client := fake.NewSimpleDynamicClient(runtime.NewScheme())
		_, err := HotplugDataVolume(context.Background(), client, "default", "missing", "d1", "dv1", DiskTypeDisk, "virtio")
		if err == nil {
			t.Fatal("Expected error for non-existent VM")
		}
		if !strings.Contains(err.Error(), "failed to get VirtualMachine") {
			t.Errorf("Expected 'failed to get VirtualMachine' error, got: %v", err)
		}
	})
}

func TestValidateDiskType(t *testing.T) {
	tests := []struct {
		name        string
		diskType    DiskType
		bus         string
		wantBus     string
		wantError   bool
		errContains string
	}{
		{name: "disk with empty bus defaults to virtio", diskType: DiskTypeDisk, bus: "", wantBus: "virtio"},
		{name: "disk with virtio", diskType: DiskTypeDisk, bus: "virtio", wantBus: "virtio"},
		{name: "disk with scsi", diskType: DiskTypeDisk, bus: "scsi", wantBus: "scsi"},
		{name: "disk with sata is invalid", diskType: DiskTypeDisk, bus: "sata", wantError: true, errContains: "invalid bus"},
		{name: "lun with empty bus defaults to scsi", diskType: DiskTypeLun, bus: "", wantBus: "scsi"},
		{name: "lun with scsi", diskType: DiskTypeLun, bus: "scsi", wantBus: "scsi"},
		{name: "lun with virtio is invalid", diskType: DiskTypeLun, bus: "virtio", wantError: true, errContains: "invalid bus"},
		{name: "invalid disk type", diskType: DiskType("floppy"), bus: "", wantError: true, errContains: "invalid disk type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus, err := ValidateDiskType(tt.diskType, tt.bus)
			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want to contain %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if bus != tt.wantBus {
				t.Errorf("bus = %q, want %q", bus, tt.wantBus)
			}
		})
	}
}
