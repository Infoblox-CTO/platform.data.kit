module github.com/Infoblox-CTO/data-platform/tests/e2e

go 1.25

require (
	github.com/Infoblox-CTO/data-platform/contracts v0.0.0
	github.com/Infoblox-CTO/data-platform/sdk v0.0.0
)

replace (
	github.com/Infoblox-CTO/data-platform/contracts => ../../contracts
	github.com/Infoblox-CTO/data-platform/sdk => ../../sdk
)
