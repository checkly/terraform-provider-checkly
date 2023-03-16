default: build

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	go generate ./...

fmt:
	gofmt -s -w -e .
	terraform fmt

version=0.0.0
chip=amd64
name=terraform-provider-checkly
ifeq ($(shell uname -m), arm64)
  chip=arm64
endif

dev:
	go build -o ${name}
	mkdir -p ~/.terraform.d/plugins/dev/checkly/checkly/${version}/darwin_${chip}/
	chmod +x ${name}
	mv ${name} ~/.terraform.d/plugins/dev/checkly/checkly/${version}/darwin_${chip}/${name}_v${version}
	cd demo && rm -f .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
	cd demo && TF_LOG=TRACE terraform init -upgrade
	cd local && rm -f .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
	cd local && TF_LOG=TRACE terraform init -upgrade

test:
	go test -v -cover -timeout=120s -parallel=4 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: build install lint generate fmt test testacc
