package vsphere

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/clustercomputeresource"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/resourcepool"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
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
				Description: "Datacenter Name where vCHA is to be deployed.",
			},
			"resourcepool": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ResourcePool Name where vCHA is deployed",
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
				Description: "Active node Management IP address with port number",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node SSO User with Admin Privileges",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Active node admin password",
			},
			"thumbprint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node SSL thumbprint",
			},
			"active_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node vCHA IP address",
			},
			"active_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Active node VM name",
			},
			"passive_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Passive node vCHA IP address",
			},
			"passive_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Passive node VM name",
			},
			"witness_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Witness node vCHA IP address",
			},
			"witness_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Witness node VM name",
			},
			"passive_vcenter_datastore": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Datastore where Passive vCenter VM Deployed",
			},
			"witness_vcenter_datastore": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Datastore where Witness vCenter VM Deployed",
			},
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
		This:           *client.Client.ServiceContent.FailoverClusterConfigurator,
		DeploymentSpec: vdcs,
	}

	var newTask *object.Task
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	taskResponse, err := methods.DeployVcha_Task(ctx, client.Client, &deployVchaTask)
	if err != nil {
		return fmt.Errorf("error while deploying vCHA.  Error: %s", err)
	}

	newTask = object.NewTask(client.Client, taskResponse.Returnval)
	tctx, tcancel := context.WithTimeout(context.TODO(), 75*time.Minute)
	defer tcancel()
	// waits for 75 minutes to complete vCHA deployment, if task passed or failed then it skips the waiting
	result, err := newTask.WaitForResult(tctx, nil)
	log.Printf("[DEBUG] task result : %+v", result)
	if err != nil {
		return fmt.Errorf("vCHA deployment failed. %s", err)
	}
	d.SetId("vcha")

	return resourceVsphereVchaRead(d, meta)
}

func resourceVsphereVchaRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*Client).vimClient
	// datacenterName := d.Get("datacenter").(string)
	passiveNodeName := d.Get("passive_name").(string)
	witnessNodeName := d.Get("witness_name").(string)

	m := view.NewManager(client.Client)
	ctx, _ := context.WithTimeout(context.Background(), defaultAPITimeout)

	v, err := m.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return fmt.Errorf("error while creating ContainerView. Error: %s", err)
	}

	var results []mo.VirtualMachine
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"datastore", "name"}, &results)
	if err != nil {
		return fmt.Errorf("error while retrieving ContainerView Virtual Machines. Error: %s", err)
	}

	// var passiveDatastore (types.ManagedObjectReference)

	for _, vm := range results {
		if vm.Name == passiveNodeName {
			passiveDatastore := vm.Datastore[0]
			ds, _ := datastore.FromID(client, passiveDatastore.Value)
			d.Set("passive_vcenter_datastore", ds.Name())
		} else if vm.Name == witnessNodeName {
			witnessDatatsore := vm.Datastore[0]
			ds, _ := datastore.FromID(client, witnessDatatsore.Value)
			d.Set("witness_vcenter_datastore", ds.Name())
		} else {
			continue
		}

	}

	defer v.Destroy(ctx)

	return nil
}

func resourceVsphereVchaUpdate(d *schema.ResourceData, meta interface{}) error {
	/*
		err := validateFields(d)
		if err != nil {
			return err
		}

		// client := meta.(*Client).vimClient
		resourceVsphereVchaRead(d, meta)
	*/
	return nil
}

func resourceVsphereVchaDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	passiveNodeName := d.Get("passive_name").(string)
	witnessNodeName := d.Get("witness_name").(string)

	setVchaClusterMode := types.SetClusterMode_Task{
		This: *client.Client.ServiceContent.FailoverClusterManager,
		Mode: "disabled",
	}
	var setVchaClusterModeTask *object.Task
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	setVchaClusterModeTaskRes, err := methods.SetClusterMode_Task(ctx, client.Client, &setVchaClusterMode)
	log.Printf("[DEBUG] setVchaClusterModeTaskRes TASKRESPONSE : %+v", setVchaClusterModeTaskRes)
	if err != nil {
		return fmt.Errorf("error while setting vCHA to Disabled Mode.  Error: %s", err)
	}

	setVchaClusterModeTask = object.NewTask(client.Client, setVchaClusterModeTaskRes.Returnval)

	result, err := setVchaClusterModeTask.WaitForResult(ctx, nil)
	log.Printf("[DEBUG] setVchaClusterModeTask result : %+v", result)
	if err != nil {
		return fmt.Errorf("error while setting vCHA to Disabled Mode. Error %s", err)
	}

	destroyVchaTask := types.DestroyVcha_Task{
		This: *client.Client.ServiceContent.FailoverClusterConfigurator,
	}
	var destroyTaskStatus *object.Task
	dctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	destroyvchataskResponse, err := methods.DestroyVcha_Task(dctx, client.Client, &destroyVchaTask)
	log.Printf("[DEBUG] destroyvchataskResponse : %+v", destroyvchataskResponse)
	if err != nil {
		return fmt.Errorf("error while Destroying vCHA.  Error: %s", err)
	}
	tctx, tcancel := context.WithTimeout(context.TODO(), defaultAPITimeout)
	defer tcancel()
	destroyTaskStatus = object.NewTask(client.Client, destroyvchataskResponse.Returnval)

	resu, err := destroyTaskStatus.WaitForResult(tctx, nil)
	log.Printf("[DEBUG] Destroy task result : %+v", resu)
	if err != nil {
		return fmt.Errorf("vCHA Configuration Destroy failed. %s", err)
	}

	for _, vmName := range []string{passiveNodeName, witnessNodeName} {
		err := DeleteVirtualMachineFromName(client, d, vmName)
		if err != nil {
			return fmt.Errorf("error while deleteing Virtual Machine From Disk : Error :%s", err)
		}
	}
	d.SetId("")

	return nil
}

func buildDeploymentSpec(client *govmomi.Client, d *schema.ResourceData) (types.VchaClusterDeploymentSpec, error) {

	datacenterName := d.Get("datacenter").(string)
	vchaNetworkName := d.Get("network").(string)
	vchaNetMask := d.Get("netmask").(string)
	activeNodeName := d.Get("active_name").(string)
	poolID := d.Get("resourcepool").(string)
	activeNodeUsername := d.Get("username").(string)
	activeNodePassword := d.Get("password").(string)
	activeNodeIp := d.Get("active_ip").(string)
	activeNodeThumbprint := d.Get("thumbprint").(string)
	address := d.Get("address").(string)

	vcds := types.VchaClusterDeploymentSpec{}
	finder := find.NewFinder(client.Client, false)

	datacenter, err := finder.Datacenter(context.TODO(), datacenterName)
	if err != nil {
		return vcds,
			fmt.Errorf("error fetching datacenter %s: %s", datacenterName, err)
	}
	finder.SetDatacenter(datacenter)

	// Get vCHA Network Object from network name
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

	// Get Active vCenter VM Object with VM NAme
	activeNodeVm, err := finder.VirtualMachine(context.TODO(), activeNodeName)
	if err != nil {
		return vcds, fmt.Errorf("error while retrieving properties for active node %s. Error: %s", activeNodeName, err)
	}

	activeVc := activeNodeVm.Reference()
	m := view.NewManager(client.Client)

	v, err := m.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return vcds, fmt.Errorf("error while retrieving ContainerView for active VM node %s. Error: %s", activeNodeName, err)
	}

	var results []mo.VirtualMachine
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"datastore", "name"}, &results)
	if err != nil {
		return vcds, fmt.Errorf("error while retrieving ContainerView for active VM node %s. Error: %s", activeNodeName, err)
	}

	var vcenterDatastore (types.ManagedObjectReference)

	for _, vm := range results {
		if vm.Name == activeNodeName {
			vcenterDatastore = vm.Datastore[0]
		}
	}

	// Get Avaliable datastores for vCHA deployment
	// Get ResourcePool Object from ResourcePool ID
	pool, err := resourcepool.FromID(client, poolID)
	if err != nil {
		return vcds, fmt.Errorf("error while retrieving properties for resource pool %s. Error: %s", poolID, err)
	}
	cluster, _ := pool.Owner(ctx)

	cls := cluster.Reference()
	cls_val := cls.Value

	clsterDet, _ := clustercomputeresource.FromID(client, cls_val)

	datastores, _ := clsterDet.Datastores(ctx)
	var avaliableDatastores [](types.ManagedObjectReference)

	for each_dst := range datastores {
		dst_val := datastores[each_dst].Reference()
		if dst_val != vcenterDatastore {
			avaliableDatastores = append(avaliableDatastores, dst_val)
		}
	}
	serviceLocatorNamePassword := types.ServiceLocatorNamePassword{
		Username: activeNodeUsername,
		Password: activeNodePassword,
	}

	mvc := types.ServiceLocator{
		InstanceUuid:  "",
		Url:           fmt.Sprintf("https://%s", address),
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

	passiveNodeIpSettings := types.CustomizationIPSettings{
		Ip: &types.CustomizationFixedIp{
			IpAddress: passiveNodeIp,
		},
		SubnetMask: vchaNetMask,
	}

	passiveNodeDatastoreReference := avaliableDatastores[0].Reference()

	log.Printf("Passive datastore : %+v\n", passiveNodeDatastoreReference)
	resourcepoolReference := pool.Reference()
	passiveNodeDeploymentSpec := types.PassiveNodeDeploymentSpec{
		NodeDeploymentSpec: types.NodeDeploymentSpec{
			Datastore:    &passiveNodeDatastoreReference,
			NodeName:     passiveNodeName,
			IpSettings:   passiveNodeIpSettings,
			Folder:       vmFolder.Reference(),
			ResourcePool: &resourcepoolReference,
		},
		FailoverIpSettings: (*types.CustomizationIPSettings)(nil),
	}

	// Witness node configuration
	witnessNodeIp := d.Get("witness_ip").(string)
	witnessNodeName := d.Get("witness_name").(string)

	witnessNodeIpSettings := types.CustomizationIPSettings{
		Ip: &types.CustomizationFixedIp{
			IpAddress: witnessNodeIp,
		},
		SubnetMask: vchaNetMask,
	}

	// witnessNodeDatastoreReference := witnessNodeDatastore.Reference()
	witnessNodeDatastoreReference := avaliableDatastores[1].Reference()

	witnessNodeDeploymentSpec := types.NodeDeploymentSpec{
		Datastore:    &witnessNodeDatastoreReference,
		IpSettings:   witnessNodeIpSettings,
		NodeName:     witnessNodeName,
		Folder:       vmFolder.Reference(),
		ResourcePool: &resourcepoolReference,
	}
	witnessDeploymentSpec := witnessNodeDeploymentSpec.GetNodeDeploymentSpec()

	vcds = types.VchaClusterDeploymentSpec{
		ActiveVcSpec:          avs,
		ActiveVcNetworkConfig: &activeVcNetworkConfigSpec,
		PassiveDeploymentSpec: passiveNodeDeploymentSpec,
		WitnessDeploymentSpec: witnessDeploymentSpec,
	}
	defer v.Destroy(ctx)

	return vcds, nil
}

func DeleteVirtualMachineFromName(client *govmomi.Client, d *schema.ResourceData, vmName string) error {
	datacenterName := d.Get("datacenter").(string)
	finder := find.NewFinder(client.Client, false)
	datacenter, err := finder.Datacenter(context.TODO(), datacenterName)
	if err != nil {
		return fmt.Errorf("error fetching datacenter %s: %s", datacenterName, err)
	}
	finder.SetDatacenter(datacenter)

	vm, err := finder.VirtualMachine(context.TODO(), vmName)
	if err != nil {
		return fmt.Errorf("error while retrieving Virtual Machine %s. Error: %s", vmName, err)
	}
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("error fetching VM properties: %s", err)
	}
	// Shutdown the VM first.
	if vprops.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOff {
		if err := virtualmachine.GracefulPowerOff(client, vm, 10, true); err != nil {
			return fmt.Errorf("error shutting down virtual machine %s: Error: %s", vmName, err)
		}
	}

	// The final operation here is to destroy the VM.
	if err := virtualmachine.Destroy(vm); err != nil {
		return fmt.Errorf("error destroying virtual machine %s: Error :%s", vmName, err)
	}
	return nil
}
