# Go Device Service SDK
[![Build Status](https://jenkins.edgexfoundry.org/view/EdgeX%20Foundry%20Project/job/edgexfoundry/job/device-sdk-go/job/main/badge/icon)](https://jenkins.edgexfoundry.org/view/EdgeX%20Foundry%20Project/job/edgexfoundry/job/device-sdk-go/job/main/) [![Code Coverage](https://codecov.io/gh/edgexfoundry/device-sdk-go/branch/main/graph/badge.svg?token=NoUXyBZgt6)](https://codecov.io/gh/edgexfoundry/device-sdk-go) [![Go Report Card](https://goreportcard.com/badge/github.com/edgexfoundry/device-sdk-go)](https://goreportcard.com/report/github.com/edgexfoundry/device-sdk-go) [![GitHub Latest Dev Tag)](https://img.shields.io/github/v/tag/edgexfoundry/device-sdk-go?include_prereleases&sort=semver&label=latest-dev)](https://github.com/edgexfoundry/device-sdk-go/tags) ![GitHub Latest Stable Tag)](https://img.shields.io/github/v/tag/edgexfoundry/device-sdk-go?sort=semver&label=latest-stable) [![GitHub License](https://img.shields.io/github/license/edgexfoundry/device-sdk-go)](https://choosealicense.com/licenses/apache-2.0/) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/edgexfoundry/device-sdk-go) [![GitHub Pull Requests](https://img.shields.io/github/issues-pr-raw/edgexfoundry/device-sdk-go)](https://github.com/edgexfoundry/device-sdk-go/pulls) [![GitHub Contributors](https://img.shields.io/github/contributors/edgexfoundry/device-sdk-go)](https://github.com/edgexfoundry/device-sdk-go/contributors) [![GitHub Committers](https://img.shields.io/badge/team-committers-green)](https://github.com/orgs/edgexfoundry/teams/device-sdk-go-committers/members) [![GitHub Commit Activity](https://img.shields.io/github/commit-activity/m/edgexfoundry/device-sdk-go)](https://github.com/edgexfoundry/device-sdk-go/commits)


## Overview

This repository is a set of Go packages that can be used to build Go-based [device services](https://docs.edgexfoundry.org/2.1/microservices/device/Ch-DeviceServices/) for use within the EdgeX framework.

## Usage

Developers can make their own device service by implementing the [`ProtocolDriver`](https://github.com/edgexfoundry/device-sdk-go/blob/main/pkg/interfaces/protocoldriver.go) interface for their desired IoT protocol, and the `main` function to start the Device Service. To implement the `main` function, the [`startup`](https://github.com/edgexfoundry/device-sdk-go/tree/main/pkg/startup) package can be optionally leveraged, or developers can write customized bootstrap code by themselves.

Please see the provided [simple device service](https://github.com/edgexfoundry/device-sdk-go/tree/main/example) as an example, included in this repository.

## Device identity in readings

The Creatus SDK sets the optional `DeviceID` field on each reading generated
from command values, including asynchronous pushes and command responses.
JSON and CBOR encode it as `deviceID` beside `deviceName`. Go consumers use
`reading.DeviceID`; JavaScript consumers use `reading.deviceID`.

The ID comes from the SDK's existing device-cache snapshot, without an
additional metadata request. Reading IDs, device names, and profile tags keep
their existing meaning. If the cached device has no ID, `deviceID` is omitted.
Consumers can retain name lookup for older readings that omit the field.

The native field is provided by the Creatus contracts fork at
`v4.0.2-creatus.5`. Driver modules must also select this fork, because Go does
not inherit dependency modules' `replace` directives:

```go
replace github.com/edgexfoundry/go-mod-core-contracts/v4 => github.com/Creatus-Cloud/go-mod-core-contracts/v4 v4.0.2-creatus.5
```

## Command Line Options

The following command line options are available

```text
  -cd=<path>
  --configDir=<path>
        Specify an alternate configuration directory.
  -p=<profile>
  --profile=<profile>
        Specify a profile other than default.
  -cf=<file>
  --configFile=<file>
        Indicates name of the local configuration file.
  -i=<instace>
  --instance=<instance>
        Provides a service name suffix which allows unique instance to be created.
        If the option is provided, service name will be replaced with "<name>_<instance>"
  -o
  --overwrite
        Overwrite configuration in the Registry with local values.
  -op 
  --overwriteProfiles
        Overwrite core-metadata with the versions of the associated device profile files.  
        *** Use with caution *** Use will clobber existing profiles in core-metadata, problematic 
        if those profiles were edited by hand intentionally or used by other services.
  -od
  --overwriteDevices
        Overwrite core-metadata with the versions of the associated device files.  
        *** Use with caution *** Use will clobber existing devices in core-metadata, problematic 
        if those devices were edited by hand intentionally.
  -r
  --registry
        Indicates the service should use the registry.
  -cp
  --configProvider
        Indicates to use Configuration Provider service at specified URL.
        URL Format: {type}.{protocol}://{host}:{port} ex: consul.http://localhost:8500
```

## Float value encoding

In EdgeX v1, float values had two kinds of encoding, [Base64](#base64), and [scientific notation (`eNotation`)](#scientific-notation-e-notation).

After v2, EdgeX only uses [scientific notation (`eNotation`)](#scientific-notation-e-notation) to present float values.

## Community

- Discussion: [https://github.com/orgs/edgexfoundry/discussions](https://github.com/orgs/edgexfoundry/discussions)
- Mailing lists: [https://lists.edgexfoundry.org/mailman/listinfo](https://lists.edgexfoundry.org/mailman/listinfo)

## License

[Apache-2.0](LICENSE)

## Versioning

Please refer to the EdgeX Foundry [versioning policy](https://wiki.edgexfoundry.org/pages/viewpage.action?pageId=21823969) for information on how EdgeX services are released and how EdgeX services are compatible with one another.  Specifically, device services (and the associated SDK), application services (and the associated app functions SDK), and client tools (like the EdgeX CLI and UI) can have independent minor releases, but these services must be compatible with the latest major release of EdgeX.

## Long Term Support

Please refer to the EdgeX Foundry [LTS policy](https://wiki.edgexfoundry.org/pages/viewpage.action?pageId=69173332) for information on support of EdgeX releases. The EdgeX community does not offer support on any non-LTS release outside of the latest release.


## Configuration preflight (Creatus)

The SDK exposes authenticated `POST /api/v3/config/validate`. Authenticated `GET` on the same route lists
`serviceID` and known configuration `sections` (names only). app-config uses this to
exclude shared defaults the receiving service does not use. The request has a
`serviceID` matching this service and a `configuration` object whose values are
strings and whose keys are relative Keeper paths, for example
`AppCustom/Limit`. Send the complete proposed service configuration, including
applicable common layers, rather than a patch or the entire gateway snapshot.

Register the existing validator directly during application startup or driver
`Initialize`, using the SDK's `RegisterCustomConfigValidator` method:

```go
if err := app.service.RegisterCustomConfigValidator(
    &config.ServiceConfig{}, appCustomConfigName, app.validateConfig,
); err != nil {
    return err
}
```

The prototype is the same outer struct used with `LoadCustomConfig`, containing
only the named top-level section (for example `AppCustom`). The callback has
signature `func(any) error` and receives that outer struct. Reuse the validation
method called by startup and Core Keeper updates, including any combined checks.
Runtime application callbacks remain separate.

The SDK retains only the prototype's type and creates a fresh zero-valued object
for every request. Prototype values and live settings are never copied into a
candidate, so deleting a required setting cannot be hidden by an old value.
Keeper's mapstructure conventions apply. The validator may prepare or normalize
the isolated candidate, but must not perform I/O or mutate live state.
`LoadCustomConfig` only records which sections need validation; it never discovers
methods. A loaded section without an explicit validator blocks preflight,
including when the candidate omits it.

`ApplicationService` and `DeviceServiceSDK` now expose this registration API.
Consumers require an SDK release containing it. Existing factory-based
`ConfigValidationRegistrar` registration remains available for configuration
groups requiring a custom factory.

Validation rejects wrong service IDs, unknown sections/fields, invalid value
types, and failures returned by custom validators. Standard configuration is
type checked and common service settings are checked; this does not invent
protocol-specific rules absent from a driver. Success is HTTP 200 with
`{"serviceID":"...","valid":true}`. Invalid configuration is HTTP 422 with
`errors` containing paths and messages; malformed requests are HTTP 400.
Request bodies are limited to 4 MiB. The device SDK additionally accepts
`devices` belonging to this service and calls its existing `ValidateDevice`
hook. Validation does not write to Core Keeper or replace active settings.
