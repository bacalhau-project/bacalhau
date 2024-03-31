#!/usr/bin/env bash

TOOL_VERSIONS_FILE=${1:-"/tmp/.tool-versions"}

# Cut each line and install the package. The first time, it's the package, then it's the package and the version
cat ${TOOL_VERSIONS_FILE} | while read line; do \
    package=$(echo $line | cut -d ' ' -f 1); \
    version=$(echo $line | cut -d ' ' -f 2); \
    asdf plugin add $package; \
    asdf install $package $version; \
    asdf global $package $version; \
done

curl -L https://git-town.com/install.sh -o /tmp/git-town-install.sh && \
chmod +x /tmp/git-town-install.sh && \
/tmp/git-town-install.sh