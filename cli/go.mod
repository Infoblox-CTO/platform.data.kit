module github.com/Infoblox-CTO/data-platform/cli

go 1.25

replace (
	github.com/Infoblox-CTO/data-platform/contracts => ../contracts
	github.com/Infoblox-CTO/data-platform/sdk => ../sdk
)

require (
	github.com/Infoblox-CTO/data-platform/sdk v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Infoblox-CTO/data-platform/contracts v0.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sync v0.13.0 // indirect
	oras.land/oras-go/v2 v2.5.0 // indirect
)
