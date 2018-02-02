[Unit]
Description=Terraform vSphere Provider OVF Example
After=network.target

[Service]
ExecStart=${service_directory}/${server_binary_name}
User=${service_user}
WorkingDirectory=${service_directory}

[Install]
WantedBy=multi-user.target
