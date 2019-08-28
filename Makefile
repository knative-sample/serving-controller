all: manager

manager:
	@echo "build k8s manager"
	go build -o bin/controller cmd/main.go
run:
	@echo "run controller"
	export SYSTEM_NAMESPACE=knative-serving;export METRICS_DOMAIN=knative.dev/custom/controller;export CONFIG_LOGGING_NAME=config-logging;export CONFIG_OBSERVABILITY_NAME=config-observability; ./bin/controller

image:
	@echo "release tekton-proxy image"
	./build/build-image.sh
