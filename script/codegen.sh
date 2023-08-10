#!/bin/bash
# step 1:  go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen
# step 2: check output exist
home=$(pwd)
models=$home/pkg/models
handler=$home/pkg/handler
if [ ! -d "$models" ]; then
  mkdir -p $models
fi
if [ ! -d "$handler" ]; then
  mkdir -p $handler
fi
# step 3: gen code
oapi-codegen --config=script/module-config.yaml api/api.yaml
oapi-codegen --config=script/server-config.yaml api/api.yaml