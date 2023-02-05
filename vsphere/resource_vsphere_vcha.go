package vsphere

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	gtask "github.com/vmware/govmomi/task"
	// "github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVsphereVcha() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereVchaCreate,
		Read:   resourceVsphereVchaRead,
		Update: resourceVsphereVchaUpdate,
		Delete: resourceVsphereVchaDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the vSphere datacenter where vCHA is to be deployed.",
			},
			"network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VCHA network name",
			},
			"netmask": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VCHA network mask",
			},
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node address",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node username",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node password",
			},
			"thumbprint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node SSL thumbprint",
			},
			"active_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node IP address",
			},
			"active_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node VM name",
			},
			"passive_datastore": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Passive node datastore name",
			},
			"passive_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Passive node IP address",
			},
			"passive_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Passive node VM name",
			},
			"witness_datastore": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Witness node datastore name",
			},
			"witness_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Witness node IP address",
			},
			"witness_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Witness node VM name",
			},
			/*
				"active": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"thumbprint": {
							Type:        schema.TypeBool,
							Optional:    false,
							Description: "Active node thumbprint",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node IP address",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node VM name",
						},
					}},
				},
				"passive": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"datastore": {
							Type:        schema.TypeBool,
							Optional:    false,
							Description: "Active node thumbprint",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node IP address",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node VM name",
						},
					}},
				},
				"witness": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"datastore": {
							Type:        schema.TypeBool,
							Optional:    false,
							Description: "Active node thumbprint",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node IP address",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node VM name",
						},
					}},
				},
			*/
		},
	}
}

func resourceVsphereVchaCreate(d *schema.ResourceData, meta interface{}) error {
	err := validateFields(d)
	if err != nil {
		return err
	}

	client := meta.(*Client).vimClient

	vdcs, err := buildDeploymentSpec(client, d)
	if err != nil {
		return err
	}

	deployVchaTask := types.DeployVcha_Task{
		This: *client.Client.ServiceContent.FailoverClusterConfigurator,
		DeploymentSpec: vdcs,
	}

	var newTask *object.Task
	// _, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	// taskResponse, err := methods.DeployVcha_Task(context.TODO(), client.Client, &deployVchaTask)
	taskResponse, err := methods.DeployVcha_Task(ctx, client.Client, &deployVchaTask)
	if err != nil {
		return fmt.Errorf("error while deploying vCHA.  Error: %s", err)
	}

	newTask = object.NewTask(client.Client, taskResponse.Returnval)

	p := property.DefaultCollector(client.Client)
	res, err := gtask.Wait(context.TODO(), newTask.Reference(), p, nil)
	if err != nil {
		return fmt.Errorf("vCHA deployment failed. %s", err)
	}
	taskResult := res.Result

	var hostID string
	taskResultType := taskResult.(types.ManagedObjectReference).Type
	switch taskResultType {
	case "VchaSystem":
		hostID = taskResult.(types.ManagedObjectReference).Value
	default:
		return fmt.Errorf("unexpected task result type encountered. Got %s while waiting ComputeResourceType or Vchasystem", taskResultType)
	}
	log.Printf("[DEBUG] Vcha added with ID %s", hostID)
	d.SetId(hostID)

	return resourceVsphereVchaRead(d, meta)
}

func resourceVsphereVchaRead(d *schema.ResourceData, meta interface{}) error {
	// NOTE: Destroying the host without telling vsphere about it will result in us not
	// knowing that the host does not exist any more.

/*
	// Look for host
	client := meta.(*Client).vimClient
	// hostID := d.Id()

	getVchaConfig := types.GetVchaConfig{
		This: *client.Client.ServiceContent.FailoverClusterConfigurator,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	taskResponse, err := methods.GetVchaConfig(ctx, client.Client, &getVchaConfig)
	if err != nil {
		return fmt.Errorf("error while deploying vCHA.  Error: %s", err)
	}

	// var newTask *object.Task
	// newTask = object.NewTask(client.Client, taskResponse.Returnval.Reference())
	log.Printf("[DEBUG] Vcha read with ID %s", hostID)
*/

	return nil
}

func resourceVsphereVchaUpdate(d *schema.ResourceData, meta interface{}) error {
	err := validateFields(d)
	if err != nil {
		return err
	}

	// client := meta.(*Client).vimClient

	return resourceVsphereVchaRead(d, meta)
}

func resourceVsphereVchaDelete(d *schema.ResourceData, meta interface{}) error {
	// client := meta.(*Client).vimClient
	// hostID := d.Id()

	return nil
}

func buildDeploymentSpec(client *govmomi.Client, d *schema.ResourceData) (types.VchaClusterDeploymentSpec, error) {
	finder := find.NewFinder(client.Client, false)
	datacenterName := d.Get("datacenter").(string)
	vchaNetworkName := d.Get("network").(string)
	vchaNetMask := d.Get("netmask").(string)

	vcds := types.VchaClusterDeploymentSpec{}

	// datacenter, err := getDatacenter(client, datacenterName)
	datacenter, err := finder.Datacenter(context.TODO(), datacenterName)
	if err != nil {
		return vcds,
			fmt.Errorf("error fetching datacenter %s: %s", datacenterName, err)
	}

	finder.SetDatacenter(datacenter)
	vchaNetwork, err := finder.Network(context.TODO(), vchaNetworkName)
	if err != nil {
		return vcds,
			fmt.Errorf("error fetching vCHA network %s: %s", vchaNetworkName, err)
	}

	dc := datacenter
	ctx, _ := context.WithTimeout(context.Background(), defaultAPITimeout)
	var dcProps mo.Datacenter
	if err := dc.Properties(ctx, dc.Reference(), nil, &dcProps); err != nil {
		return vcds, fmt.Errorf("error while retrieving properties for datacenter %s. Error: %s", datacenterName, err)
	}

	vmFolder := object.NewFolder(client.Client, dcProps.VmFolder)

	activeNodeName := d.Get("active_name").(string)
	activeNodeVm, err := finder.VirtualMachine(context.TODO(), activeNodeName)
	if err != nil {
		return vcds, fmt.Errorf("error while retrieving properties for active node %s. Error: %s", activeNodeName, err)
	}
	activeVc := activeNodeVm.Reference()


	activeNodeUsername := d.Get("username").(string)
	activeNodePassword := d.Get("password").(string)
	serviceLocatorNamePassword := types.ServiceLocatorNamePassword{
		Username: activeNodeUsername,
		Password: activeNodePassword,
	}

	// serviceLocatorCredential := serviceLocatorNamePassword.GetServiceLocatorCredential()
	// Active node configuration
	activeNodeIp := d.Get("active_ip").(string)
	activeNodeThumbprint := d.Get("thumbprint").(string)
	
	// instanceUuid := activeVc.Config.InstanceUuid

	address := d.Get("address").(string)
	mvc := types.ServiceLocator{
		InstanceUuid:  "",
		// InstanceUuid:  instanceUuid,
		Url:           fmt.Sprintf("https://%s", address),
		// Credential:    serviceLocatorCredential,
		Credential:    &serviceLocatorNamePassword,
		SslThumbprint: activeNodeThumbprint,
	}
	avs := types.SourceNodeSpec{
		ManagementVc: mvc,
		ActiveVc:     activeVc,
	}

	activeNodeIpSettings := types.CustomizationIPSettings{
		Ip: &types.CustomizationFixedIp{
			IpAddress: activeNodeIp,
		},
		SubnetMask: vchaNetMask,
	}
	activeVcNetworkConfigSpec := types.ClusterNetworkConfigSpec{
		NetworkPortGroup: vchaNetwork.Reference(),
		IpSettings:       activeNodeIpSettings,
	}

	// Passive node configuration
	passiveNodeIp := d.Get("passive_ip").(string)
	passiveNodeName := d.Get("passive_name").(string)
	passiveNodeDatastoreName := d.Get("passive_datastore").(string)

	passiveNodeDatastore, err := datastore.FromPath(client, passiveNodeDatastoreName, datacenter)
	if err != nil {
		return vcds,
			fmt.Errorf("error fetching datastore %s: %s", passiveNodeDatastoreName, err)
	}

	passiveNodeIpSettings := types.CustomizationIPSettings{
		Ip: &types.CustomizationFixedIp{
			IpAddress: passiveNodeIp,
		},
		SubnetMask: vchaNetMask,
	}
	passiveNodeDatastoreReference := passiveNodeDatastore.Reference()
	passiveNodeDeploymentSpec := types.PassiveNodeDeploymentSpec{
		NodeDeploymentSpec: types.NodeDeploymentSpec{
			Datastore:  &passiveNodeDatastoreReference,
			NodeName:   passiveNodeName,
			IpSettings: passiveNodeIpSettings,
			Folder:     vmFolder.Reference(),
		},
		FailoverIpSettings: (*types.CustomizationIPSettings)(nil),
	}

	// Witness node configuration
	witnessNodeIp := d.Get("witness_ip").(string)
	witnessNodeName := d.Get("witness_name").(string)
	witnessNodeDatastoreName := d.Get("witness_datastore").(string)
	witnessNodeDatastore, err := datastore.FromPath(client, witnessNodeDatastoreName, datacenter)
	if err != nil {
		return vcds,
			fmt.Errorf("error fetching datastore %s: %s", witnessNodeDatastoreName, err)
	}

	witnessNodeIpSettings := types.CustomizationIPSettings{
		Ip: &types.CustomizationFixedIp{
			IpAddress: witnessNodeIp,
		},
		SubnetMask: vchaNetMask,
	}
	witnessNodeDatastoreReference := witnessNodeDatastore.Reference()
	witnessNodeDeploymentSpec := types.NodeDeploymentSpec{
		Datastore:  &witnessNodeDatastoreReference,
		IpSettings: witnessNodeIpSettings,
		NodeName:   witnessNodeName,
		Folder:     vmFolder.Reference(),
	}
	witnessDeploymentSpec := witnessNodeDeploymentSpec.GetNodeDeploymentSpec()
	vcds = types.VchaClusterDeploymentSpec{
		ActiveVcSpec:          avs,
		ActiveVcNetworkConfig: &activeVcNetworkConfigSpec,
		PassiveDeploymentSpec: passiveNodeDeploymentSpec,
		WitnessDeploymentSpec: witnessDeploymentSpec,
	}
	return vcds, nil
}
