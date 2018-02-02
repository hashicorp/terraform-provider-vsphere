.PHONY: build clean test infrastructure terraform terraform_init template check_terraform check_govc
export TF_DIR ?= terraform
TF_CMD ?= apply -input=false

test:
	go test -v .

build:
	scripts/build.sh

clean:
	rm -rf $(TF_DIR)/pkg .terraform

devclean: clean
	rm -rf terraform.tfstate* tf.log

template: check_govc
	scripts/template.sh

infrastructure: terraform

terraform: terraform_init
	terraform $(TF_CMD) -var-file=$(TF_DIR)/terraform.tfvars $(TF_DIR)

terraform_init:
	terraform init $(TF_DIR)

check_terraform:
ifeq ($(VSPHERE_SERVER),)
	$(error Please define the VSPHERE_SERVER environment variable)
endif
ifeq ($(VSPHERE_USER),)
	$(error Please define the VSPHERE_USER environment variable)
endif
ifeq ($(VSPHERE_PASSWORD),)
	$(error Please define the VSPHERE_PASSWORD environment variable)
endif

check_govc:
ifeq ($(GOVC_URL),)
	$(error Please define the GOVC_URL environment variable)
endif
ifeq ($(GOVC_USERNAME),)
	$(error Please define the GOVC_USERNAME environment variable)
endif
ifeq ($(GOVC_PASSWORD),)
	$(error Please define the GOVC_PASSWORD environment variable)
endif
