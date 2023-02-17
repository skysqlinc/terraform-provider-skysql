default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

install-alpha:
	go build -o terraform-provider-skysql-alpha .
	mv terraform-provider-skysql-alpha $(GOBIN)/terraform-provider-skysql-alpha

install-beta:
	go build -o terraform-provider-skysql-beta .
	mv terraform-provider-skysql-beta $(GOBIN)/terraform-provider-skysql-beta

install:
	go build -o terraform-provider-skysql .
	mv terraform-provider-skysql $(GOBIN)/terraform-provider-skysql
