# OVH Bare Metal Preparation

## Context

We have 4 high-spec `ubuntu-2204-lts` nodes on OVH US @ Hillsboro (HIL1) - United States.
They are HGR-HCI-5 machines, check [this page](https://us.ovhcloud.com/bare-metal/high-grade/hgr-hci-5/) for more details.
These are their IP addresses:

* 51.81.184.74
* 51.81.184.112
* 51.81.184.118
* 51.81.184.117

These machine need ad-hoc preparation to host Bacalhau nodes, these steps are specific to this OVH bundle and therefore is worth keeping these notes separated from the normal installation instructions.
This page contains instructions on how to prepare those hosts to run Bacalhau:

1. Install Ubuntu via OVH Console
1. Configure RAID for data disks
1. Configure Firewall
1. Add team's ssh pub keys

Let's dive in! :zap:

## 1) Install Ubuntu via OVH Console

Get the credentails from David Aronchick, then head over to https://us.ovhcloud.com/manager/#/dedicated/configuration.
Under "Dedicated servers" you'll see 4 entries. Click on one of those (need to repeat this step for each machine), then use the `...` menu to open the installation modal.

<img width="1269" alt="Screenshot 2022-10-11 at 18 04 21" src="https://user-images.githubusercontent.com/4340327/195142902-17249c42-0e27-4925-82d6-f875d254964e.png">

Select `Install one of your templates` and pick `bacalhau - ubuntu2204-server`, confirm.

<img width="627" alt="Screenshot 2022-10-11 at 18 06 25" src="https://user-images.githubusercontent.com/4340327/195143328-3c455971-1248-42ec-aeb9-7808b518a373.png">

OVH console allows you to select only one ssh key (using Enrico's for now), that's not a problem because we'll add the team's keys later on, but to move past this point you need to figure out how to add your key to OVH console so that it's listed in the dropdown menu.
Once you confrim you'll see the progess bar slowing making its way to the right end of the stick.
Go brew one or two ☕ because this step takes really a long time...

<img width="609" alt="Screenshot 2022-10-11 at 18 08 52" src="https://user-images.githubusercontent.com/4340327/195143814-978f5f54-deb5-4d25-a911-9f286920a8c1.png">

After it completes you can finally ssh into the machines with the `ubuntu` user: `ssh ubuntu@51.81.184.xx`

## 2) Configure RAID for data disks

On top of the boot drive, these hosts ship with `6× 3.84TB` NVMe data disks.
Follow the steps below to create a `RAID0` array and persist it upon reboot.

```bash
> lsblk -o NAME,SIZE,FSTYPE,TYPE,MOUNTPOINT

NAME          SIZE FSTYPE            TYPE  MOUNTPOINT
sda         447.1G                   disk
...
nvme0n1       3.5T                   disk
nvme1n1       3.5T                   disk
nvme2n1       3.5T                   disk
nvme3n1       3.5T                   disk
nvme4n1       3.5T                   disk
nvme5n1       3.5T                   disk
```

### Partition disks

Create a partition on the first disk.

```bash
> sudo fdisk /dev/nvme0n1
```

Note `fdisk` is an interactive util so you need to manually follow a number of steps:

1. press `n` for a new partition
1. press enter and confirm all defaults
1. press `t` to select the partiton type
1. insert `29` that is the alias for `29 Linux RAID` for `51.81.184.74` only. For the other 3 hosts use `FD` that stands for `raid`. The machines are different, not sure why.
1. press `w` to write out the partiton to disk

Repeat the steps above for each disk:

```bash
> sudo fdisk /dev/nvme1n1
> sudo fdisk /dev/nvme2n1
> sudo fdisk /dev/nvme3n1
> sudo fdisk /dev/nvme4n1
> sudo fdisk /dev/nvme5n1
```

At this point you should see a partition under each disk:

```bash
> lsblk -o NAME,SIZE,FSTYPE,TYPE,MOUNTPOINT
NAME          SIZE FSTYPE            TYPE  MOUNTPOINT
...
nvme0n1       3.5T                   disk
└─nvme0n1p1   3.5T linux_raid_member part
...
```

### Format partitions

```bash
> sudo mkfs.ext4 /dev/nvme0n1p1
> sudo mkfs.ext4 /dev/nvme1n1p1
> sudo mkfs.ext4 /dev/nvme2n1p1
> sudo mkfs.ext4 /dev/nvme3n1p1
> sudo mkfs.ext4 /dev/nvme4n1p1
> sudo mkfs.ext4 /dev/nvme5n1p1
```

### Create RAID Array

```bash
> sudo mdadm --create \
    --verbose /dev/md0 \
    --level=raid0 \
    --raid-devices=6 \
    /dev/nvme0n1p1 \
    /dev/nvme1n1p1 \
    /dev/nvme2n1p1 \
    /dev/nvme3n1p1 \
    /dev/nvme4n1p1 \
    /dev/nvme5n1p1
```

Confirm Yes upon promtps.
Now a new `md0` array should be visible:

```bash
> cat /proc/mdstat
Personalities : [raid0] [raid1] [linear] [multipath] [raid6] [raid5] [raid4] [raid10]
md3 : active raid1 sdb3[0] sda3[1]
      466619392 blocks super 1.2 [2/2] [UU]
      bitmap: 0/4 pages [0KB], 65536KB chunk

md2 : active raid1 sdb2[1] sda2[0]
      1046528 blocks super 1.2 [2/2] [UU]

md0 : active raid0 nvme0n1p1[0]
      3750604800 blocks super 1.2 512k chunks
```

At this point we've created the array but we still need to make it persistent.

```bash
> sudo mdadm --detail --scan | grep --color=never /dev/md0 | sudo tee -a /etc/mdadm/mdadm.conf
> sudo update-initramfs -u
> sudo mkdir -p /data
> sudo mkfs.ext4 /dev/md0
> sudo mount /dev/md0 /data
```

Check if `mount` succeeded, you should see `/dev/md0` mounted on `/data`:

```bash
> df -h
Filesystem      Size  Used Avail Use% Mounted on
...
/dev/md0        21T   28K  21T   1% /data
...
```

Last step is adding the array to `/etc/fstab` so that it's automagically mounted upon reboot:

```bash
echo '/dev/md0 /data ext4 defaults,nofail,discard 0 0' | sudo tee -a /etc/fstab
```

Ref. https://www.digitalocean.com/community/tutorials/how-to-create-raid-arrays-with-mdadm-on-ubuntu-16-04

## 3) Configure Firewall

We're going to open some ports, [this terraform script](./terraform/main.tf) is the main reference for port numbers.

```bash
> sudo ufw default deny incoming
> sudo ufw allow ssh
> sudo ufw allow 4001 # ipfs swarm
> sudo ufw allow 1234 # bacalhau API
> sudo ufw allow 1235 # bacalhau swarm
> sudo ufw allow 2112 # bacalhau metrics
> sudo ufw allow 9090 # prometheus service
> sudo ufw allow 44443 # nginx is healthy - for running health check scripts
> sudo ufw allow 44444 # nginx node health check scripts
> sudo ufw enable
```

Confrim the last prompt, then check the firewall status:

```bash
> sudo ufw status numbered
Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] 4001                       ALLOW IN    Anywhere
[ 3] 1234                       ALLOW IN    Anywhere
[ 4] 1235                       ALLOW IN    Anywhere
[ 5] 2112                       ALLOW IN    Anywhere
[ 6] 9090                       ALLOW IN    Anywhere
[ 7] 44443                      ALLOW IN    Anywhere
[ 8] 44444                      ALLOW IN    Anywhere
[ 9] 22/tcp (v6)                ALLOW IN    Anywhere (v6)
[10] 4001 (v6)                  ALLOW IN    Anywhere (v6)
[11] 1234 (v6)                  ALLOW IN    Anywhere (v6)
[12] 1235 (v6)                  ALLOW IN    Anywhere (v6)
[13] 2112 (v6)                  ALLOW IN    Anywhere (v6)
[14] 9090 (v6)                  ALLOW IN    Anywhere (v6)
[15] 44443 (v6)                 ALLOW IN    Anywhere (v6)
[16] 44444 (v6)                 ALLOW IN    Anywhere (v6)

```

The firewall setup is loaded at every reboot!

Ref. https://www.digitalocean.com/community/tutorials/how-to-set-up-a-firewall-with-ufw-on-ubuntu-22-04

## 4) Add team's ssh pub keys

First we need the GitHub usernames of the team members whose ssh key we'd like to add.
The [Bacalhau's contributors](https://github.com/bacalhau-project/bacalhau/graphs/contributors) is a good starting point.

```bash
> wget -q --output-document - github.com/enricorotundo.keys >> ~/.ssh/authorized_keys
> wget -q --output-document - github.com/binocarlos.keys >> ~/.ssh/authorized_keys
> wget -q --output-document - github.com/aronchick.keys >> ~/.ssh/authorized_keys
> wget -q --output-document - github.com/lukemarsden.keys >> ~/.ssh/authorized_keys
> wget -q --output-document - github.com/philwinder.keys >> ~/.ssh/authorized_keys
> wget -q --output-document - github.com/wdbaruni.keys >> ~/.ssh/authorized_keys
...
```

Confirm that worked with `cat ~/.ssh/authorized_keys`.

---

That's all folks 🥳
