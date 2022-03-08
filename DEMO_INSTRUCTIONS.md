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

1. Download a build of the bacalhau cli from here - <https://github.com/filecoin-project/bacalhau/releases/>

1. Open two terminal windows. In the first one, type the following command:

```bash
# You may need to install ipfs, if not already installed - https://docs.ipfs.io/install/command-line/#official-distributions
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

1. In the second, create a new file and add it to IPFS:

```bash
openssl rand -out large_file.txt -base64 $(( 2**30 * 3/4 ))
```

1. Add this to the bacalhau network

```bash
# IN THE SECOND TERMINAL
# Add the file to ipfs and get the CID (you should have this command from the last terminal you ran)
file_path="path_to_file" # large_file.txt above
cid=$(IPFS_PATH=/tmp/bacalhau-ipfs489449709 ipfs add -q $file_path)

# Execute a command against IPFS on all nodes
 ./bacalhau submit --cids=$cid --commands="grep -o 'W' /ipfs/$cid | wc -l" # Counts the number of the letter 'W' in the file
```
You can watch this resolve by watching this:

```
./bacalhau list --jsonrpc-port=41923
```

STOPPED HERE WITH THE FOLLOWING ERROR - job just sitting there doing nothing:
```
#Terminal 1
aronchick@bacalhau:~$ ./code/bacalhau/bin/bacalhau list --jsonrpc-port=41923
 JOB       COMMAND                  DATA                     NODE                     STATE     STATUS                                             OUTPUT
 d978308c  grep -o 'W' /ipfs/Qm...  Qmdd3GdLhQJENPLs6VyG...  QmSPXxU5aFDwjMeqedkK...  selected  Job was selected because jobs CID are local:
                                                                                                 [Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu]

aronchick@bacalhau:~$

# TERMINAL 2
we are updating a job!:
&{JobId:d978308c-95d2-4193-99b6-a1c1f9fbd189 NodeId:QmSPXxU5aFDwjMeqedkKhGGWmEkSPPVjFtXE811SYak2cL State:selected Status:Job was selected because jobs CID are local:
 [Qmdd3GdLhQJENPLs6VyGByaZWytDCcjJezGUcWonEybrzu]
 Output:}
INFO[0001] Created VM with ID "8af9ab483799ac66" and name "d978308c-95d2-4193-99b6-a1c1f9fbd189a9fa74f4-e62f-46b8-87ae-28676ba25fd4"
INFO[0001] Networking is handled by "cni"
INFO[0001] Started Firecracker VM "8af9ab483799ac66" in a container with ID "ignite-8af9ab483799ac66"
INFO[0001] Waiting for the ssh daemon within the VM to start...
generating ED25519 keypair...done
peer identity: 12D3KooWLaE9mxuP57Pn4wC3JrKTnpmeqKWQJmYwB7RwjSbiYaqK
initializing IPFS node at /root/.ipfs
to get started, enter:

        ipfs cat /ipfs/QmQPeNsJPyVWPFDVHb77w8G42Fvo15z4bG2X8D2GhfbSXc/readme

2022-03-01T17:22:42.502Z        ERROR   provider.queue  queue/queue.go:124      Failed to enqueue cid: leveldb: closed
removed /dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN
removed /dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa
removed /dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb
removed /dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt
removed /ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
removed /ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
added /ip4/127.0.0.1/tcp/36815/p2p/12D3KooWPszx6ZjoexB2nMnWWxPCtxbnSv6S4tck61jPCDLGid5x
```
