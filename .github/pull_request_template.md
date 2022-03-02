### Description

<!--- Please leave a helpful description of the pull request here. --->

### Acceptance tests
- [ ] Have you added an acceptance test for the functionality being added?
- [ ] Have you run the acceptance tests on this branch?

Output from acceptance testing:

<!--
Replace TestAccXXX with a pattern that matches the tests affected by this PR.

For more information on the `-run` flag, see the `go test` documentation at https://tip.golang.org/cmd/go/#hdr-Testing_flags.
-->
```
$ make testacc TESTARGS='-run=TestAccXXX'

...
```

### Release Note
Release note for [CHANGELOG](https://github.com/hashicorp/terraform-provider-vsphere/blob/main/CHANGELOG.md):
<!--
If change is not user facing, just write "NONE" in the release-note block below.
-->

```release-note
...
```
### References

<!---
Are there any other GitHub issues (open or closed) or pull requests that should be linked here? Vendor blog posts or documentation?
--->
