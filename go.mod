module github.com/terraform-providers/terraform-provider-vsphere

go 1.13

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.3.0

require (
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/aws/aws-sdk-go v1.28.7 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dustinkirkland/golang-petname v0.0.0-20191129215211-8e5a1ed0cff0 // indirect
	github.com/hashicorp/terraform v0.12.18
	github.com/mitchellh/copystructure v1.0.0
	github.com/terraform-providers/terraform-provider-null v1.0.0
	github.com/terraform-providers/terraform-provider-random v2.0.0+incompatible
	github.com/terraform-providers/terraform-provider-template v1.0.0
	github.com/vmware/govmomi v0.21.0
	github.com/vmware/vic v1.5.4
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
)
