#!/usr/bin/env bash

USERNAME=vscode

# Other env vars
export GO111MODULE=auto
export CGO_ENABLED=0
export DOCKER_BUILDKIT=1
export ASDF_DIR=/home/${USERNAME}/.asdf

git config --global advice.detachedHead false
git clone https://github.com/asdf-vm/asdf.git "${ASDF_DIR}" --branch v0.14.0
echo "source \${ASDF_DIR}/asdf.sh" >> /home/"${USERNAME}"/.zshrc
echo "export PATH=\${ASDF_DIR}/bin:\${PATH}" >> /home/"${USERNAME}"/.zshrc
export PATH="\${ASDF_DIR}/bin:=\${ASDF_DIR}/shims:\${PATH}"

TOOL_VERSIONS_FILE=${1:-".tool-versions"}

# Cut each line and install the package. The first time, it's the package, then it's the package and the version
TOOL_VERSIONS_CONTENT=$(cat "${TOOL_VERSIONS_FILE}")
echo "$TOOL_VERSIONS_CONTENT" | while read -r line; do \
    package=$(echo "$line" | cut -d ' ' -f 1); \
    version=$(echo "$line" | cut -d ' ' -f 2); \
    asdf plugin add "$package"; \
    asdf install "$package" "$version"; \
    asdf global "$package" "$version"; \
done

curl -L https://git-town.com/install.sh -o /tmp/git-town-install.sh && \
chmod +x /tmp/git-town-install.sh && \
/tmp/git-town-install.sh