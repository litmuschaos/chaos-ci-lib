# Makefile for building Chaos CI LIB
# Reference Guide - https://www.gnu.org/software/make/manual/make.html

IS_DOCKER_INSTALLED = $(shell which docker >> /dev/null 2>&1; echo $$?)

# list only our namespaced directories
PACKAGES = $(shell go list ./... | grep -v '/vendor/')

# docker info
DOCKER_REPO ?= litmuschaos
DOCKER_IMAGE ?= chaos-ci-lib
DOCKER_TAG ?= ci

.PHONY: all
all: format lint deps build test trivy-check push 

.PHONY: help
help:
	@echo ""
	@echo "Usage:-"
	@echo "\tmake all   -- [default] builds the chaos exporter container"
	@echo ""

.PHONY: format
format:
	@echo "------------------"
	@echo "--> Running go fmt"
	@echo "------------------"
	@go fmt $(PACKAGES)

.PHONY: lint
lint:
	@echo "------------------"
	@echo "--> Running golint"
	@echo "------------------"
	@golint $(PACKAGES)
	@echo "------------------"
	@echo "--> Running go vet"
	@echo "------------------"
	@go vet $(PACKAGES)

.PHONY: deps
deps: _build_check_docker godeps

_build_check_docker:
	@if [ $(IS_DOCKER_INSTALLED) -eq 1 ]; \
		then echo "" \
		&& echo "ERROR:\tdocker is not installed. Please install it before build." \
		&& echo "" \
		&& exit 1; \
		fi;

godeps:
	@echo "INFO:\tverifying dependencies for chaos-ci-lib build ..."
	@go get -u -v golang.org/x/lint/golint
	@go get -u -v golang.org/x/tools/cmd/goimports


PHONY: build
build: go-binary-build docker-build

PHONY: go-binary-build
go-binary-build:
	@echo "---------------------------"
	@echo "--> Building Go Test Binary" 
	@echo "---------------------------"
	@sh build/generate_go_binary

docker-build: 
	@echo "----------------------------"
	@echo "--> Build chaos-ci-lib image" 
	@echo "----------------------------"
	@docker build . -f build/Dockerfile -t $(DOCKER_REPO)/$(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: push
push: docker-push

docker-push:
	@echo "---------------------------"
	@echo "--> Push chaos-ci-lib image" 
	@echo "---------------------------"
	@docker push $(DOCKER_REPO)/$(DOCKER_IMAGE):$(DOCKER_TAG)


.PHONY: trivy-check
trivy-check:

	@echo "------------------------"
	@echo "---> Running Trivy Check"
	@echo "------------------------"
	@./trivy --exit-code 0 --severity HIGH --no-progress $(DOCKER_REPO)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@./trivy --exit-code 0 --severity CRITICAL --no-progress $(DOCKER_REPO)/$(DOCKER_IMAGE):$(DOCKER_TAG)


.PHONY: install
install:

	@echo "--------------------------------------"
	@echo "---> Installing LitmusChaos"
	@echo "--------------------------------------"
	@go test litmus/install-litmus_test.go -v -count=1

.PHONY: uninstall
uninstall:

	@echo "--------------------------------------"
	@echo "---> Uninstalling LitmusChaos"
	@echo "--------------------------------------"
	@go test litmus/uninstall-litmus_test.go -v -count=1	

.PHONY: container-kill
container-kill:

	@echo "--------------------------------------"
	@echo "---> Running Container Kill Experiment"
	@echo "--------------------------------------"
	@go test experiments/container-kill_test.go -v -count=1

.PHONY: disk-fill
disk-fill:

	@echo "--------------------------------------"
	@echo "---> Running Disk Fill Experiment"
	@echo "--------------------------------------"
	@go test experiments/disk-fill_test.go -v -count=1	

.PHONY: node-cpu-hog
node-cpu-hog:

	@echo "--------------------------------------"
	@echo "---> Running Node CPU Hog Experiment"
	@echo "--------------------------------------"
	@go test experiments/node-cpu-hog_test.go -v -count=1	

.PHONY: node-io-stress
node-io-stress:

	@echo "--------------------------------------"
	@echo "---> Running Node IO Stess Experiment"
	@echo "--------------------------------------"
	@go test experiments/node-io-stress_test.go -v -count=1

.PHONY: node-memory-hog
node-memory-hog:

	@echo "--------------------------------------"
	@echo "---> Running Node Memory Hog Experiment"
	@echo "--------------------------------------"
	@go test experiments/node-memory-hog_test.go -v -count=1

.PHONY: pod-autoscaler
pod-autoscaler:

	@echo "--------------------------------------"
	@echo "---> Running Pod Autoscaler Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-autoscaler_test.go -v -count=1

.PHONY: pod-cpu-hog
pod-cpu-hog:

	@echo "--------------------------------------"
	@echo "---> Running Pod CPU Hog Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-cpu-hog_test.go -v -count=1				

.PHONY: pod-delete
pod-delete:

	@echo "--------------------------------------"
	@echo "---> Running Pod Delete Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-delete_test.go -v -count=1

.PHONY: pod-memory-hog
pod-memory-hog:

	@echo "--------------------------------------"
	@echo "---> Running Pod Memory Hog Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-memory-hog_test.go -v -count=1

.PHONY: pod-network-corruption
pod-network-corruption:

	@echo "--------------------------------------"
	@echo "---> Running Pod Network Corruption Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-network-corruption_test.go -v -count=1

.PHONY: pod-network-duplication
pod-network-duplication:

	@echo "--------------------------------------"
	@echo "---> Running Pod Network Duplication Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-network-duplication_test.go -v -count=1

.PHONY: pod-network-latency
pod-network-latency:

	@echo "--------------------------------------"
	@echo "---> Running Pod Network Latency Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-network-latency_test.go -v -count=1

.PHONY: pod-network-loss
pod-network-loss:

	@echo "--------------------------------------"
	@echo "---> Running Pod Network Loss Experiment"
	@echo "--------------------------------------"
	@go test experiments/pod-network-loss_test.go -v -count=1
