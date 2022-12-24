## dev

Deploy VM:

```bash
cd bacalhau/dashboard/terraform
terraform init
terraform apply -var-file prod.tfvars
```

Build images:

```bash
cd bacalhau
bash dashboard/scripts/deploy.sh build:api
bash dashboard/scripts/deploy.sh build:frontend
# copy the images
gcloud compute ssh dashboard-vm-default-0
cd /data/dashboard
vi .env
# paste the images and save
```

Start and stop stack:

```bash
gcloud compute ssh dashboard-vm-default-0
cd /data/dashboard
sudo docker-compose stop
sudo docker-compose up -d
```

Production postgres:

```bash
gcloud compute ssh dashboard-vm-default-0
docker ps -a
# copy the id of the postgres container
docker exec -ti <id> psql --user postgres
```

Get TLS cert:

```bash
sudo apt install certbot python3-certbot-nginx
```

copy the `certbot/nginx.conf` file to the vm `/etc/nginx/sites-available/dashboard.bacalhau.org` and delete the default file

```bash
sudo systemctl stop nginx
sudo systemctl start nginx
sudo certbot --nginx -d dashboard.bacalhau.org
```
