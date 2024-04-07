---
sidebar_label: "Python Custom Container"
sidebar_position: 5
---
# Building and Running Custom Python  Container



[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## **Introduction**


In this tutorial example, we will walk you through building your own Python container and running the container on Bacalhau.

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

## 1. Sample Recommendation Dataset

We will be using a simple recommendation script that, when given a movie ID, recommends other movies based on user ratings. Assuming you want recommendations for the movie 'Toy Story' (1995), it will suggest movies from similar categories:

```
Recommendations for Toy Story (1995):
1  :  Toy Story (1995)
58  :  Postino, Il (The Postman) (1994)
3159  :  Fantasia 2000 (1999)
359  :  I Like It Like That (1994)
756  :  Carmen Miranda: Bananas Is My Business (1994)
618  :  Two Much (1996)
48  :  Pocahontas (1995)
2695  :  Boys, The (1997)
2923  :  Citizen's Band (a.k.a. Handle with Care) (1977)
688  :  Operation Dumbo Drop (1995)
```



### Downloading the dataset

Download Movielens1M dataset from this link [https://files.grouplens.org/datasets/movielens/ml-1m.zip](https://files.grouplens.org/datasets/movielens/ml-1m.zip)


```python
!wget https://files.grouplens.org/datasets/movielens/ml-1m.zip
```

In this example, we’ll be using 2 files from the MovieLens 1M dataset: `ratings.dat` and `movies.dat`.After the dataset is downloaded extract the zip and place `ratings.dat` and `movies.dat` into a folder called `input`:

```python
# Extracting the downloaded zip file
!unzip ml-1m.zip
```


```python
#moving  ratings.dat and movies.dat into a folder called 'input'
!mkdir input; mv ml-1m/movies.dat ml-1m/ratings.dat input/
```

The structure of the input directory should be

```
input
├── movies.dat
└── ratings.dat
```

### Installing Dependencies

To create a `requirements.txt` for the Python libraries we’ll be using, run:


```python
%%writefile requirements.txt
numpy
pandas
```

To install the dependencies, run:



```bash
%%bash
pip install -r requirements.txt
```

### Writing the Script

Create a new file called `similar-movies.py` and in it paste the following script


<!-- cspell: disable -->
```python
%%writefile similar-movies.py
# Imports
import numpy as np
import pandas as pd
import argparse
from distutils.dir_util import mkpath
import warnings
warnings.filterwarnings("ignore")
# Read the files with pandas
data = pd.io.parsers.read_csv('input/ratings.dat',
names=['user_id', 'movie_id', 'rating', 'time'],
engine='python', delimiter='::', encoding='latin-1')
movie_data = pd.io.parsers.read_csv('input/movies.dat',
names=['movie_id', 'title', 'genre'],
engine='python', delimiter='::', encoding='latin-1')

# Create the ratings matrix of shape (m×u) with rows as movies and columns as users

ratings_mat = np.ndarray(
shape=((np.max(data.movie_id.values)), np.max(data.user_id.values)),
dtype=np.uint8)
ratings_mat[data.movie_id.values-1, data.user_id.values-1] = data.rating.values

# Normalise matrix (subtract mean off)

normalised_mat = ratings_mat - np.asarray([(np.mean(ratings_mat, 1))]).T

# Compute SVD

normalised_mat = ratings_mat - np.matrix(np.mean(ratings_mat, 1)).T
cov_mat = np.cov(normalised_mat)
evals, evecs = np.linalg.eig(cov_mat)

# Calculate cosine similarity, sort by most similar, and return the top N.

def top_cosine_similarity(data, movie_id, top_n=10):

index = movie_id - 1
# Movie id starts from 1

movie_row = data[index, :]
magnitude = np.sqrt(np.einsum('ij, ij -> i', data, data))
similarity = np.dot(movie_row, data.T) / (magnitude[index] * magnitude)
sort_indexes = np.argsort(-similarity)
return sort_indexes[:top_n]

# Helper function to print top N similar movies
def print_similar_movies(movie_data, movie_id, top_indexes):
print('Recommendations for {0}: \n'.format(
movie_data[movie_data.movie_id == movie_id].title.values[0]))
for id in top_indexes + 1:
print(str(id),' : ',movie_data[movie_data.movie_id == id].title.values[0])


parser = argparse.ArgumentParser(description='Personal information')
parser.add_argument('--k', dest='k', type=int, help='principal components to represent the movies',default=50)
parser.add_argument('--id', dest='id', type=int, help='Id of the movie',default=1)
parser.add_argument('--n', dest='n', type=int, help='No of recommendations',default=10)

args = parser.parse_args()
k = args.k
movie_id = args.id # Grab an id from movies.dat
top_n = args.n

# k = 50
# # Grab an id from movies.dat
# movie_id = 1
# top_n = 10

sliced = evecs[:, :k] # representative data
top_indexes = top_cosine_similarity(sliced, movie_id, top_n)
print_similar_movies(movie_data, movie_id, top_indexes)
```
<!-- cspell: enable -->

What the similar-movies.py script does

1. Read the files with pandas. The code uses Pandas to read data from the files `ratings.dat` and `movies.dat`.

2. Create the ratings matrix of shape (m×u) with rows as movies and columns as user
3. Normalise matrix (subtract mean off). The ratings matrix is normalized by subtracting the mean off.
4. Compute SVD: a singular value decomposition (SVD) of the normalized ratings matrix is performed.
5. Calculate cosine similarity, sort by most similar, and return the top N.
6. Select k principal components to represent the movies, a `movie_id` to find recommendations, and print the `top_n` results.

For further reading on how the script works, go to [Simple Movie Recommender Using SVD | Alyssa](https://alyssaq.github.io/2015/20150426-simple-movie-recommender-using-svd/)


### Running the Script

Running the script `similar-movies.py` using the default values:


```python
! python similar-movies.py
```
You can also use other flags to set your own values.

## 2. Setting Up Docker

We will create a  `Dockerfile` and add the desired configuration to the file. These commands specify how the image will be built, and what extra requirements will be included.


```python
%%writefile Dockerfile
FROM python:3.8
ADD similar-movies.py .
ADD /input input
COPY ./requirements.txt /requirements.txt
RUN pip install -r requirements.txt
```


We will use the `python:3.8` docker image and add our script `similar-movies.py` to copy the script to the docker image, similarly, we also add the `dataset` directory and also the `requirements`, after that run the command to install the dependencies in the image

The final folder structure will look like this:


```
├── Dockerfile
├── input
│   ├── movies.dat
│   └── ratings.dat
├── requirements.txt
└── similar-movies.py
```


:::info
See more information on how to containerize your script/app [here](https://docs.docker.com/get-started/02_our_app/)
:::

### Build the container

We will run `docker build` command to build the container:

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace:

**`hub-user`** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create a docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

**`repo-name`** with the name of the container, you can name it anything you want

**`tag`** this is not required, but you can use the `latest` tag

In our case:

```bash
docker build -t jsace/python-similar-movies .
```

### Push the container

Next, upload the image to the registry. This can be done by using the `Docker hub username`, `repo name` or `tag`.

```
docker push <hub-user>/<repo-name>:<tag>
```

In our case

```bash
docker push jsace/python-similar-movies
```

## 3. Running a Bacalhau Job

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. You can submit a Bacalhau job by running your container on Bacalhau with default or custom parameters.

### Running the Container with Default Parameters

To submit a Bacalhau job by running your container on Bacalhau with default parameters, run the following Bacalhau command:


```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    jsace/python-similar-movies \
    -- python similar-movies.py
```

### Structure of the command

`bacalhau docker run`: call to Bacalhau

`jsace/python-similar-movies`: the name and of the docker image we are using

`-- python similar-movies.py`: execute the Python script

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

### Running the Container with Custom Parameters

To submit a Bacalhau job by running your container on Bacalhau with custom parameters, run the following Bacalhau command:


```
bacalhau docker run \
    jsace/python-similar-movies \
    -- python similar-movies.py --k 50 --id 10 --n 10
```

### Structure of the command

`bacalhau docker run`: call to Bacalhau

`jsace/python-similar-movies`: the name of the docker image we are using

`-- python similar-movies.py --k 50 --id 10 --n 10`: execute the python script. The script will use Singular Value Decomposition (SVD) and cosine similarity to find 10 movies most similar to the one with identifier 10, using 50 principal components.

## 4. Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

**Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory (`results`) and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## 5. Viewing your Job Output

To view the file, run the following command:


```python
!cat results/stdout # displays the contents of the file
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
