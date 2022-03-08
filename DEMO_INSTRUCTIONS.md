# Demo instructions

Create a bare metal instance that supports ignite (<https://ignite.readthedocs.io/en/stable/cloudprovider/#digitalocean>)
    1. For the purposes of this demo, we will assume you use Digital Ocean
    2. Install digital ocean CLI tool
       1. Mac: `brew update && brew install doctl`
       2. Ubuntu: `sudo snap install doctl`
       3. Others: <https://docs.digitalocean.com/reference/doctl/how-to/install/>
    3. Create VM:

```bash
# NOTE you should already have an ssh key, below, assuming you're using the default name 'id_rsa.pub'
# If it's not already in Digital Ocean, execute the following
doctl compute ssh-key import A_UNIQUE_KEY_NAME --public-key-file ~/.ssh/id_rsa.pub

# Use ID field frome output above - can get again with doctl compute ssh-key list
export SSH_FINGERPRINT="$(doctl compute ssh-key get ID_FIELD_FROM_OUTPUT --no-header --format 'FingerPrint')"

# Below requires having login in via 'doctl auth init'
doctl compute droplet create --size s-4vcpu-8gb --region nyc1 --image ubuntu-20-04-x64 --ssh-keys $SSH_FINGERPRINT bacalhau.node

# Get the IP Address
export DROPLET_IP_ADDRESS="$(doctl compute droplet get $DROPLET_NAME --format PublicIPv4 --no-header)"
export DROPLET_USERNAME="STANDARD_UNIX_USERNAME"

# Bypass the yes/no host confirmation dialogue
ssh-keyscan $DROPLET_IP_ADDRESS >> $HOME/.ssh/known_hosts
# Create a non-root user
ssh root@$DROPLET_IP_ADDRESS "useradd --create-home $DROPLET_USERNAME && usermod -aG sudo $DROPLET_USERNAME"
ssh root@$DROPLET_IP_ADDRESS 'echo "ALL            ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers'

# Copy the ssh keys in
cat ~/.ssh/id_rsa.pub | ssh root@$DROPLET_IP_ADDRESS  "su - $DROPLET_USERNAME -c 'mkdir -p ~/.ssh && tee -a ~/.ssh/authorized_keys'"

# Test
ssh $DROPLET_USERNAME@$DROPLET_IP_ADDRESS
```


Open two terminal windows. In the first one, type the following commands:

```
bash

# Either A) Download a build of the bacalhau cli from here - <https://github.com/filecoin-project/bacalhau/releases/>
sudo apt-get update
sudo apt-get zip -y
wget https://github.com/filecoin-project/bacalhau/releases/download/v0.0.2/bacalhau_v0.0.2_amd64.tar.gz
tar -xvzf bacalhau_v0.0.2_amd64.tar.gz

# Or B) build the latest release from scratch
sudo apt-get update
sudo apt-get install make gcc zip -y
sudo snap install go --classic
wget https://github.com/filecoin-project/bacalhau/archive/refs/heads/main.zip
unzip main.zip
cd bacalhau-main
go build

# Install IPFS *v0.11* specifically (due to issues in v0.12) via https://docs.ipfs.io/install/command-line/#official-distributions
wget https://dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz
tar -xvzf go-ipfs_v0.11.0_linux-amd64.tar.gz
cd go-ipfs
sudo bash install.sh

# Install containerd
sudo apt-get install containerd

# Install the CNI plugin - https://ignite.readthedocs.io/en/stable/installation/#cni-plugins
export CNI_VERSION=v0.9.1
export ARCH=$([ $(uname -m) = "x86_64" ] && echo amd64 || echo arm64)
sudo mkdir -p /opt/cni/bin
curl -sSL https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-${ARCH}-${CNI_VERSION}.tgz | sudo tar -xz -C /opt/cni/bin

# Start bacalhau dev stack
bacalhau --dev devstack
```

In the second terminal window, create a new file and add it to IPFS:

```
export DROPLET_NAME="bacalhau.node"
export DROPLET_IP_ADDRESS="$(doctl compute droplet get $DROPLET_NAME --format PublicIPv4 --no-header)"
export DROPLET_USERNAME="STANDARD_UNIX_USERNAME"
ssh $DROPLET_USERNAME@$DROPLET_IP_ADDRESS
bash
openssl rand -out large_file.txt -base64 $(( 2**30 * 3/4 ))
file_path="/home/STANDARD_UNIX_USERNAME/large_file.txt" # large_file.txt above
# Note the IPFS PATH directory value must be manually edited
export IPFS_PATH=/tmp/bacalhau-ipfs2925144396
cid=$( ipfs add -q $file_path)
export JSON_RPC_PORT=12345

# Execute a command against IPFS on all nodes
# Counts the number of the letter 'W' in the file
./bacalhau submit --cids=$cid --commands="grep -o 'W' /ipfs/$cid | wc -l" --jsonrpc-port $JSON_RPC_PORT

```
You can watch this resolve by watching this:

```
./bacalhau list --jsonrpc-port=$JSON_RPC_PORT
```

Reminder to delete your droplet when finished
```
doctl compute droplet delete $DROPLET_NAME 
```
