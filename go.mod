module github.com/terraform-providers/terraform-provider-vsphere

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.3.0
	github.com/golang/lint v0.0.0-20190227174305-5b3e6a55c961 => github.com/golang/lint v0.0.0-20181217174547-8f45f776aaf1
	github.com/vmware/vic => github.com/hashicorp/vic v1.5.10
)

require (
	github.com/aws/aws-sdk-go v1.17.6 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dustinkirkland/golang-petname v0.0.0-20170921220637-d3c2ba80e75e // indirect
	github.com/golang/protobuf v1.3.0 // indirect
	github.com/hashicorp/go-hclog v0.7.0 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/hcl2 v0.0.0-20190226234159-7e26f2f34612 // indirect
	github.com/hashicorp/hil v0.0.0-20190212132231-97b3a9cdfa93 // indirect
	github.com/hashicorp/terraform v0.12.0-alpha4.0.20190226230829-c2f653cf1a35
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mattn/go-isatty v0.0.6 // indirect
	github.com/mitchellh/copystructure v1.0.0
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.3.0 // indirect
	github.com/terraform-providers/terraform-provider-null v1.0.0
	github.com/terraform-providers/terraform-provider-random v2.0.0+incompatible
	github.com/terraform-providers/terraform-provider-template v1.0.0
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/vmihailenco/msgpack v4.0.2+incompatible // indirect
	github.com/vmware/govmomi v0.20.0
	github.com/vmware/vic v1.5.0
	github.com/zclconf/go-cty v0.0.0-20190212192503-19dda139b164 // indirect
	golang.org/x/crypto v0.0.0-20190227175134-215aa809caaf // indirect
	golang.org/x/net v0.0.0-20190227160552-c95aed5357e7 // indirect
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6 // indirect
	golang.org/x/sys v0.0.0-20190226215855-775f8194d0f9 // indirect
	google.golang.org/genproto v0.0.0-20190226184841-fc2db5cae922 // indirect
	google.golang.org/grpc v1.19.0 // indirect
)
