

**Common Errors**
- Using alternatives to `npm` - we have explored using `bun` but `prettier` did not work properly with it.

- We use `pre-commit` to run pre-commit hooks. If you run into an error like the below, it is likely because you are using 3.12+ (which as of the end of 2023, pre-commit does not support).
```
[INFO] Installing environment for https://github.com/pre-commit/pre-commit-hooks.
[INFO] Once installed this environment will be reused.
[INFO] This may take a few minutes...
An unexpected error has occurred: CalledProcessError: command: ('/home/daaronch/.cache/pre-commit/repowmf7smz7/py_env-python3.12/bin/python', '-mpip', 'install', '.')
return code: 2
stdout: (none)
stderr:
    ERROR: Exception:
    Traceback (most recent call last):
      File "/home/daaronch/.cache/pre-commit/repowmf7smz7/py_env-python3.12/lib/python3.12/site-packages/pip/_internal/cli/base_command.py", line 160, in exc_logging_wrapper
        status = run_func(*args)
                 ^^^^^^^^^^^^^^^
```
