# step 1:  go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen
# step 2: run produce
oapi-codegen --config=script/module-config.yaml api/api.yaml
oapi-codegen --config=script/server-config.yaml api/api.yaml