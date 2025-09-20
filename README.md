# Finch Resources
The Finch Resources module provides a manifest-driven resource management framework for loading, unloading, and caching runtime assets in Finch applications. It supports registration of custom resource systems for different asset types and includes built-in systems for common Ebitengine assets.

## Resource Manifest
The resource manifest is the driving mechanism behind the Finch resource framework. This manifest is considered the source of truth for locating and loading all assets of an application.

The module provides mechanisms for auto-generating the manifest as well as a resource handler enum for every resource defined in it. While the module provides these mechanisms, it is up to an application's implementation to trigger when and how the module generates these files.

## Usage
### Registering The Module
Within the `module` package there is a `Register()` method. Call this method at some point before the application is run.
```go
package main

import (
	"github.com/adm87/finch-core/finch"
	resmodule "github.com/adm87/finch-resources/module"
)

func main() {
	f := finch.NewApp()

    // Register the resource module
    resources.Register(f.ctx)

	if err := finch.Run(f); err != nil {
		panic(err)
	}
}
```
### Defining Custom Resource Systems
Defining a custom resource system allows different modules to define their own assets types. 

To start, define a model to store data for your asset.
```go
package conf

type ConfAsset {
    Field1 string
    Field2 []string
}
```
Next, define a struct that implements the `resources.ResourceSystem` interface.
```go
package conf

var confResourceSystemType = resources.NewResourceSystemKey[*ConfResourceSystem]()

type ConfResourceSystem struct {
    confs map[string]ConfAsset
    loading types.HashSet[string]
    mu sync.Mutex
}

func NewConfResourceSystem() *ConfResourceSystem {
    return &ConfResourceSystem{}
}

func (c *ConfResourceSystem) ResourceTypes() []string {
    return []string {"conf"}
}

func (c *ConfResourceSystem) Type() ResourceSystemType {
    return confResourceSystemType
}

func (c *ConfResourceSystem) Load(ctx finch.Context, handle ResourceHandle) error {
    // TASK: Add your implementation here
}

func (c *ConfResourceSystem) Unload(ctx finch.Context, handle ResourceHandle) error {
    // TASK: Add your implementation here

}

func (c *ConfResourceSystem) GetDependencies(ctx finch.Context, handle ResourceHandle) []ResourceHandle {
    // TASK: Add your implementation here
}

func (c *ConfResourceSystem) GenerateMetadata(ctx finch.Context, key string, metadata *Metadata) error {
    // TASK: Add your implementation here

}

func (c *ConfResourceSystem) IsLoaded(handle ResourceHandle) bool {
    // TASK: Add your implementation here
}
```
Finally, register your custom resource system with the resource module
```go
import (
    "github.com/adm87/finch-core/finch"

    resmodule "github.com/adm87/finch-resources/module"
    "github.com/adm87/finch-resources/resources"
)

func main() {
	f := finch.NewApp()

    // Register the resource module
    resmodule.Register(f.ctx)

    // Register custom *.conf resource system
    resources.RegisterSystem(conf.NewConfResourceSystem())

	if err := finch.Run(f); err != nil {
		panic(err)
	}
}
```
With this in place, the resource module will now be able to recognize resources with the `.conf` extension. To access your loaded resources, you will need to define a global method within your module to get the system from the resource module and return the instances of the asset.
```go
// ResourceHandle is a type-safe key for accessing resources managed by the module.
// Use it to retrieve assets from your custom resource system.
func GetConf(handle resources.ResourceHandle) (*ConfAsset, bool) {
    sys, ok := resources.GetSystem(confResourceSystemType).(*ConfResourceSystem)
    if !ok {
        return nil, false
    }
    // Retrieve the asset from the system's cache
    c, exists := sys.confs[handle.Key()]
    return &c, exists
}
```
The resource module does not currently provide a way to retrieve untyped resources. Nor does it provide a method to retrieve the raw `[]byte`.
### Generating Manifest and Enums, Loading and Accessing Resources
In the `resources` and `enums` packages within the module, there are generation methods. You will need to create a mechanism for calling these to update the manifest and ResourceHandle enum. Manifest generation ensures all assets are discoverable, while enum generation provides type-safe handles for resource access.
```go
m := resources.GenerateManifest(ctx, resourcePath)
if err : fsys.WriteJson(filepath.Join(resourcePath, resources.JsonName), m); err != nil {
    return err
}

content := enums.Generate(ctx, m, "data", resourcePath)
if err := os.WriteFile(path.Join(resourcePath, "enums.go"), content, 0644); err != nil {
    return err
}

resources.Load(ctx, data.GameConf)

gameConf, exists := GetConf(data.GameConf)
```

> Important: It's possible for the resource module to load assets concurrently if a single request exceeds 100 resources. It is recommended that you ensure your resource system is thread safe when reading/writing the cache.
---

**Troubleshooting & FAQ**

- If your custom resource system isn't loading assets, check that you registered it before calling `finch.Run()`.
- For thread safety, always protect shared maps or caches with a mutex if your system supports concurrent loading.
- If you need to access raw bytes, you must extend the resource system interface, as the module does not provide this by default.