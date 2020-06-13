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

[Unreleased]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.2...HEAD
[v0.0.2]: https://github.com/newrelic/newrelic-kubernetes-operator/compare/v0.0.1...v0.0.2
