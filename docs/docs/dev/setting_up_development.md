---
sidebar_label: Setting Up Your Development Environment
---

If you are looking to develop on the project, this page will help you get started.

**Instructions**

- Set environment variables:

```
export PYTHONVER='3.11.7'
export GOLANGCILINTVER='v1.51.2'
export GOLANGVER='1.21'
```

- Install asdf: `brew install asdf`
- Install asdf python plug-in: `asdf plugin add python`
- Install python: `asdf local python $PYTHONVER`
- Install asdf golang plug-in: `asdf plugin add golang`
- Install golang: `asdf install golang $GOLANGVER`
- Install golangci-lint: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $GOLANGCILINTVER`

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
