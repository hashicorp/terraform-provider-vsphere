# Issue Reporting and Lifecycle

## Issue Reporting Checklists

We welcome your feature requests and bug reports. Below you'll find short checklists with guidelines for well-formed issues of each type.

### [Bug Reports](https://github.com/hashicorp/terraform-provider-vsphere/issues/new/choose)

- [ ] __Test Against the Latest Release__: 

    Make sure you test against the latest released version. It's possible we already fixed the bug you're experiencing.

- [ ] __Search for Possible Duplicate Issues__: 

    It's helpful to keep issues consolidated to one thread, so do a quick search on existing issues to check if anybody else has reported the same thing. You can [scope searches by the label "bug"](https://github.com/hashicorp/terraform-provider-vsphere/issues?q=is%3Aopen+is%3Aissue+label%3Abug) to help narrow your search.

- [ ] __Include the Steps to Reproduce__: 

    Provide steps to reproduce the issue, along with your `.tf` files. Without this information, it makes it much harder to diagnose and resolve the issue. Please ensure all secrets and identifiable information is removed.

- [ ] __For Panics, Include the `crash.log`__: 

    If you experienced a panic, please create a [gist](https://gist.github.com) of the *entire* generated crash log. Review the log to ensure no sensitive or identifiable information is included.

### [Enhancement and Feature Requests](https://github.com/hashicorp/terraform-provider-vsphere/issues/new/choose)

- [ ] __Search for Possible Duplicate Requests__: 

    It's helpful to keep requests consolidated to one thread. so do a quick search on existing issues to check if anybody else has reported the same thing. You can [scope searches by the label "enhancement"](https://github.com/hashicorp/terraform-provider-vsphere/issues?q=is%3Aopen+is%3Aissue+label%3Aenhancement) to help narrow your search.

- [ ] __Include a Use Case Description__: 

    In addition to describing the behavior of the feature you'd like to see added, it's helpful to also describe the reason why the enhancement or feature would be important and how it would benefit the provider users.

## Issue Lifecycle

1. The issue is reported on GitHub.

2. The issue is ackknowledged and categorized by a provider collaborator. Categorization is done via GitHub labels. We use one of `bug`, `enhancement`, `feature`, `documentation`, or `question`.

3. An initial triage process determines whether the issue is critical and must be addressed immediately, or can be left open for community discussion. In this step, we typically assign a size estimate to the work involved for that issue for our reference.

4. The issue is then queued in our backlog to be addressed in a pull request or commit. The issue number will be referenced in the commit message and pull request so that the code that fixes it is clearly linked.

5. The issue is closed. Sometimes, valid issues will be closed because they are tracked elsewhere or non-actionable. The issue is still indexed and available for future viewers or can be re-opened, if necessary.
