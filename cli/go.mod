module github.com/Infoblox-CTO/platform.data.kit/cli

go 1.25

replace (
	github.com/Infoblox-CTO/platform.data.kit/contracts => ../contracts
	github.com/Infoblox-CTO/platform.data.kit/sdk => ../sdk
)

require (
	github.com/Infoblox-CTO/platform.data.kit/sdk v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Infoblox-CTO/platform.data.kit/contracts v0.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sync v0.16.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	oras.land/oras-go/v2 v2.5.0 // indirect
)
