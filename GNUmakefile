default: testacc
version="0.0.0-canary"

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

local-sdk:
	go mod edit -replace github.com/checkly/checkly-go-sdk=../checkly-go-sdk

dev:
	# for dev purposes only, build the provider and install
	# it as dev/checkly/check + version number,
	go build -o terraform-provider-checkly
	mkdir -p ~/.terraform.d/plugins/dev/checkly/checkly/${version}/darwin_amd64/
	chmod +x terraform-provider-checkly
	mv terraform-provider-checkly ~/.terraform.d/plugins/dev/checkly/checkly/${version}/darwin_amd64/terraform-provider-checkly_v${version}
	cd demo && rm -f .terraform.lock.hcl
	cd demo && TF_LOG=TRACE terraform init -upgrade

fmt:
	go fmt ./checkly
	terraform fmt

doc:
	./tools/tfplugindocs
