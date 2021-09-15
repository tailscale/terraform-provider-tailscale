# Contributing Guidelines

This document contains tips for contributing to this repository. These are not hard and fast rules, but suggestions and
advice.

## Raising Issues

Please use the GitHub [issues](https://github.com/davidsbond/terraform-provider-tailscale/issues/new/choose) tab to create a new issue, 
choosing an appropriate category from the list available.

This terraform provider is limited by the functionality available in the [Tailscale API](https://github.com/tailscale/tailscale/blob/main/api.md),
it may be the case that a feature you want implemented may not be available. If this is the case, please raise an issue
on the [Tailscale repository](https://github.com/tailscale/tailscale) describing what you would like to do via the API.

## Opening Pull Requests

Pull requests are welcome for this repository, please try to link the pull request to an issue or create an issue first before opening 
the pull request it relates to.

## Making Changes

To work in this repository, you will need go 1.17. You can use the standard go toolchain for building and testing your
changes. 

If you want to enable acceptance tests, you *must* set the `TF_ACC` environment variable. 

Be careful with acceptance
tests as they will run against the Tailscale API and use your local environment. You may end up borking your own Tailscale
devices/ACLs if you don't use a test account.
