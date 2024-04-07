---
sidebar_label: Setting Up Your Development Environment
---

If you are looking to develop on the project, this page will help you get started.

**Instructions**
- We recommend `brew` for all packages. Install it by following the instructions [here](https://brew.sh/) - `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
- In order to install all the tools, you'll need some core build tools. This should be the only tool you install outside of asdf (see below).
  - MAC: `brew install openssl readline sqlite3 xz zlib && xcode-select --install`
  - Linux: `sudo apt install -y git libc6-dev build-essential liblzma-dev zlib1g-dev libbz2-dev libncurses5-dev libffi-dev libssl-dev zlib1g-dev sqlite3 libreadline-dev libsqlite3-dev`
- Add the following to your `.bashrc` or `.zshrc`:
```
    (echo; echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"') >> /home/$USER/.bashrc
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
```
- Set environment variables:

```
export PYTHONVER='3.11.7'
export GOLANGCILINTVER='v1.51.2'
export GOLANGVER='1.21'
```
- We use asdf to manage our development environment. Install asdf by following the instructions [here](https://asdf-vm.com/#/core-manage-asdf-vm) - `brew install asdf`
- Add asdf to your .bashrc: `echo -e "\n. $(brew --prefix asdf)/libexec/asdf.sh" >> ~/.bashrc`
- Install the github client: `brew install gh`
- Login to the repository: `gh auth login -p ssh -w --hostname github.com`
- Clone the repository: `gh repo clone bacalhau-project/bacalhau`
- Log into the directory: `cd bacalhau`
- Install the plugins for asdf by executing this at the root of the directory: `cut -d' ' -f1 .tool-versions|xargs -I{} asdf plugin add {}`
- Now install all the versions of the tools by executing this at the root of the directory: `asdf install`
- Install global `uv` the virtual environment tool: `pip3 install uv`
- Create a virtual environment: `python -m uv venv`
- Enable direnv in your shell: `direnv allow`
- Add direnv to your shell: `echo 'eval "$(direnv hook bash)"' >> ~/.bashrc`
- Reload direnv: `direnv reload`
- Activate the virtual environment: `source .venv/bin/activate`
- REInstall uv (yes again - this is for the virtual environment): `python3 -m pip install uv`
- Update pip (in the virtual environment): `python3 -m pip install --upgrade pip`
- Install poetry: `python3 -m uv pip install poetry`
- Install docker repository:
```
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
```
- Install docker: `sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin`
- Add your user to the docker group: `sudo usermod -aG docker $USER`
- Logout - or rehash the group by running `newgrp docker`
- Install pre-commit: `make install-pre-commit`
- You're done! You can now run `make` to see all the commands you can run.


**Useful VSCode launch.json**

```
{
  // Use IntelliSense to learn about possible attributes.
  // Hover to view descriptions of existing attributes.
  // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
  "version": "0.2.0",
  "configurations": [
    {
      "type": "node",
      "request": "launch",
      "name": "Jest All",
      "program": "${workspaceFolder}/webui/node_modules/.bin/jest",
      "args": ["--runInBand", "--config", "webui/jest.config.js"],
      "console": "integratedTerminal",
      "internalConsoleOptions": "neverOpen",
      "windows": {
        "program": "${workspaceFolder}/webui/node_modules/jest/bin/jest"
      }
    },
    {
      "type": "node",
      "request": "launch",
      "name": "Jest Current File",
      "program": "${workspaceFolder}/webui/node_modules/.bin/jest",
      "args": ["${fileBasenameNoExtension}", "--config", "webui/jest.config.js"],
      "console": "integratedTerminal",
      "internalConsoleOptions": "neverOpen",
      "windows": {
        "program": "${workspaceFolder}/webui/node_modules/jest/bin/jest"
      }
    },
    {
        "name": "Launch test function",
        "type": "go",
        "request": "launch",
        "mode": "test",
        "program": "${workspaceFolder}",
        "args": [
            "-test.run",
            "MyTestFunction"
        ]
    },
    {
        "name": "Launch file",
        "type": "go",
        "request": "launch",
        "mode": "debug",
        "program": "${file}"
    },
    {
        "name": "Launch Package",
        "type": "go",
        "request": "launch",
        "mode": "auto",
        "program": "${fileDirname}"
    },
    {
        "name": "Launch WebUI",
        "type": "go",
        "request": "launch",
        "mode": "debug",
        "program": "main.go",
        "args": [
            "serve",
            "--web-ui"
            "--web-ui-port",
            "8888"
        ]
    }
    ]
}
  ]
}
```

**Common Errors**

- Using alternatives to `npm` - we have explored using `bun` but `prettier` did not work properly with it.

- We use `pre-commit` to run pre-commit hooks. If you run into an error like the below, it is likely because you are using 3.12+ (which as of the end of 2023, pre-commit does not support).

```
[INFO] Installing environment for https://github.com/pre-commit/pre-commit-hooks.
[INFO] Once installed this environment will be reused.
[INFO] This may take a few minutes...
An unexpected error has occurred: CalledProcessError: command: ('/home/daaronch/.cache/pre-commit/.../py_env-python3.12/bin/python', '-mpip', 'install', '.')
return code: 2
stdout: (none)
stderr:
    ERROR: Exception:
    Traceback (most recent call last):
      File "/home/daaronch/.cache/pre-commit/.../py_env-python3.12/lib/python3.12/site-packages/pip/_internal/cli/base_command.py", line 160, in exc_logging_wrapper
        status = run_func(*args)
                 ^^^^^^^^^^^^^^^
```
