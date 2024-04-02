---
sidebar_label: "Training Tensorflow Model"
sidebar_position: 3
---
# Training Tensorflow Model



[TensorFlow](https://www.tensorflow.org/) is an open-source machine learning software library, TensorFlow is used to train neural networks. Expressed in the form of stateful dataflow graphs, each node in the graph
represents the operations performed by neural networks on multi-dimensional arrays. These multi-dimensional arrays are commonly known as “tensors”, hence the name TensorFlow. In this example, we will be training a MNIST model.

## TL;DR

```bash
bacalhau docker run \
  --wait \
  --id-only \
  -w /inputs  \
  -i https://gist.githubusercontent.com/js-ts/e7d32c7d19ffde7811c683d4fcb1a219/raw/ff44ac5b157d231f464f4d43ce0e05bccb4c1d7b/train.py \
  -i https://storage.googleapis.com/tensorflow/tf-keras-datasets/mnist.npz \
  tensorflow/tensorflow \
  -- python train.py
```

## Training TensorFlow models Locally
This section is from  [TensorFlow 2 quickstart for beginners](https://colab.research.google.com/github/tensorflow/docs/blob/master/site/en/tutorials/quickstart/beginner.ipynb)

### TensorFlow 2 quickstart for beginners

This short introduction uses [Keras](https://www.tensorflow.org/guide/keras/overview) to:

1. Load a prebuilt dataset.
1. Build a neural network machine learning model that classifies images.
2. Train this neural network.
3. Evaluate the accuracy of the model.

### Set up TensorFlow

Import TensorFlow into your program to check whether it is installed
```python
import tensorflow as tf
import os
print("TensorFlow version:", tf.__version__)
```


```bash
mkdir /inputs
wget https://storage.googleapis.com/tensorflow/tf-keras-datasets/mnist.npz -O /inputs/mnist.npz
```


```python
mnist = tf.keras.datasets.mnist

CWD = '' if os.getcwd() == '/' else os.getcwd()
(x_train, y_train), (x_test, y_test) = mnist.load_data('/inputs/mnist.npz')
x_train, x_test = x_train / 255.0, x_test / 255.0
```

### Build a machine-learning model

Build a `tf.keras.Sequential` model by stacking layers.


```python
model = tf.keras.models.Sequential([
  tf.keras.layers.Flatten(input_shape=(28, 28)),
  tf.keras.layers.Dense(128, activation='relu'),
  tf.keras.layers.Dropout(0.2),
  tf.keras.layers.Dense(10)
])
```

For each example, the model returns a vector of [logits](https://developers.google.com/machine-learning/glossary#logits) or [log-odds](https://developers.google.com/machine-learning/glossary#log-odds) scores, one for each class.


```python
predictions = model(x_train[:1]).numpy()
predictions
```

The `tf.nn.softmax` function converts these logits to *probabilities* for each class:


```python
tf.nn.softmax(predictions).numpy()
```

Note: It is possible to bake the `tf.nn.softmax` function into the activation function for the last layer of the network. While this can make the model output more directly interpretable, this approach is discouraged as it's impossible to provide an exact and numerically stable loss calculation for all models when using a softmax output.

Define a loss function for training using `losses.SparseCategoricalCrossentropy`, which takes a vector of logits and a `True` index and returns a scalar loss for each example.


```python
loss_fn = tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True)
```

This loss is equal to the negative log probability of the true class: The loss is zero if the model is sure of the correct class.

This untrained model gives probabilities close to random (1/10 for each class), so the initial loss should be close to `-tf.math.log(1/10) ~= 2.3`.


```python
loss_fn(y_train[:1], predictions).numpy()
```

Before you start training, configure and compile the model using Keras `Model.compile`. Set the [`optimizer`](https://www.tensorflow.org/api_docs/python/tf/keras/optimizers) class to `adam`, set the `loss` to the `loss_fn` function you defined earlier, and specify a metric to be evaluated for the model by setting the `metrics` parameter to `accuracy`.


```python
model.compile(optimizer='adam',
              loss=loss_fn,
              metrics=['accuracy'])
```

### Train and evaluate your model

Use the `Model.fit` method to adjust your model parameters and minimize the loss:


```python
model.fit(x_train, y_train, epochs=5)
```

The `Model.evaluate` method checks the models performance, usually on a "[Validation-set](https://developers.google.com/machine-learning/glossary#validation-set)" or "[Test-set](https://developers.google.com/machine-learning/glossary#test-set)".


```python
model.evaluate(x_test,  y_test, verbose=2)
```

The image classifier is now trained to ~98% accuracy on this dataset. To learn more, read the [TensorFlow tutorials](https://www.tensorflow.org/tutorials/).

If you want your model to return a probability, you can wrap the trained model, and attach the softmax to it:


```python
probability_model = tf.keras.Sequential([
  model,
  tf.keras.layers.Softmax()
])
```


```python
probability_model(x_test[:5])
```


```bash
mkdir /outputs
```

The following method can be used to save the model as a checkpoint


```python
model.save_weights('/outputs/checkpoints/my_checkpoint')
```


```bash
ls /outputs/
```



## Running on bacalhau


The dataset and the script are mounted to the TensorFlow container using an URL we then run the script inside the container

### Structure of the command

Let's look closely at the command below:

1. `export JOB_ID=$( ... )` exports the job ID as environment variable
1. `bacalhau docker run`: call to bacalhau
1. The `-i  https://gist.githubusercontent.com/js-ts/e7d32c7d19ffde7811c683d4fcb1a219/raw/ff44ac5b157d231f464f4d43ce0e05bccb4c1d7b/train.py` flag is used to mount the training script
1. The `-i https://storage.googleapis.com/tensorflow/tf-keras-datasets/mnist.npz` flag is used to mount the dataset
1. `tensorflow/tensorflow`: the name and the tag of the docker image we are using
1. `python train.py`: command to execute the script

By default whatever URL you mount using the `-i` flag gets mounted at the path `/inputs` so we choose that as our input directory `-w /inputs`

```bash
export JOB_ID=$(bacalhau docker run \
  --wait \
  --id-only \
  -w /inputs  \
  -i https://gist.githubusercontent.com/js-ts/e7d32c7d19ffde7811c683d4fcb1a219/raw/ff44ac5b157d231f464f4d43ce0e05bccb4c1d7b/train.py \
  -i https://storage.googleapis.com/tensorflow/tf-keras-datasets/mnist.npz \
  tensorflow/tensorflow \
  -- python train.py)
```

```bash
bacalhau list --id-filter ${JOB_ID}
```
When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

### Declarative job description

The same job can be presented in the [declarative](../../../setting-up/jobs/job-specification/job.md) format. In this case, the description will look like this:

```yaml
name: Training ML model using tensorflow
type: batch
count: 1
tasks:
  - name: My main task
    Engine:
      type: docker
      params:
        WorkingDirectory: "/inputs"
        Image: "tensorflow/tensorflow" 
        Entrypoint:
          - /bin/bash
        Parameters:
          - -c
          - python train.py
    InputSources:
      - Source:
          Type: urlDownload
          Params:
            URL: https://storage.googleapis.com/tensorflow/tf-keras-datasets/mnist.npz
        Target: /inputs
      - Source:
          Type: urlDownload
          Params:
            URL: https://gist.githubusercontent.com/js-ts/e7d32c7d19ffde7811c683d4fcb1a219/raw/ff44ac5b157d231f464f4d43ce0e05bccb4c1d7b/train.py
        Target: /inputs
    Resources:
      GPU: "1"
```

The job description should be saved in `.yaml` format, e.g. `tensorflow.yaml`, and then run with the command:
```bash
bacalhau job run tensorflow.yaml
```

## Checking the State of your Jobs

### Job status

You can check the status of the job using `bacalhau list`.


```bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Completed`, that means the job is done, and we can get the results.

### Job information

You can find out more information about your job by using `bacalhau describe`.


```bash
bacalhau describe ${JOB_ID}
```

### Job download

You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

After the download has finished you should see the following contents in results directory

## Viewing your Job Output

Now you can find the file in the `results/outputs` folder. To view it, run the following command:


```bash
cat results/outputs/
```
