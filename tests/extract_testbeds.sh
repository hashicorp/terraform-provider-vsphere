#!/usr/bin/env bash

# ------ READ ENV VARS START ------

vsphere_server=$(cat vctestbed.json | jq '.vcSpec.endpoint' | tr -d '"')
vsphere_user="$(cat vctestbed.json | jq '.vcSpec.user.username' | tr -d '"')@$(cat vctestbed.json | jq '.vcSpec.user.domain' | tr -d '"')"
vsphere_password=$(cat vctestbed.json | jq '.vcSpec.user.password' | tr -d '"')

host1_server=$(cat hosttestbed.json | jq '.hostSpec.name' | tr -d '"')
host1_password=$(cat hosttestbed.json | jq '.hostSpec.user.password' | tr -d '"')
host2_server=$(cat hosttestbed.json | jq '.hostSpec.name' | tr -d '"')
host2_password=$(cat hosttestbed.json | jq '.hostSpec.user.password' | tr -d '"')
host3_server=$(cat hosttestbed.json | jq '.hostSpec.name' | tr -d '"')
host3_password=$(cat hosttestbed.json | jq '.hostSpec.user.password' | tr -d '"')
host4_server=$(cat hosttestbed.json | jq '.hostSpec.name' | tr -d '"')
host4_password=$(cat hosttestbed.json | jq '.hostSpec.user.password' | tr -d '"')

nfs_server=$(cat nfstestbed.json | jq '.nfsShareSpec.serverIpAddress' | tr -d '"')

# ------ READ ENV VARS END ------

# ------ POPULATE setup_env_vars.sh START ------

sed -i '' "s/VSPHERE_SERVER=/VSPHERE_SERVER=$vsphere_server/" setup_env_vars.sh
sed -i '' "s/VSPHERE_USER=/VSPHERE_USER=$vsphere_user/" setup_env_vars.sh
sed -i '' "s/VSPHERE_PASSWORD=/VSPHERE_PASSWORD=$vsphere_password/" setup_env_vars.sh

sed -i '' "s/TF_VAR_VSPHERE_ESXI1=/TF_VAR_VSPHERE_ESXI1=$host1_server/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI1_PASSWORD=/TF_VAR_VSPHERE_ESXI1_PASSWORD=$host1_password/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI2=/TF_VAR_VSPHERE_ESXI2=$host2_server/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI2_PASSWORD=/TF_VAR_VSPHERE_ESXI2_PASSWORD=$host2_password/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI3=/TF_VAR_VSPHERE_ESXI3=$host3_server/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI3_PASSWORD=/TF_VAR_VSPHERE_ESXI3_PASSWORD=$host3_password/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI4=/TF_VAR_VSPHERE_ESXI4=$host4_server/" setup_env_vars.sh
sed -i '' "s/TF_VAR_VSPHERE_ESXI4_PASSWORD=/TF_VAR_VSPHERE_ESXI4_PASSWORD=$host4_password/" setup_env_vars.sh

sed -i '' "s/TF_VAR_VSPHERE_NAS_HOST=/TF_VAR_VSPHERE_NAS_HOST=$nfs_server/" setup_env_vars.sh

# ------ POPULATE setup_env_vars.sh END ------

# ------ GENERATE terraform.tfvars START ------

touch "terraform.tfvars"

echo "vcenter_username = \"$vsphere_user\"" >> "terraform.tfvars"
echo "vcenter_password = \"$vsphere_password\"" >> "terraform.tfvars"
echo "vcenter_server = \"$vsphere_server\"" >> "terraform.tfvars"
echo "hosts = [" >> "terraform.tfvars"
echo "  {" >> "terraform.tfvars"
echo "    hostname = \"$host1_server\"" >> "terraform.tfvars"
echo "    password = \"$host1_password\"" >> "terraform.tfvars"
echo "    username = \"root\"" >> "terraform.tfvars"
echo "  }," >> "terraform.tfvars"
echo "  {" >> "terraform.tfvars"
echo "    hostname = \"$host2_server\"" >> "terraform.tfvars"
echo "    password = \"$host2_password\"" >> "terraform.tfvars"
echo "    username = \"root\"" >> "terraform.tfvars"
echo "  }," >> "terraform.tfvars"
echo "  {" >> "terraform.tfvars"
echo "    hostname = \"$host3_server\"" >> "terraform.tfvars"
echo "    password = \"$host3_password\"" >> "terraform.tfvars"
echo "    username = \"root\"" >> "terraform.tfvars"
echo "  }," >> "terraform.tfvars"
echo "  {" >> "terraform.tfvars"
echo "    hostname = \"$host4_server\"" >> "terraform.tfvars"
echo "    password = \"$host4_password\"" >> "terraform.tfvars"
echo "    username = \"root\"" >> "terraform.tfvars"
echo "  }" >> "terraform.tfvars"
echo "]" >> "terraform.tfvars"
echo "nfs_host = \"$nfs_server\"" >> "terraform.tfvars"

# ------ GENERATE terraform.tfvars END ------