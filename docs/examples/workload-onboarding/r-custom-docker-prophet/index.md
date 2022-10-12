# Building and Running your Custom R Containers on Bacalhau

## **Introduction**

This example will walk you through building Time Series Forecasting using Prophet 

Prophet is a forecasting procedure implemented in R and Python. It is fast and provides completely automated forecasts that can be tuned by hand by data scientists and analysts.




[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/r-custom-docker-prophet/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/r-custom-docker-prophet/index.ipynb)

## **Running the script locally**

Open R studio or R supported IDE

Prophet is a CRAN package so you can use install.packages to install the prophet package

Run this command in console to install prophet


```bash
R -e "install.packages('prophet',dependencies=TRUE, repos='http://cran.rstudio.com/')"
```


After installation is finished

Download the dataset by clicking this link


```bash
wget https://cloudflare-ipfs.com/ipfs/QmZiwZz7fXAvQANKYnt7ya838VPpj4agJt5EDvRYp3Deeo/example_wp_log_R.csv
```


```bash
mkdir outputs
mkdir R
```


```python
%%writefile Saturating-Forecasts.R
library('prophet')

args = commandArgs(trailingOnly=TRUE)
args

input = args[1]
output = args[2]
output1 = args[3]


I <- paste("", input, sep ="")

O <- paste("", output, sep ="")

O1 <- paste("", output1 ,sep ="")


df <- read.csv(I)

df$cap <- 8.5
m <- prophet(df, growth = 'logistic')

future <- make_future_dataframe(m, periods = 1826)
future$cap <- 8.5
fcst <- predict(m, future)
pdf(O)
plot(m, fcst)
dev.off()

df$y <- 10 - df$y
df$cap <- 6
df$floor <- 1.5
future$cap <- 6
future$floor <- 1.5
m <- prophet(df, growth = 'logistic')
fcst <- predict(m, future)
pdf(O1)
plot(m, fcst)
dev.off()
```

Command to run the script



 We provide parameters like name of the input csv dataset

And Path and name of the First and second output which is a graph that is saved when the script is ran



```bash
Rscript Saturating-Forecasts.R "example_wp_log_R.csv" "outputs/output0.pdf" "outputs/output1.pdf"
```


**Setting Up Docker**

In this step you will create a  `Dockerfile` to create your Docker deployment. The `Dockerfile` is a text document that contains the commands used to assemble the image.

First, create the `Dockerfile`.

Dockerfile


```
FROM r-base
RUN R -e "install.packages('prophet',dependencies=TRUE, repos='http://cran.rstudio.com/')"
COPY . R
WORKDIR /R
```


Next, add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included.

What the Dockerfile does


```
FROM r-base
```


We use r-base as the base image 


```
RUN R -e "install.packages('prophet',dependencies=TRUE,repos='http://cran.rstudio.com/')"
```


install packages 


```
COPY . R
```


Copy the contents of your PWD which includes your scripts


```
WORKDIR /R
```


Make the R directory which we copied to be the working directory

Build the container


```
docker build -t <hub-user>/<repo-name>:<tag> .
```


After you have build the container successfully, the next step is to test it locally and then push it docker hub

Before pushing you first need to create a repo which you can create by following the instructions here [https://docs.docker.com/docker-hub/repos/](https://docs.docker.com/docker-hub/repos/)

Now you can push this repository to the registry designated by its name or tag.


```
 docker push <hub-user>/<repo-name>:<tag>
```


After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau


To mount your dataset there are 2 options

Mounting the dataset using the -u or The URL flag


```
bacalhau docker run \
-u https://raw.githubusercontent.com/facebook/prophet/main/examples/example_wp_log_R.csv:/input \
jsace/r-prophet \
-- Rscript Saturating-Forecasts.R "example_wp_log_R.csv" "outputs/output0.pdf" "outputs/output1.pdf"
```


Mounting the dataset using CID


```
bacalhau docker run \
-v QmY8BAftd48wWRYDf5XnZGkhwqgjpzjyUG3hN1se6SYaFt:/example_wp_log_R.csv \
jsace/r-prophet \
-- Rscript Saturating-Forecasts.R "example_wp_log_R.csv" "outputs/output0.pdf" "outputs/output1.pdf"
```



Insalling bacalhau


```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```


```bash
echo $(bacalhau docker run --id-only --wait --wait-timeout-secs 1000 -v QmY8BAftd48wWRYDf5XnZGkhwqgjpzjyUG3hN1se6SYaFt:/example_wp_log_R.csv jsace/r-prophet -- Rscript Saturating-Forecasts.R "example_wp_log_R.csv" "outputs/output0.pdf" "outputs/output1.pdf") > job_id.txt
cat job_id.txt
```


Running the commands will output a UUID (like `54506541-4eb9-45f4-a0b1-ea0aecd34b3e`). This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
bacalhau list --id-filter $(cat job_id.txt)
```


Where it says "`Published `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
bacalhau describe $(cat job_id.txt)
```

Since there is no error we can’t see any error instead we see the state of our job to be complete, that means 
we can download the results!
we create a temporary directory to save our results


```bash
mkdir results
```

To Download the results of your job, run 

---

the following command:


```bash
bacalhau get  $(cat job_id.txt)  --output-dir results
```

After the download has finished you should 
see the following contents in results directory


```bash
ls results/
```


```bash
bacalhau describe $(cat job_id.txt) --spec > job.yaml
```


```bash
cat job.yaml
```


Viewing the output pdf files which are located at volumes/outputs

output0.pdf

![](https://i.imgur.com/dVLgpLA.png)



output1.pdf


![](https://i.imgur.com/qvoJKdB.png)

