#!/usr/bin/env python
import os

all_tokens = {"PYPI_TOKEN": False, "TEST_PYPI_TOKEN": False}

# If the PYPI_TOKEN is in the .secrets file, and the user doesn't add the --update flag, we're done
if os.path.exists(".secret") and "--update" not in os.sys.argv:
    with open(".secret", "r") as f:
        lines = f.readlines()
        # Check to see each of the "all_tokens" is in the .secret file
        for token in all_tokens:
            for line in lines:
                if line.startswith(f"{token}="):
                    tokenValue = line.split("=")[1].strip()
                    if tokenValue:
                        all_tokens[token] = True

        # If all the tokens are found, we're done
        if all(all_tokens[token] for token in all_tokens):
            print(f"All tokens found in .secret: {all_tokens}")
            exit(0)
        else:
            # If we're missing tokens, print out the missing tokens
            print(
                "Missing tokens in the .secret file - looking in .env and environment variables for them:"
            )
            for token in all_tokens:
                if not all_tokens[token]:
                    print(token)
            print("\n")

# Truncate the .secret file if it exists
if os.path.exists(".secret"):
    with open(".secret", "w") as f:
        f.truncate()

for token in all_tokens:
    tokenValue = None
    # First check if the PYPI_TOKEN is in the environment
    if token in os.environ:
        tokenValue = os.environ[token]
    # If not, check if it's in the .env file
    elif os.path.exists(".env"):
        with open(".env", "r") as f:
            lines = f.readlines()
            for line in lines:
                if line.startswith(token):
                    tokenValue = line.split("=")[1].strip()

    # If we found the token, write it to the .secret file, and delete the line if it's already there
    if tokenValue:
        with open(".secret", "+a") as f:
            f.writelines(f"{token}={tokenValue}\n")
        all_tokens[token] = True

if all(all_tokens[token] for token in all_tokens):
    print("All tokens found and written to .secret")
    exit(0)
else:
    print("Missing tokens:")
    for token in all_tokens:
        if not all_tokens[token]:
            print(token)
    exit(1)
