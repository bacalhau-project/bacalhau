import pem
from Crypto.PublicKey import RSA

key_file = "/Users/enricorotundo/.bacalhau/user_id.pem"
with open(key_file, 'rb') as f:
   certs = pem.parse(f.read())
private_key = RSA.import_key(certs[0].as_bytes())
public_key = private_key.public_key()

print(public_key.export_key('PEM', pkcs=1).decode())