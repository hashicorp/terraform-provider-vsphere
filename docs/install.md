<!--
© Broadcom. All Rights Reserved.
The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
SPDX-License-Identifier: MPL-2.0
-->

<!-- markdownlint-disable first-line-h1 no-inline-html -->

<img src="images/icon-color.svg" alt="VMware vSphere" width="150">

# Installing the Terraform Provider for VMware vSphere

## Automated Installation (Recommended)

The Terraform Provider for VMware vSphere is a Partner Tier provider.

Partner providers are owned and maintained by a partner in the HashiCorp Technology Partner Program. HashiCorp verifies the authenticity of the publisher and the provider are listed on the [Terraform Registry][terraform-registry] with a partner tier label.

### Configure the Terraform Configuration Files

Providers listed on the Terraform Registry can be automatically downloaded when initializing a working directory with `terraform init`. The Terraform configuration block is used to configure some behaviors of Terraform itself, such as the Terraform version and the required providers and versions.

**Example**: A Terraform configuration block.

```hcl
terraform {
  required_providers {
    vsphere = {
      source = "vmware/vsphere"
    }
  }
  required_version = ">= x.y.z"
}
```

You can use `version` locking and operators to require specific versions of the provider.

**Example**: A Terraform configuration block with the provider versions.

```hcl
terraform {
  required_providers {
    vsphere = {
      source  = "vmware/vsphere"
      version = ">= x.y.z"
    }
  }
  required_version = ">= x.y.z"
}
```

To specify a particular provider version when installing released providers, see the Terraform documentation [on provider versioning][terraform-provider-versioning]

### Verify Terraform Initialization Using the Terraform Registry

To verify the initialization, navigate to the working directory for your Terraform configuration and run `terraform init`. You should see a message indicating that Terraform has been successfully initialized and has installed the provider from the Terraform Registry.

**Example**: Initialize and Download the Provider.

```console
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding vmware/vsphere versions matching ">= x.y.z" ...
- Installing vmware/vsphere x.y.z ...
- Installed vmware/vsphere x.y.z
...

Terraform has been successfully initialized!
```

## Manual Installation

The latest release of the provider can be found in the [releases][releases]. You can download the appropriate version of the provider for your operating system using a command line shell or a browser.

This can be useful in environments that do not allow direct access to the Internet.

### Linux

The following examples use Bash on Linux (x64).

1. On a Linux operating system with Internet access, download the plugin from GitHub using the shell.

   ```console
   RELEASE=x.y.z
   wget -q https://github.com/vmware/terraform-provider-vsphere/releases/download/v${RELEASE}/terraform-provider-vsphere_${RELEASE}_linux_amd64.zip
   ```

2. Extract the plugin.

   ```console
   unzip terraform-provider-vsphere_${RELEASE}_linux_amd64.zip -d terraform-provider-vsphere_${RELEASE}
   ```

3. Create a directory for the provider.

   > **Note**
   >
   > The directory hierarchy that Terraform uses to precisely determine the source of each provider it finds locally.
   >
   > `$PLUGIN_DIRECTORY/$SOURCEHOSTNAME/$SOURCENAMESPACE/$NAME/$VERSION/$OS_$ARCH/`

   ```console
   mkdir -p ~/.terraform.d/plugins/local/vmware/vsphere/${RELEASE}/linux_amd64
   ```

4. Copy the extracted plugin to a target system and move to the Terraform plugins directory.

   ```console
   mv terraform-provider-vsphere_${RELEASE}/terraform-provider-vsphere_v${RELEASE} ~/.terraform.d/plugins/local/vmware/vsphere/${RELEASE}/linux_amd64
   ```

5. Verify the presence of the plugin in the Terraform plugins directory.

   ```console
   cd ~/.terraform.d/plugins/local/vmware/vsphere/${RELEASE}/linux_amd64
   ls
   ```

### macOS

The following example uses Bash (default) on macOS (Intel).

1. On a macOS operating system with Internet access, install wget with [Homebrew](https://brew.sh).

   ```console
   brew install wget
   ```

2. Download the plugin from GitHub using the shell.

   ```console
   RELEASE=x.y.z
   wget -q https://github.com/vmware/terraform-provider-vsphere/releases/download/v${RELEASE}/terraform-provider-vsphere_${RELEASE}_darwin_amd64.zip
   ```

3. Extract the plugin.

   ```console
   unzip terraform-provider-vsphere_${RELEASE}_darwin_amd64.zip -d terraform-provider-vsphere_${RELEASE}
   ```

4. Create a directory for the provider.

   > **Note**
   >
   > The directory hierarchy that Terraform uses to precisely determine the source of each provider it finds locally.
   >
   > `$PLUGIN_DIRECTORY/$SOURCEHOSTNAME/$SOURCENAMESPACE/$NAME/$VERSION/$OS_$ARCH/`

   ```console
   mkdir -p ~/.terraform.d/plugins/local/vmware/vsphere/${RELEASE}/darwin_amd64
   ```

5. Copy the extracted plugin to a target system and move to the Terraform plugins directory.

   ```console
   mv terraform-provider-vsphere_${RELEASE}/terraform-provider-vsphere_v${RELEASE} ~/.terraform.d/plugins/local/vmware/vsphere/${RELEASE}/darwin_amd64
   ```

6. Verify the presence of the plugin in the Terraform plugins directory.

   ```console
   cd ~/.terraform.d/plugins/local/vmware/vsphere/${RELEASE}/darwin_amd64
   ls
   ```

### Windows

The following examples use PowerShell on Windows (x64).

1. On a Windows operating system with Internet access, download the plugin using the PowerShell.

   ```powershell
   $RELEASE="x.y.z"
   Invoke-WebRequest -Uri "https://github.com/vmware/terraform-provider-vsphere/releases/download/v${RELEASE}/terraform-provider-vsphere_${RELEASE}_windows_amd64.zip" -OutFile "terraform-provider-vsphere_${RELEASE}_windows_amd64.zip"
   ```

2. Extract the plugin.

   ```powershell
   Expand-Archive terraform-provider-vsphere_${RELEASE}_windows_amd64.zip

   cd terraform-provider-vsphere_${RELEASE}_windows_amd64
   ```

3. Copy the extracted plugin to a target system and move to the Terraform plugins directory.

   > **Note**
   >
   > The directory hierarchy that Terraform uses to precisely determine the source of each provider it finds locally.
   >
   > `$PLUGIN_DIRECTORY/$SOURCEHOSTNAME/$SOURCENAMESPACE/$NAME/$VERSION/$OS_$ARCH/`

   ```powershell
   New-Item $ENV:APPDATA\terraform.d\plugins\local\vmware\vsphere\${RELEASE}\ -Name "windows_amd64" -ItemType "directory"

   Move-Item terraform-provider-vsphere_v${RELEASE}.exe $ENV:APPDATA\terraform.d\plugins\local\vmware\vsphere\${RELEASE}\windows_amd64\terraform-provider-vsphere_v${RELEASE}.exe
   ```

4. Verify the presence of the plugin in the Terraform plugins directory.

   ```powershell
   cd $ENV:APPDATA\terraform.d\plugins\local\vmware\vsphere\${RELEASE}\windows_amd64
   dir
   ```

### Configure the Terraform Configuration Files

A working directory can be initialized with providers that are installed locally on a system by using `terraform init`. The Terraform configuration block is used to configure some behaviors of Terraform itself, such as the Terraform version and the required providers source and version.

**Example**: A Terraform configuration block.

```hcl
terraform {
  required_providers {
    vsphere = {
      source  = "local/vmware/vsphere"
      version = ">= x.y.z"
    }
  }
  required_version = ">= x.y.z"
}
```

### Verify the Terraform Initialization of a Manually Installed Provider

To verify the initialization, navigate to the working directory for your Terraform configuration and run `terraform init`. You should see a message indicating that Terraform has been successfully initialized and the installed version of the Terraform Provider for VMware vSphere.

**Example**: Initialize and Use a Manually Installed Provider

```console
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding local/vmware/vsphere versions matching ">= x.y.x" ...
- Installing local/vmware/vsphere x.y.x ...
- Installed local/vmware/vsphere x.y.x (unauthenticated)
...

Terraform has been successfully initialized!
```

## Get the Provider Version

To find the provider version, navigate to the working directory of your Terraform configuration and run `terraform version`. You should see a message indicating the provider version.

**Example**: Terraform Provider Version from the Terraform Registry

```console
$ terraform version
Terraform x.y.z
on linux_amd64
+ provider registry.terraform.io/vmware/vsphere x.y.z
```

**Example**: Terraform Provider Version for a Manually Installed Provider

```console
$ terraform version
Terraform x.y.z
on linux_amd64
+ provider local/vmware/vsphere x.y.z
```

[releases]: https://github.com/vmware/terraform-provider-vsphere/releases
[terraform-provider-versioning]: https://developer.hashicorp.com/terraform/language/providers/configuration#version-provider-versions
[terraform-registry]: https://registry.terraform.io
