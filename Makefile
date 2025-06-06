ifndef VERSION_REF
	VERSION_REF ?= $(shell git describe --tags --always --dirty="-dev")
endif

LDFLAGS := -ldflags='-s -w -X "main.VersionRef=$(VERSION_REF)"'
export GOFLAGS := -trimpath

GOFILES = $(shell find . -iname '*.go' | grep -v -e vendor -e _modules -e _cache -e /data/)
TEST_KUBECONFIG = .kube/kind-kubeapply-test.yaml

LAMBDAZIP := kubeapply-lambda-$(VERSION_REF).zip

# Main targets
.PHONY: kubeapply
kubeapply: data
	go build $(LDFLAGS) -o build/kubeapply ./cmd/kubeapply

.PHONY: install
install: data
	go install $(LDFLAGS) ./cmd/kubeapply

# Lambda and server-related targets
.PHONY: kubeapply-lambda
kubeapply-lambda: data
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -tags lambda.norpc -o build/kubeapply-lambda ./cmd/kubeapply-lambda

.PHONY: kubeapply-lambda-kubeapply
kubeapply-lambda-kubeapply: data
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/kubeapply ./cmd/kubeapply

.PHONY: lambda-zip
lambda-zip: clean kubeapply-lambda kubeapply-lambda-kubeapply
	$Q./scripts/create-lambda-bundle.sh $(LAMBDAZIP)

.PHONY: kubeapply-server
kubeapply-server: data
	go build $(LDFLAGS) -o build/kubeapply-server ./cmd/kubeapply-server

# Test and formatting targets
.PHONY: test
test: kubeapply data vet $(TEST_KUBECONFIG)
	PATH=$(CURDIR)/build:$$PATH KIND_ENABLED=true go test -count=1 -cover ./...

.PHONY: test-ci
test-ci: data vet
	# Kind is not supported in CI yet.
	# TODO: Get this working.
	KIND_ENABLED=false go test -count=1 -cover ./...

.PHONY: vet
vet: data
	go vet ./...

.PHONY: data
data: go-bindata
	go-bindata -pkg data -o ./data/data.go \
		-ignore=.*\.pyc \
		-ignore=.*__pycache__.* \
		./pkg/pullreq/templates/... \
		./scripts/...

.PHONY: fmtgo
fmtgo:
	goimports -w $(GOFILES)

.PHONY: fmtpy
fmtpy:
	autopep8 -i scripts/*py scripts/cluster-summary/cluster_summary.py

$(TEST_KUBECONFIG):
	./scripts/kindctl.sh start

.PHONY: go-bindata
go-bindata:
ifeq (, $(shell which go-bindata))
	go install github.com/kevinburke/go-bindata/v4/...@latest
endif

.PHONY: clean
clean:
	rm -Rf *.zip .kube build vendor
