# Production-grade Deployment

This page contains complimentary steps for the [official GCP (GKE) setup](https://docs.flyte.org/en/v1.0.0/deployment/gcp/manual.html). For a complete setup you may also want to take a look at [this page](https://docs.flyte.org/en/latest/deployment/deployment/cloud_production.html).

## Certificate manager

Please proceed by following all GCP (GKE) instructions linked above up to the "SSL Certificate" section.
You shall use the (updated) manifest below.

cert-issuer.yaml:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt-production
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: issue-email-id
    privateKeySecretRef:
      name: letsencrypt-production
    solvers:
    - selector: {}
      http01:
        ingress:
          class: nginx
```

You need to use a newer version than the one in the official docs, and add the `installCRDs=true` too.

```console
$ helm repo add jetstack https://charts.jetstack.io
$ helm repo update
$ helm install cert-manager --namespace flyte --create-namespace --version v1.12.3 jetstack/cert-manager --set installCRDs=true
$ kubectl apply --namespace=flyte -f cert-issuer.yaml
```

Move on with the official instructions up to the "Installing Flyte" section.

## Bacalhau Agent

Before installing Flyte with the provided Helm chart, install the Bacalhau Agent in the `flyte` namespace.

bacalhau-agent-deployment.yaml:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
    meta.helm.sh/release-name: flyte
    meta.helm.sh/release-namespace: flyte
  labels:
    app.kubernetes.io/instance: flyte
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: flyteagent
  name: bacalhau-flyteagent
  namespace: flyte
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/instance: flyte
      app.kubernetes.io/name: bacalhau-flyteagent
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: flyte
        app.kubernetes.io/name: bacalhau-flyteagent
    spec:
      containers:
      - command:
        - pyflyte
        - serve
        image: docker.io/winderresearch/flytekit-bacalhau:latest
        imagePullPolicy: IfNotPresent
        name: bacalhau-flyteagent
        ports:
        - containerPort: 8000
          name: agent-grpc
          protocol: TCP
        resources:
          limits:
            cpu: 500m
            ephemeral-storage: 100Mi
            memory: 500Mi
          requests:
            cpu: 10m
            ephemeral-storage: 50Mi
            memory: 50Mi
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: flyteagent
      serviceAccountName: flyteagent
      terminationGracePeriodSeconds: 30
```

bacalhau-agent-svc.yaml:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    cloud.google.com/neg: '{"ingress":true}'
    meta.helm.sh/release-name: flyte
    meta.helm.sh/release-namespace: flyte
    projectcontour.io/upstream-protocol.h2c: grpc
  labels:
    app.kubernetes.io/instance: flyte
    app.kubernetes.io/name: bacalhau-flyteagent
    helm.sh/chart: flyte-core-v1.8.1
  name: bacalhau-flyteagent
  namespace: flyte
spec:
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: agent-grpc
    port: 8000
    protocol: TCP
    targetPort: agent-grpc
  selector:
    app.kubernetes.io/instance: flyte
    app.kubernetes.io/name: bacalhau-flyteagent
  sessionAffinity: None
```

## Install Flyte

The official docs tell you to download a yaml file, instead use the one below which adds the Bacalhau Agent config as well as a number of minor fixes:

<details>
  <summary>values-gcp.yaml:</summary>

```yaml
Release:
  Name: <RELEASE-NAME>

userSettings:
  googleProjectId: <PROJECT-ID>
  dbHost: <CLOUD-SQL-IP>
  dbPassword: <DBPASSWORD>
  bucketName: <BUCKETNAME>
  hostName: <HOSTNAME>

#
# FLYTEADMIN
#

flyteadmin:
  extraArgs:
    - --auth.disableForGrpc
    - --auth.disableForHttp
    - --logger.level=5
  initialProjects:
    - flytebacalhau
  replicaCount: 1
  serviceAccount:
    # -- If the service account is created by you, make this false, else a new service account will be created and the flyteadmin role will be added
    # you can change the name of this role
    create: true
    annotations:
      # Needed for gcp workload identity to function
      # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
      iam.gke.io/gcp-service-account: gsa-flyteadmin@{{ .Values.userSettings.googleProjectId }}.iam.gserviceaccount.com
  resources:
    limits:
      cpu: 500m
      ephemeral-storage: 2Gi
      memory: 1G
    requests:
      cpu: 500m
      ephemeral-storage: 2Gi
      memory: 1G
  service:
    annotations:
      # Required for the ingress to properly route grpc traffic to grpc port
      cloud.google.com/app-protocols: '{"grpc":"HTTP2"}'
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: flyteadmin
          topologyKey: kubernetes.io/hostname

#
# DATACATALOG
#

datacatalog:
  replicaCount: 1
  serviceAccount:
    # -- If the service account is created by you, make this false, else a new service account will be created and the iam-role-flyte will be added
    # you can change the name of this role
    create: true
    annotations:
      # Needed for gcp workload identity to function
      # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
      iam.gke.io/gcp-service-account: gsa-datacatalog@{{ .Values.userSettings.googleProjectId }}.iam.gserviceaccount.com
  resources:
    limits:
      cpu: 500m
      ephemeral-storage: 2Gi
    requests:
      cpu: 50m
      ephemeral-storage: 2Gi
      memory: 200Mi
  service:
    annotations:
      # Required for the ingress to properly route grpc traffic to grpc port
      cloud.google.com/app-protocols: '{"grpc":"HTTP2"}'
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: datacatalog
          topologyKey: kubernetes.io/hostname

#
# FLYTEPROPELLER
#

flytepropeller:
  replicaCount: 1
  manager: false
  serviceAccount:
    # -- If the service account is created by you, make this false, else a new service account will be created and the iam-role-flyte will be added
    # you can change the name of this role
    create: true
    annotations:
      # Needed for gcp workload identity to function
      # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
      iam.gke.io/gcp-service-account: gsa-flytepropeller@{{ .Values.userSettings.googleProjectId }}.iam.gserviceaccount.com
  resources:
    limits:
      cpu: 500m
      ephemeral-storage: 2Gi
      memory: 1Gi
    requests:
      cpu: 50m
      ephemeral-storage: 2Gi
      memory: 1Gi
  cacheSizeMbs: 1024
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: flytepropeller
          topologyKey: kubernetes.io/hostname

#
# FLYTE_AGENT
#
flyteagent:
  enabled: true

#
# FLYTECONSOLE
#

flyteconsole:
  replicaCount: 1
  resources:
    limits:
      cpu: 500m
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: flyteconsole
          topologyKey: kubernetes.io/hostname

# --
# Flyte uses a cloud hosted Cron scheduler to run workflows on a schedule. The following module is optional. Without,
# this module, you will not have scheduled launchplans/workflows.
workflow_scheduler:
  enabled: true
  type: native

# --
# Workflow notifications module is an optional dependency. Flyte uses cloud native pub-sub systems to notify users of
# various events in their workflows
workflow_notifications:
  enabled: false

#
# COMMON
#

common:
  ingress:
    host: "{{ .Values.userSettings.hostName }}"
    tls:
      enabled: false
    annotations:
      kubernetes.io/ingress.class: nginx
      nginx.ingress.kubernetes.io/ssl-redirect: "true"
      cert-manager.io/issuer: "letsencrypt-production"
    # --- separateGrpcIngress puts GRPC routes into a separate ingress if true. Required for certain ingress controllers like nginx.
    separateGrpcIngress: true
    # --- Extra Ingress annotations applied only to the GRPC ingress. Only makes sense if `separateGrpcIngress` is enabled.
    separateGrpcIngressAnnotations:
      nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
  databaseSecret:
    name: db-pass
    secretManifest:
      # -- Leave it empty if your secret already exists
      # Else you can create your own secret object. You can use Kubernetes secrets, else you can configure external secrets
      # For external secrets please install Necessary dependencies, like, of your choice
      # - https://github.com/hashicorp/vault
      # - https://github.com/godaddy/kubernetes-external-secrets
      apiVersion: v1
      kind: Secret
      metadata:
        name: db-pass
      type: Opaque
      stringData:
        # -- If using plain text you can provide the password here
        pass.txt: "{{ .Values.userSettings.dbPassword }}"

# -----------------------------------------------------
# Core dependencies that should be configured for Flyte to work on any platform
# Specifically 2 - Storage (s3, gcs etc), Production RDBMS - Aurora, CloudSQL etc
# ------------------------------------------------------
#
# STORAGE SETTINGS
#

storage:
  # -- Sets the storage type. Supported values are sandbox, s3, gcs and custom.
  type: gcs
  # -- bucketName defines the storage bucket flyte will use. Required for all types except for sandbox.
  bucketName: "{{ .Values.userSettings.bucketName }}"
  # -- settings for storage type s3
  gcs:
    # -- GCP project ID. Required for storage type gcs.
    projectId: "{{ .Values.userSettings.googleProjectId }}"

db:
  datacatalog:
    database:
      port: 5432
      # -- Create a user called flyteadmin
      username: flyteadmin
      host: "{{ .Values.userSettings.dbHost }}"
      # -- Create a DB called datacatalog (OR change the name here)
      dbname: flyteadmin
      passwordPath: /etc/db/pass.txt

  admin:
    database:
      port: 5432
      # -- Create a user called flyteadmin
      username: flyteadmin
      host: "{{ .Values.userSettings.dbHost }}"
      # -- Create a DB called flyteadmin (OR change the name here)
      dbname: flyteadmin
      passwordPath: /etc/db/pass.txt

#
# CONFIGMAPS
#

configmap:
  adminServer:
    server:
      httpPort: 8088
      grpcPort: 8089
      security:
        secure: false
        useAuth: false
        allowCors: true
        allowedOrigins:
          # Accepting all domains for Sandbox installation
          - "*"
        allowedHeaders:
          - "Content-Type"

  task_resource_defaults:
    task_resources:
      defaults:
        cpu: 500m
        memory: 1G
        storage: 1G
      limits:
        storage: 2000Mi

  # Adds the remoteData config setting
  remoteData:
    remoteData:
      region:
      scheme: "gcs"
      signedUrls:
        durationMinutes: 3

  # Adds the namespace mapping to default to only domain name instead of project-domain in case of GCP
  namespace_config:
    namespace_mapping:
      template: "{{ domain }}"

  core:
    propeller:
      rawoutput-prefix: "gs://{{ .Values.userSettings.bucketName }}/"
      workers: 40
      gc-interval: 12h
      max-workflow-retries: 50
      kube-client-config:
        qps: 100
        burst: 25
        timeout: 30s
      queue:
        sub-queue:
          type: bucket
          rate: 100
          capacity: 1000

  enabled_plugins:
    # -- Tasks specific configuration [structure](https://pkg.go.dev/github.com/flyteorg/flytepropeller/pkg/controller/nodes/task/config#GetConfig)
    tasks:
      # -- Plugins configuration, [structure](https://pkg.go.dev/github.com/flyteorg/flytepropeller/pkg/controller/nodes/task/config#TaskPluginConfig)
      task-plugins:
        # -- [Enabled Plugins](https://pkg.go.dev/github.com/lyft/flyteplugins/go/tasks/config#Config). Enable sagemaker*, athena if you install the backend
        # plugins
        enabled-plugins:
          - container
          - sidecar
          - k8s-array
          - bigquery
          - agent-service
        default-for-task-types:
          container: container
          sidecar: sidecar
          container_array: k8s-array
          bigquery_query_job_task: bigquery
          bacalhau_task: agent-service

    agent-service:
      supportedTaskTypes:
        - default_task
        - bacalhau_task
      # By default, all the request will be sent to the default agent.
      defaultAgent:
        endpoint: "dns:///flyteagent.flyte.svc.cluster.local:8000"
        insecure: true
        timeouts:
          GetTask: 200ms
        defaultTimeout: 50ms
      agents:
        bacalhau_agent:
          endpoint: "dns:///bacalhau-flyteagent.flyte.svc.cluster.local:8000"
          insecure: true
          defaultServiceConfig: '{"loadBalancingConfig": [{"round_robin":{}}]}'
          timeouts:
            GetTask: 100ms
          defaultTimeout: 20ms
      agentForTaskTypes:
        # It will override the default agent for custom_task, which means propeller will send the request to this agent.
        - bacalhau_task: bacalhau_agent

  # -- Section that configures how the Task logs are displayed on the UI. This has to be changed based on your actual logging provider.
  # Refer to [structure](https://pkg.go.dev/github.com/lyft/flyteplugins/go/tasks/logs#LogConfig) to understand how to configure various
  # logging engines
  task_logs:
    plugins:
      logs:
        kubernetes-enabled: false
        # Enable GCP stackdriver integration for log display
        stackdriver-enabled: true
        stackdriver-logresourcename: k8s_container
      k8s-array:
        logs:
          config:
            stackdriver-enabled: true
            stackdriver-logresourcename: k8s_container

# ----------------------------------------------------------------
# Optional Modules
# Flyte built extensions that enable various additional features in Flyte.
# All these features are optional, but are critical to run certain features
# ------------------------------------------------------------------------

# -- Configuration for the Cluster resource manager component. This is an optional component, that enables automatic
# cluster configuration. This is useful to set default quotas, manage namespaces etc that map to a project/domain
cluster_resource_manager:
  # -- Enables the Cluster resource manager component
  enabled: true
  # -- Starts the cluster resource manager in standalone mode with requisite auth credentials to call flyteadmin service endpoints
  standalone_deploy: false
  config:
    cluster_resources:
      customData:
        - production:
            - projectQuotaCpu:
                value: "5"
            - projectQuotaMemory:
                value: "4000Mi"
            - gsa:
                value: gsa-production@{{ .Values.userSettings.googleProjectId }}.iam.gserviceaccount.com
        - staging:
            - projectQuotaCpu:
                value: "2"
            - projectQuotaMemory:
                value: "3000Mi"
            - gsa:
                value: gsa-staging@{{ .Values.userSettings.googleProjectId }}.iam.gserviceaccount.com
        - development:
            - projectQuotaCpu:
                value: "2"
            - projectQuotaMemory:
                value: "3000Mi"
            - gsa:
                value: gsa-development@{{ .Values.userSettings.googleProjectId }}.iam.gserviceaccount.com

  templates:
    # -- Template for namespaces resources
    - key: aa_namespace
      value: |
        apiVersion: v1
        kind: Namespace
        metadata:
          name: {{ namespace }}
        spec:
          finalizers:
          - kubernetes

    # -- Patch default service account
    - key: aab_default_service_account
      value: |
        apiVersion: v1
        kind: ServiceAccount
        metadata:
          name: default
          namespace: {{ namespace }}
          annotations:
            # Needed for gcp workload identity to function
            # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
            iam.gke.io/gcp-service-account: {{ gsa }}

    - key: ab_project_resource_quota
      value: |
        apiVersion: v1
        kind: ResourceQuota
        metadata:
          name: project-quota
          namespace: {{ namespace }}
        spec:
          hard:
            limits.cpu: {{ projectQuotaCpu }}
            limits.memory: {{ projectQuotaMemory }}

#
# SPARKOPERATOR
#

sparkoperator:
  enabled: false
```

</details>


## Example API Call

```shell
$ curl -GET \
    '<HOSTNAME>:80/api/v1/active_launch_plans/flytebacalhau/development?limit=32&token=32' \
    -H 'accept: application/json'
```

## Side notes:

- `<HOSTNAME>` must be a FQDN and registred as a valid domain 
- CloudFlare: https://docs.flyte.org/en/v1.0.0/community/troubleshoot.html#troubles-with-flytectl-commands-with-cloudflare-dns
