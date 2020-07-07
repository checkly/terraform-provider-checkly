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
	go build -o terraform-provider-checkly
	terraform init  
	terraform plan 

apply:
	terraform apply
