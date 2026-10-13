// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vmware/govmomi/vapi/namespace"
	"github.com/vmware/govmomi/vapi/rest"
	namespaceinstance "github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/namespace"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

func resourceVSphereNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVSphereNamespaceCreate,
		ReadContext:   resourceVSphereNamespaceRead,
		UpdateContext: resourceVSphereNamespaceUpdate,
		DeleteContext: resourceVSphereNamespaceDelete,
		CustomizeDiff: resourceVSphereNamespaceCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the vSphere Namespace",
				Required:    true,
				ForceNew:    true,
			},
			"supervisor": {
				Type:        schema.TypeString,
				Description: "The Supervisor which the vSphere Namespace belongs to",
				Required:    true,
				ForceNew:    true,
			},
			"vm_service": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content_libraries": {
							Type:        schema.TypeList,
							Description: "The content libraries associated with the vSphere Namespace",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"vm_classes": {
							Type:        schema.TypeList,
							Description: "The VM classes associated with the vSphere Namespace",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"storage": {
				Type:        schema.TypeList,
				Description: "The storage policies available in the vSphere Namespace, with optional storage limits",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"policy": {
							Type:        schema.TypeString,
							Description: "The identifier of the storage policy",
							Required:    true,
						},
						"limit": {
							Type:        schema.TypeInt,
							Description: "The maximum storage, in mebibytes (MiB), the vSphere Namespace can use for this policy. 0 or unset means no limit",
							Optional:    true,
						},
					},
				},
			},
			"access": {
				Type:        schema.TypeSet,
				Description: "The users and groups with access to the vSphere Namespace",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subject_type": {
							Type:         schema.TypeString,
							Description:  "The type of the subject: USER or GROUP",
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"USER", "GROUP"}, false),
						},
						"subject": {
							Type:        schema.TypeString,
							Description: "The name of the user or group, without the domain",
							Required:    true,
						},
						"domain": {
							Type:        schema.TypeString,
							Description: "The domain of the identity source that owns the subject",
							Required:    true,
						},
						"role": {
							Type:         schema.TypeString,
							Description:  "The role of the subject in the vSphere Namespace: OWNER, EDIT or VIEW",
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"OWNER", "EDIT", "VIEW"}, false),
						},
						"identity_provider": {
							Type:        schema.TypeString,
							Description: "The identifier of the identity provider. If unset, vCenter Single Sign-On is used",
							Optional:    true,
						},
					},
				},
			},
			"zone": {
				Type:        schema.TypeSet,
				Description: "The vSphere Zones of the vSphere Namespace. Requires vCenter 9.0.0 or later",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The identifier of the vSphere Zone",
							Required:    true,
						},
						"cpu_limit": {
							Type:        schema.TypeInt,
							Description: "The CPU limit, in megahertz (MHz), for the vSphere Namespace in this zone. 0 or unset means no zone limit",
							Optional:    true,
						},
						"cpu_reservation": {
							Type:        schema.TypeInt,
							Description: "The CPU reservation, in megahertz (MHz), for the vSphere Namespace in this zone. 0 or unset means no reservation",
							Optional:    true,
						},
						"memory_limit": {
							Type:        schema.TypeInt,
							Description: "The memory limit, in mebibytes (MiB), for the vSphere Namespace in this zone. 0 or unset means no zone limit",
							Optional:    true,
						},
						"memory_reservation": {
							Type:        schema.TypeInt,
							Description: "The memory reservation, in mebibytes (MiB), for the vSphere Namespace in this zone. 0 or unset means no reservation",
							Optional:    true,
						},
						"disallow_unreserved_direct_path_usage": {
							Type:        schema.TypeBool,
							Description: "Whether workloads that do not use a reserved VM class are disallowed from using a DirectPath device",
							Optional:    true,
						},
						"vm_reservation": {
							Type:        schema.TypeSet,
							Description: "The VM class instances reserved for the vSphere Namespace in this zone",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"reserved_vm_class": {
										Type:        schema.TypeString,
										Description: "The identifier of the VM class",
										Required:    true,
									},
									"count": {
										Type:        schema.TypeInt,
										Description: "The number of VM class instances to reserve",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"vpc_network": {
				Type:        schema.TypeList,
				Description: "The NSX VPC network configuration. Requires vCenter 9.0.0 or later",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vpc": {
							Type:        schema.TypeString,
							Description: "The NSX path of an existing VPC. If unset, the Supervisor creates one for the vSphere Namespace",
							Optional:    true,
							Computed:    true,
							ForceNew:    true,
						},
						"default_subnet_size": {
							Type:        schema.TypeInt,
							Description: "The default size of the subnets created for the vSphere Namespace",
							Optional:    true,
							Computed:    true,
						},
						"vpc_config": {
							Type:        schema.TypeList,
							Description: "The configuration of the VPC created for the vSphere Namespace. Ignored when vpc is set",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"private_cidrs": {
										Type:        schema.TypeList,
										Description: "The private CIDRs of the VPC",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Description: "The IPv4 address of the CIDR",
													Required:    true,
												},
												"prefix": {
													Type:        schema.TypeInt,
													Description: "The prefix length of the CIDR",
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
						"shared_subnet": {
							Type:        schema.TypeSet,
							Description: "The NSX subnets created outside the Supervisor that are associated with the vSphere Namespace. Requires vCenter 9.1.0 or later",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path": {
										Type:        schema.TypeString,
										Description: "The NSX path of the subnet",
										Required:    true,
									},
									"pod_default": {
										Type:        schema.TypeBool,
										Description: "Whether the subnet is the default network for vSphere Pods. Requires vCenter 9.1.1 or later",
										Optional:    true,
									},
									"vm_default": {
										Type:        schema.TypeBool,
										Description: "Whether the subnet is the default network for VMs. Requires vCenter 9.1.1 or later",
										Optional:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "The name of the subnet in the vSphere Namespace, which cannot be changed once set. Requires vCenter 9.1.1 or later",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceVSphereNamespaceCustomizeDiff(_ context.Context, d *schema.ResourceDiff, meta interface{}) error {
	version := viapi.ParseVersionFromClient(meta.(*Client).vimClient)
	return namespaceinstance.ValidateVersionRequirements(version, d)
}

func resourceVSphereNamespaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	spec := namespace.NamespaceInstanceCreateSpecV2{
		Namespace:  d.Get("name").(string),
		Supervisor: d.Get("supervisor").(string),
	}

	if storageSpecs := namespaceinstance.ExpandStorageSpecs(d); len(storageSpecs) > 0 {
		spec.StorageSpecs = &storageSpecs
	}

	if accessList := namespaceinstance.ExpandAccessList(d); len(accessList) > 0 {
		spec.AccessList = &accessList
	}

	if zones := namespaceinstance.ExpandZones(d); len(zones) > 0 {
		spec.Zones = &zones
	}

	spec.NetworkSpec = namespaceinstance.ExpandNetworkSpecCreate(d)

	if vmService := d.Get("vm_service").([]interface{}); len(vmService) > 0 {
		vmServiceData := vmService[0].(map[string]interface{})
		spec.VmServiceSpec = &namespace.VmServiceSpec{
			VmClasses:        structure.SliceInterfacesToStrings(vmServiceData["vm_classes"].([]interface{})),
			ContentLibraries: structure.SliceInterfacesToStrings(vmServiceData["content_libraries"].([]interface{})),
		}
	}

	if err := m.CreateNamespaceV2(ctx, spec); err != nil {
		return diag.FromErr(err)
	}

	if err := waitForNamespaceCreation(ctx, d, meta); err != nil {
		return err
	}

	return resourceVSphereNamespaceRead(ctx, d, meta)
}

func resourceVSphereNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	name := d.Get("name").(string)
	data, err := m.GetNamespaceV2(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}

	var vmServiceAttr []map[string]interface{}
	if len(data.VmServiceSpec.ContentLibraries) > 0 || len(data.VmServiceSpec.VmClasses) > 0 {
		vmService := make(map[string]interface{})
		vmService["content_libraries"] = data.VmServiceSpec.ContentLibraries
		vmService["vm_classes"] = data.VmServiceSpec.VmClasses
		vmServiceAttr = append(vmServiceAttr, vmService)
	}
	_ = d.Set("vm_service", vmServiceAttr)

	_ = d.Set("storage", namespaceinstance.FlattenStorageSpecs(data.StorageSpecs))
	_ = d.Set("access", namespaceinstance.FlattenAccessList(data.AccessList))
	_ = d.Set("zone", namespaceinstance.FlattenZones(d, data.Zones))
	_ = d.Set("vpc_network", namespaceinstance.FlattenNetworkSpecTrackingNames(d, data.NetworkSpec))

	d.SetId(name)

	return nil
}

func resourceVSphereNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	name := d.Get("name").(string)
	spec := namespace.NamespacesInstanceUpdateSpec{}

	if storageSpecs := namespaceinstance.ExpandStorageSpecs(d); len(storageSpecs) > 0 {
		spec.StorageSpecs = storageSpecs
	}

	if d.HasChange("access") {
		accessList := namespaceinstance.ExpandAccessList(d)
		spec.AccessList = &accessList
	}

	if d.HasChange("zone") {
		zones := namespaceinstance.ExpandZones(d)
		spec.Zones = &zones
	}

	if d.HasChange("vpc_network") {
		spec.NetworkSpec = namespaceinstance.ExpandNetworkSpecUpdate(d)
	}

	if vmService := d.Get("vm_service").([]interface{}); len(vmService) > 0 {
		vmServiceData := vmService[0].(map[string]interface{})
		spec.VmServiceSpec = namespace.VmServiceSpec{
			VmClasses:        structure.SliceInterfacesToStrings(vmServiceData["vm_classes"].([]interface{})),
			ContentLibraries: structure.SliceInterfacesToStrings(vmServiceData["content_libraries"].([]interface{})),
		}
	}

	return diag.FromErr(m.UpdateNamespace(ctx, name, spec))
}

func resourceVSphereNamespaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	if err := m.DeleteNamespace(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return waitForNamespaceDeletion(ctx, d, meta)
}

func waitForNamespaceDeletion(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	name := d.Get("name").(string)

	tickerCtx, cancel := context.WithTimeout(ctx, time.Minute*time.Duration(1))
	defer cancel()
	ticker := time.NewTicker(time.Second * time.Duration(5))
	defer ticker.Stop()

	for {
		select {
		case <-tickerCtx.Done():
			return diag.Errorf("timed out waiting for namespace to be deleted: %s", name)
		case <-ticker.C:
			_, err := m.GetNamespaceV2(ctx, name)
			if rest.IsStatusError(err, http.StatusNotFound) {
				return nil
			}
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
}

func waitForNamespaceCreation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	name := d.Get("name").(string)

	tickerCtx, cancel := context.WithTimeout(ctx, time.Minute*time.Duration(5))
	defer cancel()
	ticker := time.NewTicker(time.Second * time.Duration(15))
	defer ticker.Stop()

	for {
		select {
		case <-tickerCtx.Done():
			return diag.Errorf("failed to create namespace: %s", name)
		case <-ticker.C:
			data, err := m.GetNamespaceV2(ctx, name)
			if err != nil {
				return diag.FromErr(err)
			}

			switch data.ConfigStatus {
			case namespace.NamespaceInstanceConfigStatusRunning:
				return nil
			case namespace.NamespaceInstanceConfigStatusError:
				return diag.Errorf("namespace %s entered ERROR state", name)
			}
		}
	}
}
