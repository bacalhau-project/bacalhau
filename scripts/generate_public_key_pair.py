import base64
import os
import secrets
import string

from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa


# Function to generate a secure passphrase
def generate_secure_passphrase(length=16):
    alphabet = string.ascii_letters + string.digits + string.punctuation
    return "".join(secrets.choice(alphabet) for _ in range(length))


# Step 1: Create the directory
keypair_dir = "testdata/keypairs"
os.makedirs(keypair_dir, exist_ok=True)

# Step 2: Generate RSA Private Key
private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)

# Step 3: Generate Secure Passphrase
passphrase = generate_secure_passphrase()

# Step 4: Serialize Private Key with Passphrase
private_key_path = os.path.join(keypair_dir, "private_key.pem")
with open(private_key_path, "wb") as private_key_file:
    private_key_file.write(
        private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.TraditionalOpenSSL,
            encryption_algorithm=serialization.BestAvailableEncryption(
                passphrase.encode()
            ),
        )
    )

# Step 5: Generate Public Key
public_key = private_key.public_key()
public_key_path = os.path.join(keypair_dir, "public_key.pem")
with open(public_key_path, "wb") as public_key_file:
    public_key_file.write(
        public_key.public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo,
        )
    )

# Step 6: Base64 Encode the Keys
with open(private_key_path, "rb") as pk_file:
    private_key_base64 = base64.b64encode(pk_file.read()).decode("utf-8")
with open(public_key_path, "rb") as pub_file:
    public_key_base64 = base64.b64encode(pub_file.read()).decode("utf-8")

# Save Base64 Encoded Keys to Files
private_key_base64_path = os.path.join(keypair_dir, "private_key_base64.pem")
with open(private_key_base64_path, "w") as pkb64_file:
    pkb64_file.write(private_key_base64)

public_key_base64_path = os.path.join(keypair_dir, "public_key_base64.pem")
with open(public_key_base64_path, "w") as pubkb64_file:
    pubkb64_file.write(public_key_base64)

# Print the paths of generated files
print(f"Private Key Path: {private_key_path}")
print(f"Public Key Path: {public_key_path}")
print(f"Private Key Base64 Path: {private_key_base64_path}")
print(f"Public Key Base64 Path: {public_key_base64_path}")
print(f"Passphrase: {passphrase}")

# Step 7: Write environment variables to .env file, overwriting existing variables
env_file_path = os.path.join(os.getcwd(), ".env")

env_vars = {
    "PRIVATE_KEY_FILE": private_key_path,
    "PUBLIC_KEY_FILE": public_key_path,
    "PRIVATE_KEY_BASE64_FILE": private_key_base64_path,
    "PUBLIC_KEY_BASE64_FILE": public_key_base64_path,
    "PRIVATE_KEY_PASSPHRASE": passphrase,
}

# Read existing .env file if it exists
if os.path.exists(env_file_path):
    with open(env_file_path, "r") as env_file:
        existing_vars = dict(
            line.strip().split("=", 1) for line in env_file if "=" in line
        )
else:
    existing_vars = {}

# Update existing vars with new ones
existing_vars.update(env_vars)

# Write back to .env file
with open(env_file_path, "w") as env_file:
    for key, value in existing_vars.items():
        env_file.write(f"{key}={value}\n")

print(f".env file created/updated at: {env_file_path}")
