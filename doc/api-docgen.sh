#!/bin/bash

# generated cli markdown
../bin/goloop doc goloop-cli.md

echo "`date +'%Y/%m/%d %H:%M:%S'` generated swagger ./doc/swagger.yaml"
# swagger generate spec -b ../node -m -o swagger.yaml -i tags.yaml
swagger generate spec -b ../node -x server -m -o swagger.yaml -i tags.yaml
yq w -i swagger.yaml "info.version" "`git describe --always --tags --dirty`"

echo "`date +'%Y/%m/%d %H:%M:%S'` include markdown in the swagger"
# yq w -i swagger.yaml "tags[5].description" "`cat goloop-cli.md`"
yq w -i swagger.yaml "tags[2].description" "`cat goloop-cli.md`"

#rm swagger_node.yaml
rm goloop-cli.md

