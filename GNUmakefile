default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

setup:
	go get github.com/hashicorp/terraform
	go get github.com/terraform-providers/terraform-provider-template
	go install github.com/hashicorp/terraform
	go install github.com/terraform-providers/terraform-provider-template
	#After running the above commands, both Terraform core and the template 
	# provider will both be installed in the current GOPATH and $GOPATH/bin 
	# will contain both terraform and terraform-provider-template executables. 
	# This terraform executable will find and use the template provider plugin 
	# alongside it in the bin directory in preference to downloading and 
	# installing an official release.

replace-dep:
	go mod edit -replace github.com/checkly/checkly-go-sdk=../checkly-go-sdk

plan:
	# for dev purposes only, build the provider and install 
	# it as dev/checkly/check 0.0.1, 
	go build -o terraform-provider-checkly
	mkdir -p ~/.terraform.d/plugins/dev/checkly/checkly/0.0.1/darwin_amd64/
	chmod +x terraform-provider-checkly
	mv terraform-provider-checkly ~/.terraform.d/plugins/dev/checkly/checkly/0.0.1/darwin_amd64/terraform-provider-checkly_v0.0.1
	TF_LOG=TRACE terraform init  
	terraform plan 
	terraform apply

apply:
	terraform apply

format-code:
	go fmt ./checkly
	terraform fmt
