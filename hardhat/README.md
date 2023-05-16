## MODICUM demo

Have docker and node.js >= v16 installed.

The following commands run in a tmux pane - they will boot and start geth:

```bash
cd MODICUM/src/js
npm install
npx hardhat node --port 10000
```

Update your SSH config file to add an extra Port 222:
```
Port 22
Port 222
```

```
sudo systemctl restart ssh
```

Now we fix the version of python:

```bash
sudo add-apt-repository ppa:deadsnakes/ppa
sudo apt update
sudo apt install -y python3.8-dev
python3.8 -m pip install virtualenv
```

Then create a virtualenv:

```bash
cd MODICUM/src/python/
python3.8 -m virtualenv venv
. venv/bin/activate
sudo apt install -y libcurl4-openssl-dev
pip3 install -e .
```

Now activate the virtual env in all panes:

```bash
. ./venv/bin/activate
source .env
```

Then we create a new ssh keypair:

```bash
ssh-keygen -f ~/.ssh/modicum
```

Now we adjust the values on the `src/python/.env` file paying note to the following:

 * `HOST` = `127.0.0.1`
 * `DIR` = `/home/kai/projects/protocol-labs`
 * `MODICUMPATH` = `${DIR}/MODICUM`
 * `PROJECT` = `MODICUM`
 * `GETHPATH` = `/usr/local/bin`
 * `pubkey` = the public key we just generated
 * `sshkey` = the path to the private key we just generated

Then we source the file and compile the contracts:

```bash
cd MODICUM/src/python/
source .env
echo $CONTRACTSRC
docker run -it --rm\
		--name solcTest \
		--mount type=bind,source="${CONTRACTSRC}",target=/solidity/input \
		--mount type=bind,source="${CONTRACTSRC}/output",target=/solidity/output \
		ethereum/solc:0.4.25 \
		--overwrite --bin --bin-runtime --ast --abi --asm -o /solidity/output /solidity/input/Modicum.sol
```

Then we setup influxDB - in another pane:

```bash
docker run -d \
  --name influx \
  -p 8086:8086 \
  influxdb:1.8.10
```

Now we need to setup the database:

```bash
docker exec -ti influx influx
> create database collectd;
> show databases;
exit
```

Now we start the various processes (each in it's own pane):

IMPORTANT: don't forget to activate the virtualenv in each pane!
IMPORTANT: don't forget to `source .env` in each pane!

IMPORTANT: run these in this exact order!

```bash
modicum runAsCM
```

```bash
modicum runAsSolver
```

```bash
sudo modicum runAsDir
```
(sudo because it will do a bunch of stuff like creating users :-O)

```bash
modicum runAsMediator
```

```bash
modicum startRP --path /home/kai/projects/protocol-labs/MODICUM/0_experiments/demo/ --index 1
```


```bash
modicum startJC --playerpath /home/kai/projects/protocol-labs/MODICUM/0_experiments/demo/ --index 0
```