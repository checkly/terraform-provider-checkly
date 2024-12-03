default: testacc
version="0.0.0-canary"

chip=amd64
ifeq ($(shell uname -m), arm64)
  chip=arm64
endif
ifeq ($(shell uname -s),Linux)
  os="linux"
endif
ifeq ($(shell uname -s),Darwin)
  os="darwin"
endif

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

local-sdk:
	go mod edit -replace github.com/checkly/checkly-go-sdk=../checkly-go-sdk

dev:
	# for dev purposes only, build the provider and install
	# it as dev/checkly/check + version number
	go build -o terraform-provider-checkly
	mkdir -p ~/.terraform.d/plugins/dev/checkly/checkly/${version}/${os}_${chip}/
	chmod +x terraform-provider-checkly
	mv terraform-provider-checkly ~/.terraform.d/plugins/dev/checkly/checkly/${version}/${os}_${chip}/terraform-provider-checkly_v${version}
	cd demo && rm -f .terraform.lock.hcl
	cd demo && TF_LOG=TRACE terraform init -upgrade
	cd local && rm -f .terraform.lock.hcl
	cd local && TF_LOG=TRACE terraform init -upgrade

fmt:
	go fmt ./checkly
	terraform fmt

# Generate docs
generate:
	cd tools; go generate ./...
