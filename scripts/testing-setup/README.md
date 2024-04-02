# Instructions for setup

- **May need to build image - if so**
  - `cd build-ami`
  - `sudo apt install packer`
  - `packer build package.json.pkr.hcl`
- Install terraform - `sudo apt install terraform`
- Update links in variables.tf to be correct - all are fine by default, but make sure it points to your public/private keypair. Make sure to check your ami (if you built a new one)
- Run `./setup_cluster.sh`

That's it!