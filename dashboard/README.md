## local development

```
export DOCKER_REGISTRY=${DOCKER_REGISTRY:=gcr.io}
export GCP_PROJECT_ID=${GCP_PROJECT_ID:=bacalhau-production}
export IMAGE_FRONTEND=$DOCKER_REGISTRY/$GCP_PROJECT_ID/dashboard-frontend:dev
export IMAGE_API=$DOCKER_REGISTRY/$GCP_PROJECT_ID/dashboard-api:dev

<from top level bacalhau directory>
docker build -t $IMAGE_FRONTEND dashboard/frontend
docker build -t $IMAGE_API -f Dockerfile.dashboard .

<from dashboard directory>
export POSTGRES_DATA_DIR=$(pwd)/pgdata
export JWT_SECRET=a1b2c3d4
export PEER_CONNECT=/ip4/172.17.0.1/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL

export POSTGRES_PASSWORD=secret
docker-compose up -d
open http://localhost:8080
```

restarting:
```
docker-compose down; docker-compose up -d
```
rebuilding:
```
(cd .. ; docker build -t $IMAGE_API -f Dockerfile.dashboard .) && docker-compose down && docker-compose up -d && docker logs -f dashboard_api_1
```
^ from the dashboard folder


## deploy to production

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
