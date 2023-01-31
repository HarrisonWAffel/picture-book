# Picture-Book

#### A container registry synchronization tool created during SUSE Hackweek 2023

---

Picture-book is a tool for loading and synchronizing (mirroring) container images into container registries. picture-book can be used
on any container registry which supports the Docker API. 

Two ways of loading registries are offered by picture-book

1. one-time synchronization
2. continuous synchronization

Before using either mode of synchronization you must configure picture-book properly

## Picture-book configuration

In order for picture-book to understand what registries should be synchronized they must be configured in picture-books `config.yaml` file

A sample configuration can be found below
```yaml 

# registries contains a list of registries that can be used and their relevant details 
registries:
  -
    # The FQDN of the regsitry. This can also be an IP address
    hostname: 'my-registry.com'
    # A prefix added to each image used to create a container repository within the registry server
    repository: 'test-images'
    # the DockerConfigJSON required to pull images from the registry being synchronized from
    pullAuthConfig: ''
    # the DockerConfigJSON required to push images to the registry being synchronized to
    pushAuthConfig: ''
    # A cron syntax representing the continuous synchronization schedule
    syncPeriod: '*/1 * * * *'
    # The executable that will list the images which must by synchronized, including their tags
    syncerScript: 'test.sh'
    # Any arguments that need to be provided to the syncerScript
    syncerScriptArgs: ''


# The display format that picture-book will use. This may be either 'spinner' or empty.
# The 'spinner' display looks more appealing, but is not very detailed and does not always handle multiple registries being synchronized at the same time well.
# Leaving display empty will result in picture-book logging the docker SDK output, which is much more verbose and deatiled.
display: "spinner"

# This is the configuration for picture-book's HTTP API which can be started during continuous synchronizations
api:
  # Toggle the API  
  enabled: true
  # The port the API server will operate on
  port: 8001
  # Denote if the API requires an Authorization header
  enableAuth: true
  # Provide the token which must be supplied in the Authorization header when API authentication is enabled.
  authToken: this-is-a-token-lol

```

## One-time synchronization

The simplest way to use picture-book is as a CLI tool using the `load` command. For example,

`picture-book load --registry my-test-registry.com`

will read in the picture-book `config.yaml` and find all details for the value provided in the `--registry` flag. It will then pull, re-tag, and push all the images returned by the sync script. 

## Continuous synchronization 

A core goal for picture-book is to provide a continuous synchronization server so that `picture-book load` does not have to be run manually, and can instead be run on a predefined schedule. 

Picture-book will perform automatic synchronization for all registries defined in the `config.yaml`, using the cron syntax defined in the `syncPeriod` attribute. 

## Picture-book HTTP API

When running picture-book in continuous mode an HTTP API server may optionally be started. This API can be used to query, pause, resume, and configure synchronization jobs. It is suggested that this API enable authentication if external access is possible.


API Reference
---
+ Endpoint: `http://localhost:8001/list`
+ Query options
  + `type`
    + `type=configured` will respond with all registries configured in the detected `config.yaml`
    + `type=active` will response with all active registry synchronizers




+ Endpoint: `http://localhost:8001/ops`
+ Query Options
  + `sync`
    + The hostname name of the registry whose synchronizer you would like to modify. **This query parameter is required for all `ops` endpoint**  
  + `action`
    + `action=details` Will respond with a JSON payload describing the configuration of the registries synchronizer
    + `action=pause` Will pause the synchronizer for the provided registry
    + `action=resume` Will resume the synchronizer for the provided registry

