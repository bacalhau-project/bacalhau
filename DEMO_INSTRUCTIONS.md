# Demo instructions


For the purposes of this demo, we will assume you use Digital Ocean. To install digital ocean CLI tool:
      - Mac: `brew update && brew install doctl`
      - Ubuntu: `sudo snap install doctl`
      - Others: <https://docs.digitalocean.com/reference/doctl/how-to/install/>

1. Create a bare metal instance that supports [Weave Ignite](https://ignite.readthedocs.io/en/stable/cloudprovider/#digitalocean)
  


```bash
# NOTE you should already have an ssh key, below, assuming you're using the default name 'id_rsa.pub'
# If it's not already in Digital Ocean, execute the following
doctl compute ssh-key import A_UNIQUE_KEY_NAME --public-key-file ~/.ssh/id_rsa.pub

# Use ID field frome output above - can get again with doctl compute ssh-key list
export SSH_FINGERPRINT="$(doctl compute ssh-key get ID_FIELD_FROM_OUTPUT --no-header --format 'FingerPrint')"
export DROPLET_NAME="bacalhau.node"

# Below requires having login in via 'doctl auth init'
doctl compute droplet create --size s-4vcpu-8gb --region nyc1 --image ubuntu-20-04-x64 --ssh-keys $SSH_FINGERPRINT $DROPLET_NAME

# Get the IP Address
export DROPLET_IP_ADDRESS="$(doctl compute droplet get $DROPLET_NAME --format PublicIPv4 --no-header)"
export DROPLET_USERNAME="STANDARD_UNIX_USERNAME"

# Bypass the yes/no host confirmation dialogue
ssh-keyscan $DROPLET_IP_ADDRESS >> $HOME/.ssh/known_hosts
# wait 20s for sshd daemon to initialize on the host
# Create a non-root user
ssh root@$DROPLET_IP_ADDRESS "useradd --create-home $DROPLET_USERNAME && usermod -aG sudo $DROPLET_USERNAME"
ssh root@$DROPLET_IP_ADDRESS 'echo "ALL            ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers'

# Copy the ssh keys in
cat ~/.ssh/id_rsa.pub | ssh root@$DROPLET_IP_ADDRESS  "su - $DROPLET_USERNAME -c 'mkdir -p ~/.ssh && tee -a ~/.ssh/authorized_keys'"

```


2. Open two terminal windows. In the first one, type the following commands:

```
ssh $DROPLET_USERNAME@$DROPLET_IP_ADDRESS
bash

# Either A) Download a build of the bacalhau cli from here - <https://github.com/filecoin-project/bacalhau/releases/>
sudo apt-get update
sudo apt-get zip -y
wget https://github.com/filecoin-project/bacalhau/releases/download/v0.0.2/bacalhau_v0.0.2_amd64.tar.gz
tar -xvzf bacalhau_v0.0.2_amd64.tar.gz
cd bacalhau_v0.0.2_amd64/

# Or B) build the latest release from scratch
sudo apt-get update && sudo apt-get install -y make gcc zip
sudo snap install go --classic
wget https://github.com/filecoin-project/bacalhau/archive/refs/heads/main.zip
unzip main.zip
cd bacalhau-main
go build

# Install IPFS *v0.11* specifically (due to issues in v0.12) via https://docs.ipfs.io/install/command-line/#official-distributions
cd ..
wget https://dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz
tar -xvzf go-ipfs_v0.11.0_linux-amd64.tar.gz
cd go-ipfs
sudo bash install.sh
cd -

# Install containerd
sudo apt-get install -y containerd

# Install the CNI plugin - https://ignite.readthedocs.io/en/stable/installation/#cni-plugins
export CNI_VERSION=v0.9.1
export ARCH=$([ $(uname -m) = "x86_64" ] && echo amd64 || echo arm64)
sudo mkdir -p /opt/cni/bin
curl -sSL https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-${ARCH}-${CNI_VERSION}.tgz | sudo tar -xz -C /opt/cni/bin

# Install Ignite
sudo apt-get install -y --no-install-recommends dmsetup openssh-client git binutils
export VERSION=v0.10.0
export GOARCH=$(go env GOARCH 2>/dev/null || echo "amd64")
for binary in ignite ignited; do
    echo "Installing ${binary}..."
    curl -sfLo ${binary} https://github.com/weaveworks/ignite/releases/download/${VERSION}/${binary}-${GOARCH}
    chmod +x ${binary}
    sudo mv ${binary} /usr/local/bin
done

# Optional: install and set runtime to docker
# sudo apt install -y docker.io
# BACALHAU_RUNTIME=docker
# Start bacalhau dev stack
./bacalhau --dev devstack
```

3. In the second terminal window, create a new file and add it to IPFS:

```
export DROPLET_NAME="bacalhau.node"
export DROPLET_IP_ADDRESS="$(doctl compute droplet get $DROPLET_NAME --format PublicIPv4 --no-header)"
export DROPLET_USERNAME="STANDARD_UNIX_USERNAME"
ssh $DROPLET_USERNAME@$DROPLET_IP_ADDRESS
bash
openssl rand -out large_file.txt -base64 $(( 2**30 * 3/4 ))
file_path="/home/STANDARD_UNIX_USERNAME/large_file.txt" # large_file.txt above
export IPFS_PATH="$(ls -d /tmp/bacalhau* | head -1)"
cid=$( ipfs add -q $file_path)

# Set the port number manually from the output of lsof
sudo lsof -i -P -n | grep bacalhau | grep LISTEN | tail -n 1
export JSON_RPC_PORT=12345

# Execute a command against IPFS on all nodes
# Counts the number of the letter 'W' in the file
./bacalhau submit --cids=$cid --commands="grep -o 'W' /ipfs/$cid | wc -l" --jsonrpc-port $JSON_RPC_PORT

```
4. Watch this resolve by watching this:

```
./bacalhau list --jsonrpc-port=$JSON_RPC_PORT
```

5. Reminder to delete your droplet when finished
```
doctl compute droplet delete $DROPLET_NAME 
```
