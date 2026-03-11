package hotplug

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubevirt"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

const defaultDiskType = "disk"

func Tools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "vm_hotplug_pvc",
				Description: "Hotplug a PersistentVolumeClaim as a disk into a running VirtualMachine. The volume is added with hotpluggable: true to enable live attachment.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "The namespace of the virtual machine",
						},
						"name": {
							Type:        "string",
							Description: "The name of the virtual machine",
						},
						"diskName": {
							Type:        "string",
							Description: "The name for the disk and volume entry (must be unique within the VM)",
						},
						"claimName": {
							Type:        "string",
							Description: "The name of the PersistentVolumeClaim to hotplug",
						},
						"diskType": {
							Type:        "string",
							Enum:        []any{"disk", "lun"},
							Description: "The disk device type (default: disk). 'disk' supports virtio/scsi bus, 'lun' supports scsi bus only",
						},
						"bus": {
							Type:        "string",
							Enum:        []any{"virtio", "scsi"},
							Description: "The disk bus type. Defaults depend on diskType: disk=virtio, lun=scsi. Must be compatible with the chosen diskType",
						},
					},
					Required: []string{"namespace", "name", "diskName", "claimName"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Virtual Machine: Hotplug PVC",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(false),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: hotplugPVC,
		},
		{
			Tool: api.Tool{
				Name:        "vm_hotplug_datavolume",
				Description: "Hotplug a DataVolume as a disk into a running VirtualMachine. The volume is added with hotpluggable: true to enable live attachment.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "The namespace of the virtual machine",
						},
						"name": {
							Type:        "string",
							Description: "The name of the virtual machine",
						},
						"diskName": {
							Type:        "string",
							Description: "The name for the disk and volume entry (must be unique within the VM)",
						},
						"dataVolumeName": {
							Type:        "string",
							Description: "The name of the DataVolume to hotplug",
						},
						"diskType": {
							Type:        "string",
							Enum:        []any{"disk", "lun"},
							Description: "The disk device type (default: disk). 'disk' supports virtio/scsi bus, 'lun' supports scsi bus only",
						},
						"bus": {
							Type:        "string",
							Enum:        []any{"virtio", "scsi"},
							Description: "The disk bus type. Defaults depend on diskType: disk=virtio, lun=scsi. Must be compatible with the chosen diskType",
						},
					},
					Required: []string{"namespace", "name", "diskName", "dataVolumeName"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Virtual Machine: Hotplug DataVolume",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(false),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: hotplugDataVolume,
		},
	}
}

func hotplugPVC(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, err := api.RequiredString(params, "namespace")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	name, err := api.RequiredString(params, "name")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	diskName, err := api.RequiredString(params, "diskName")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	claimName, err := api.RequiredString(params, "claimName")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	diskType := kubevirt.DiskType(api.OptionalString(params, "diskType", defaultDiskType))
	bus := api.OptionalString(params, "bus", "")

	result, err := kubevirt.HotplugPVC(params.Context, params.DynamicClient(), namespace, name, diskName, claimName, diskType, bus)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	return formatResult("PVC", result)
}

func hotplugDataVolume(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, err := api.RequiredString(params, "namespace")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	name, err := api.RequiredString(params, "name")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	diskName, err := api.RequiredString(params, "diskName")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	dataVolumeName, err := api.RequiredString(params, "dataVolumeName")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	diskType := kubevirt.DiskType(api.OptionalString(params, "diskType", defaultDiskType))
	bus := api.OptionalString(params, "bus", "")

	result, err := kubevirt.HotplugDataVolume(params.Context, params.DynamicClient(), namespace, name, diskName, dataVolumeName, diskType, bus)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	return formatResult("DataVolume", result)
}

func formatResult(volumeType string, vm *unstructured.Unstructured) (*api.ToolCallResult, error) {
	marshalledYaml, err := output.MarshalYaml([]*unstructured.Unstructured{vm})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal VirtualMachine: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("# %s hotplugged successfully\n%s", volumeType, marshalledYaml), nil), nil
}
