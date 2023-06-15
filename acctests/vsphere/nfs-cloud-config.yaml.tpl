#cloud-config

write_files:
  - path: /etc/netplan/01-netcfg.yaml
    content: |
      network:
        version: 2
        renderer: networkd
        ethernets:
          ens192:
            dhcp4: no
            addresses: [${ip}/29]
            gateway4: ${gateway}
            nameservers:
              addresses: [8.8.8.8, 8.8.4.4]

runcmd:
  - netplan apply
  # internet should now be available
  - mkdir -p /nfs/ds1 /nfs/ds2 /nfs/ds3
  - apt-get update
  - apt-get install nfs-kernel-server -y
  - echo "/nfs *(rw,no_root_squash)" > /etc/exports
  - echo "/nfs/ds1 *(rw,no_root_squash)" >> /etc/exports
  - echo "/nfs/ds2 *(rw,no_root_squash)" >> /etc/exports
  - echo "/nfs/ds3 *(rw,no_root_squash)" >> /etc/exports
  - exportfs -a
