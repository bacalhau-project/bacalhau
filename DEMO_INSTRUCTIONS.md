# Demo instructions

1. Create a bare metal instance that supports ignite (<https://ignite.readthedocs.io/en/stable/cloudprovider/#digitalocean>)
    1. For the purposes of this demo, we will assume you use Digital Ocean
    1. Install digital ocean CLI tool
       1. Mac: `brew update && brew install doctl`
       2. Ubuntu: `sudo snap install doctl`
       3. Others: <https://docs.digitalocean.com/reference/doctl/how-to/install/>
    1. Create VM:

```bash
# NOTE you should already have an ssh key, below, assuming you're using the default name 'id_rsa.pub'
# If it's not already in Digital Ocean, execute the following
doctl compute ssh-key import A_UNIQUE_KEY_NAME --public-key-file ~/.ssh/id_rsa.pub

# Use ID field frome output above - can get again with doctl compute ssh-key list
export SSH_FINGERPRINT="$(doctl compute ssh-key get ID_FIELD_FROM_OUTPUT --no-header --format 'FingerPrint')"

# Below requires having login in via 'doctl auth init'
doctl compute droplet create --size s-4vcpu-8gb --region nyc1 --image ubuntu-20-04-x64 --ssh-keys $SSH_FINGERPRINT bacalhau.node

# Get the IP Address
doctl compute droplet list # Get the IP address
export DROPLET_IP_ADDRESS="IP_ADDRESS_FROM_LIST"
export DROPLET_USERNAME="STANDARD_UNIX_USERNAME"

# Create a non-root user
ssh root@$DROPLET_IP_ADDRESS "useradd --create-home $DROPLET_USERNAME && usermod -aG sudo $DROPLET_USERNAME"
ssh root@$DROPLET_IP_ADDRESS 'echo "ALL            ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers'

# Copy the ssh keys in
cat ~/.ssh/id_rsa.pub | ssh root@$DROPLET_IP_ADDRESS  "su - $DROPLET_USERNAME -c 'mkdir -p ~/.ssh && tee -a ~/.ssh/authorized_keys'"

# Test
ssh $DROPLET_USERNAME@$DROPLET_IP_ADDRESS
```

1. Install Ignite:

```bash
ssh $DROPLET_USERNAME@$DROPLET_IP_ADDRESS

# Create a local ssh key
ssh-keygen

# Install go (if not already installed)
wget https://go.dev/dl/go1.17.7.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.7.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
source ~/.profile

# NOTE, you should check to make sure your go version matches your architecture
export VERSION=v0.10.0
export GOARCH=$(go env GOARCH 2>/dev/null || echo "amd64")

for binary in ignite ignited; do
    echo "Installing ${binary}..."
    curl -sfLo ${binary} https://github.com/weaveworks/ignite/releases/download/${VERSION}/${binary}-${GOARCH}
    chmod +x ${binary}
    sudo mv ${binary} /usr/local/bin
done

# Test the ignite installation
ignite version
```

1. Download a build of the bacalhau cli from here - <https://github.com/filecoin-project/bacalhau/releases/>

1. Open three terminal windows. In the first one, type the following command:

```bash
bacalhau --dev serve

# The above will output a line like the following:
# ./bacalhau serve --peer /ip4/0.0.0.0/tcp/8080/p2p/QmaxwhcG8cf8rduKg5dc5amsa2ycNRNmiqLKU5bq43ZkeH --jsonrpc-port 33743 --dev
```

1. In the second, copy and paste the command from above.
2. In the third, copy and paste the command from the SECOND window.
3. In the fourth, create a new file and add it to IPFS:

```bash
openssl rand -out large_file.txt -base64 $(( 2**30 * 3/4 ))
```

1. Add this to the bacalhau network

```bash
# You may need to install ipfs, if not already
sudo snap install ipfs
ipfs init

# Install containerd
sudo apt-get install containerd

# Install the CNI plugin - https://ignite.readthedocs.io/en/stable/installation/#cni-plugins
export CNI_VERSION=v0.9.1
export ARCH=$([ $(uname -m) = "x86_64" ] && echo amd64 || echo arm64)
sudo mkdir -p /opt/cni/bin
curl -sSL https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-${ARCH}-${CNI_VERSION}.tgz | sudo tar -xz -C /opt/cni/bin

# IN THE FOURTH TERMINAL (make sure, otherwise your 'cid' value below will be blank)
# Now add the file to ipfs and get the CID (you should have this command from the last terminal you ran)
cid=$(IPFS_PATH=data/ipfs/QmaxwhcG8cf8rduKg5dc5amsa2ycNRNmiqLKU5bq43ZkeH ipfs add -q large_file.txt)

# Execute a command against IPFS on all nodes
 ./bc submit --cids=$cid --commands="grep -o 'W' /ipfs/$cid | wc -l" # Counts the number of the letter 'W' in the file
```

STOPPED HERE WITH FOLLOWING ERROR:
```
we ignored a job self selecting:
&{Id:5edb3a03-5e8a-4960-80be-d8fd0f0355ca Cids:[Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu] Commands:[grep -o 'W' /ipfs/Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu | wc -l] Cpu:1 Memory:2 Disk:10}
we have 7 jobs
[{Id:970cc7f2-97c6-4786-95f4-c86b6fdf85c0 Cids:[] Commands:[grep -o 'W' large_file.txt | wc -l] Cpu:1 Memory:2 Disk:10} {Id:8c08596f-5281-4654-8efb-b006a44e0e5a Cids:[] Commands:[grep -o 'W' large_file.txt | wc -l] Cpu:1 Memory:2 Disk:10} {Id:caeb6350-dbdd-41fa-9384-710b003026eb Cids:[] Commands:[grep -o 'W' large_file.txt | wc -l] Cpu:1 Memory:2 Disk:10} {Id:97d3097b-349e-4b0d-a766-5a700f204b72 Cids:[] Commands:[grep -o 'W' /ipfs/ | wc -l] Cpu:1 Memory:2 Disk:10} {Id:00be1ea9-4d71-4d7c-a73a-51c2c811ee2a Cids:[Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu] Commands:[grep -o 'W' /ipfs/Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu | wc -l] Cpu:1 Memory:2 Disk:10} {Id:5edb3a03-5e8a-4960-80be-d8fd0f0355ca Cids:[Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu] Commands:[grep -o 'W' /ipfs/Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu | wc -l] Cpu:1 Memory:2 Disk:10} {Id:4bf86e2b-5536-42e1-8cd6-ecfc2bafb1d5 Cids:[Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu] Commands:[grep -o 'W' /ipfs/Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu | wc -l] Cpu:1 Memory:2 Disk:10}]
we ignored a job self selecting:
&{Id:4bf86e2b-5536-42e1-8cd6-ecfc2bafb1d5 Cids:[Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu] Commands:[grep -o 'W' /ipfs/Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu | wc -l] Cpu:1 Memory:2 Disk:10}
ERRO[0152] failed to run shell command: wait: remote command exited without exit status or exit signal
2022/02/22 04:07:13 Starting ipfs daemon --mount inside the vm failed with: exit status 1
```


```
