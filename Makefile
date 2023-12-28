IMAGE=registry.cn-beijing.aliyuncs.com/xxx/sd-api
TAG=v1

build-agent:
	sh script/codegen.sh
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/agent/agentServer cmd/agent/main.go
build-proxy:
	sh script/codegen.sh
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/proxy/proxyServer cmd/proxy/main.go
build-agent-image: build-agent
	chmod 755 build/agent/entrypoint.sh
	DOCKER_BUILDKIT=1 docker build  -f build/agent/Dockerfile -t ${IMAGE}:agent_${TAG} .
build-proxy-image: build-proxy
	chmod 755 build/proxy/entrypoint.sh
	DOCKER_BUILDKIT=1 docker build  -f build/proxy/Dockerfile -t ${IMAGE}:proxy_${TAG} .
