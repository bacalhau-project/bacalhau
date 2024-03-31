#!/usr/bin/env bash
# Syntax: ./setup-user.sh [USERNAME] [SECURE_PATH_BASE]

USERNAME=${1:-"bacalhau"}
DIRECTORY=${2:-"/workspaces/${USERNAME}"}
PYPROJECT_FILE=${3:-"/tmp/pyproject.toml"}

# Allow direnv to work in the current shell
mkdir -p "${DIRECTORY}"
echo "source ${DIRECTORY}/.venv/bin/activate" > "${DIRECTORY}/.envrc"
echo "export PATH=${DIRECTORY}/.venv/bin:${PATH}" >> "${DIRECTORY}/.envrc"
echo "$(direnv hook bash)" >> "${HOME}/.bashrc"
echo "$(direnv hook zsh)" >> "${HOME}/.zshrc"

source ${HOME}/.bashrc

# Setup zsh to use direnv
cd ${DIRECTORY}
cp ${PYPROJECT_FILE} ${DIRECTORY}/pyproject.toml
direnv allow

which python
python --version
python -m ensurepip
python -m pip install --upgrade pip
python -m pip install uv
python -m uv venv --seed ${DIRECTORY}/.venv

# Add the current directory to the safe directory list
git config --global --add safe.directory ${DIRECTORY}

python -m pip install poetry
python -m poetry install --no-root