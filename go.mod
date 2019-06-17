module github.com/terraform-providers/terraform-provider-vsphere

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.3.0

go 1.12

require (
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dustinkirkland/golang-petname v0.0.0-20190613200456-11339a705ed2 // indirect
	github.com/hashicorp/terraform v0.12.2
	github.com/mitchellh/copystructure v1.0.0
	github.com/terraform-providers/terraform-provider-null v1.0.0
	github.com/terraform-providers/terraform-provider-random v2.0.0+incompatible
	github.com/terraform-providers/terraform-provider-template v1.0.0
	github.com/vmware/govmomi v0.20.1
	github.com/vmware/vic v1.5.2
)
