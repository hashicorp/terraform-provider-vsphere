# Legacy scripts and testing workflow

This folder contain files and scripts mostly used in the CI and development workflow of the provider from previous maintainers that are no longer active/supported. The acceptance tests have a patchy history and have run in TeamCity, Circle, Packet (now Equinix Metal), then on local homelabs, and now finally back in Equinix Metal (for now). The vSphere provider is unique in nature, it operates much closer to physical hardware, providing test coverage for these environments has proven to be challenging over the years, as well as costly.

These files are being kept around and isolated until a new official acceptance testing setup is developed and documented. For the current Equinix Metal based setup, see the folder `acctests`.

The provider will be undergoing some technical debt servicing and part of that effort is to develop a more scalable acceptance testing workflow.