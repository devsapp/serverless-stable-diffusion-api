# step 1:  go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen
# step 2: run produce
oapi-codegen --config=module-config.yaml sd_proxy.yaml
oapi-codegen --config=server-config.yaml sd_proxy.yaml