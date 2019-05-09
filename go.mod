module github.com/terraform-providers/terraform-provider-vsphere

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.3.0
	github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	sourcegraph.com/sourcegraph/go-diff => github.com/sourcegraph/go-diff v0.5.1
)

require (
	cloud.google.com/go v0.38.0 // indirect
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/aws/aws-sdk-go v1.19.25 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dustinkirkland/golang-petname v0.0.0-20170921220637-d3c2ba80e75e // indirect
	github.com/golang/mock v1.3.0 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-getter v1.2.0 // indirect
	github.com/hashicorp/go-hclog v0.9.0 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/hcl2 v0.0.0-20190503213020-640445e16309 // indirect
	github.com/hashicorp/hil v0.0.0-20190212132231-97b3a9cdfa93 // indirect
	github.com/hashicorp/terraform v0.12.0-rc1
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/mitchellh/copystructure v1.0.0
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.2.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/terraform-providers/terraform-provider-null v1.0.0
	github.com/terraform-providers/terraform-provider-random v2.0.0+incompatible
	github.com/terraform-providers/terraform-provider-template v1.0.0
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmware/govmomi v0.20.0
	github.com/vmware/vic v1.5.2
	golang.org/x/crypto v0.0.0-20190506204251-e1dfcc566284 // indirect
	golang.org/x/net v0.0.0-20190503192946-f4e77d36d62c // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	golang.org/x/sys v0.0.0-20190506115046-ca7f33d4116e // indirect
	google.golang.org/genproto v0.0.0-20190502173448-54afdca5d873 // indirect
	google.golang.org/grpc v1.20.1 // indirect
)
