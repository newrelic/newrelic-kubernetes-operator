<a name="v0.0.7"></a>
## [v0.0.7] - 2020-10-15
### Features
- add parent kubernetes object references

<a name="v0.0.6"></a>
## [v0.0.6] - 2020-09-09
### Bug Fixes
- rollback ns change
- update manager namespace created

### Documentation Updates
- remove hardcoded examples in README

<a name="v0.0.5"></a>
## [v0.0.5] - 2020-09-01
### Features
- support for secrets in the headers of webhook alerts channels
- add schema for headers and payload definition

<a name="v0.0.4"></a>
## [v0.0.4] - 2020-08-10
<a name="v0.0.3"></a>
## [v0.0.3] - 2020-08-05
### Bug Fixes
- **buikd:** match release manager docker image tag with goreleaser

### Documentation Updates
- add instructions for monitoring with the go agent

### Features
- **alertChannel:** add initial support for alertChannel CRD

<a name="v0.0.2"></a>
## [v0.0.2] - 2020-06-11
### Bug Fixes
- set LeaderElectionID
- use per-function contexts and printer package
- revert gnostic version to pre-case-change
- try go 1.14
- go.mod wrangling
- skip go mod tidy

### Documentation Updates
- update the README
- update the other examples
- update policy example
- revert go version change

### Features
- support kubectl diff
- **ci:** add release workflow + goreleaser

<a name="v0.0.1"></a>
## v0.0.1 - 2020-05-29
### Bug Fixes
- log only when error occurs
- **build:** Fix manifest generation with renamed configs dir
- **build:** Disable CGO for all compiling operations
- **build:** ensure we generate the interfaces
- **build:** no trailing slash for BUILD_DIR, use updated CONFIG_ROOT for make deploy
- **build:** Correct spelling of DOCKER_IMAGE
- **changelog:** reference correct repo in git-chglog config
- **config:** increase memory for operator
- **rbac:** update rbac permissions

### Documentation Updates
- **README:** cleaned up instructions for running
- **build:** Recommend kustomize build ... | kubectl apply -f -
- **build:** Need to install cert-manager
- **build:** Documentation driven development
- **build:** Correct url for kustomize build in quick start
- **policy:** change example policy to create nrql & apm conditions
- **readme:** reorganize the README a bit
- **readme:** update table of contents
- **readme:** update examples, update helpful commands, other minor updates

### Features
- **alerts:** add apm alerts methods to interface
- **api:** merged upstream changes and revved API version
- **api:** extend the CRD to include API key and region
- **api:** fixed the tests
- **api:** refactored alerts client behavior to read from condition
- **api:** added webhook tests
- **api:** fixing linting
- **api:** fixing more linting
- **api:** added kubbernetes secrets support
- **api:** added secrets rbac bindings
- **ci:** add release
- **conditions:** adds initial template for APM conditions
- **examples:** add secrets example yaml, continue doc updates
- **manager:** add switch for dev logging
- **manager:** Add custom version / service info headers
- **manager:** Add version flag, and version / appName vars to pass through
- **policy:** added update logic to conditions created by policy controller
- **policy:** added tests and default webhook for policy
- **policy:** Adds policy scaffolding.
- **policy:** add defaulting and validation logic
- **policy:** added policy condition creation and deletion
- **policy:** Added most of reconcile function

[Unreleased]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.7...HEAD
[v0.0.7]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.6...v0.0.7
[v0.0.6]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.5...v0.0.6
[v0.0.5]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.4...v0.0.5
[v0.0.4]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.3...v0.0.4
[v0.0.3]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.2...v0.0.3
[v0.0.2]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.1...v0.0.2
