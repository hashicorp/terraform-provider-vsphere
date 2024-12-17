# Notes

The .tf files within this folder provision infrastructure bare metal resources on Equinix Metal. The resources had been streamlined down to a single somewhat large ESXI host on a vlan and it also created access resources such as passwords and a private key. The private key is used for a very specific part of `vsphere/testrun` where we SSH into the physical ESXI host and retrieve thumbprints for nested hosts.

The version of ESXI is installed and managed by Equinix Metal, right now set to vSphere 7. As detailed in the `acctests/README.md` a user must then install vSphere onto the ESXI host.