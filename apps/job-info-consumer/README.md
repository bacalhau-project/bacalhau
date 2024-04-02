## Job Info Consumer
This application consumes job info events from a gossipsub topic and feeds them into a postgres database.

Deploy VM:

```bash
cd bacalhau/apps/job-info-consumer/terraform
terraform init
terraform apply -var-file prod.tfvars
```

Build and push images:

```bash
cd bacalhau/apps/job-info-consumer
make docker-push
make restart # to restart the service
```

Update docker compose and config:
```bash
# update docker compose
make docker-compose-push

# update config
gcloud compute ssh dashboard-vm-default-0
cd /data/dashboard
vi .env

# restart
sudo docker-compose down && sudo docker-compose up -d
```


Production postgres:

```bash
gcloud compute ssh dashboard-vm-default-0
docker ps -a
# copy the id of the postgres container
docker exec -ti <id> psql --user postgres
```
