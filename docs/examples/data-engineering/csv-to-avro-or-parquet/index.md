---
sidebar_label: "csv-to-avro-or-parquet"
sidebar_position: 10
---
# Convert CSV To Parquet Or Arrow

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/csv-to-avro-or-parquet/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering/csv-to-avro-or-parquet/index.ipynb)

## Introduction

Converting from csv to parquet or avro reduces the size of file and allows for faster read and write speeds, using bacalhau you can convert your csv files stored on ipfs or on the web without
The need to download files and install dependencies locally

In this example we will convert a csv file from a url to parquet format and save the converted parquet file to IPFS


## Running Locally​


Installing dependencies



```bash
git clone https://github.com/js-ts/csv_to_avro_or_parquet/
pip3 install -r csv_to_avro_or_parquet/requirements.txt
```

    Looking in indexes: https://pypi.org/simple, https://us-python.pkg.dev/colab-wheels/public/simple/
    Requirement already satisfied: fastavro==1.4.7 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 1)) (1.4.7)
    Requirement already satisfied: numpy==1.21.5 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 22)) (1.21.5)
    Requirement already satisfied: pandas==1.3.5 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 53)) (1.3.5)
    Requirement already satisfied: pyarrow==6.0.1 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 79)) (6.0.1)
    Requirement already satisfied: python-dateutil==2.8.2 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 116)) (2.8.2)
    Requirement already satisfied: pytz==2021.3 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 119)) (2021.3)
    Requirement already satisfied: six==1.16.0 in /usr/local/lib/python3.7/dist-packages (from -r csv_to_avro_or_parquet/requirements.txt (line 122)) (1.16.0)


    Cloning into 'csv_to_avro_or_parquet'...



```python
%cd csv_to_avro_or_parquet
```

    /content/csv_to_avro_or_parquet


Downloading the test dataset



```python
!wget https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv  
```

Running the conversion script

arguments
```
python converter.py <path_to_csv> <path_to_result_file> <extension>
```

Running the script





```bash
python3 src/converter.py ./movies.csv  ./movies.parquet parquet
```

viewing the parquet file


```python
import pandas as pd
pd.read_parquet('./movies.parquet')
```





  <div id="df-1fe0efd6-6153-47d1-be2a-92e7f64de713">
    <div class="colab-df-container">
      <div>
<style scoped>
    .dataframe tbody tr th:only-of-type {
        vertical-align: middle;
    }

    .dataframe tbody tr th {
        vertical-align: top;
    }

    .dataframe thead th {
        text-align: right;
    }
</style>
<table border="1" class="dataframe">
  <thead>
    <tr style="text-align: right;">
      <th></th>
      <th>title</th>
      <th>rating</th>
      <th>year</th>
      <th>runtime</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th>0</th>
      <td>Almost Famous</td>
      <td>R</td>
      <td>2000</td>
      <td>122</td>
    </tr>
    <tr>
      <th>1</th>
      <td>American Pie</td>
      <td>R</td>
      <td>1999</td>
      <td>95</td>
    </tr>
    <tr>
      <th>2</th>
      <td>Back to the Future</td>
      <td>PG</td>
      <td>1985</td>
      <td>116</td>
    </tr>
    <tr>
      <th>3</th>
      <td>Blade Runner</td>
      <td>R</td>
      <td>1982</td>
      <td>117</td>
    </tr>
    <tr>
      <th>4</th>
      <td>Blood for Dracula</td>
      <td>R</td>
      <td>1974</td>
      <td>106</td>
    </tr>
    <tr>
      <th>5</th>
      <td>Blue Velvet</td>
      <td>R</td>
      <td>1986</td>
      <td>120</td>
    </tr>
    <tr>
      <th>6</th>
      <td>The Breakfast Club</td>
      <td>R</td>
      <td>1985</td>
      <td>97</td>
    </tr>
    <tr>
      <th>7</th>
      <td>Clueless</td>
      <td>PG-13</td>
      <td>1995</td>
      <td>97</td>
    </tr>
    <tr>
      <th>8</th>
      <td>Cool Hand Luke</td>
      <td>GP</td>
      <td>1967</td>
      <td>127</td>
    </tr>
    <tr>
      <th>9</th>
      <td>The Craft</td>
      <td>R</td>
      <td>1996</td>
      <td>101</td>
    </tr>
    <tr>
      <th>10</th>
      <td>Doctor Zhivago</td>
      <td>PG-13</td>
      <td>1965</td>
      <td>197</td>
    </tr>
    <tr>
      <th>11</th>
      <td>el Topo</td>
      <td>Not Rated</td>
      <td>1970</td>
      <td>125</td>
    </tr>
    <tr>
      <th>12</th>
      <td>Evil Dead</td>
      <td>NC-17</td>
      <td>1981</td>
      <td>85</td>
    </tr>
    <tr>
      <th>13</th>
      <td>Ghostbusters</td>
      <td>PG</td>
      <td>1984</td>
      <td>105</td>
    </tr>
    <tr>
      <th>14</th>
      <td>Grease</td>
      <td>PG-13</td>
      <td>1978</td>
      <td>110</td>
    </tr>
    <tr>
      <th>15</th>
      <td>Heathers</td>
      <td>R</td>
      <td>1988</td>
      <td>103</td>
    </tr>
    <tr>
      <th>16</th>
      <td>Labyrinth</td>
      <td>PG</td>
      <td>1986</td>
      <td>101</td>
    </tr>
    <tr>
      <th>17</th>
      <td>The Lost Boys</td>
      <td>R</td>
      <td>1987</td>
      <td>97</td>
    </tr>
    <tr>
      <th>18</th>
      <td>Mean Girls</td>
      <td>PG-13</td>
      <td>2004</td>
      <td>97</td>
    </tr>
    <tr>
      <th>19</th>
      <td>Millennium Actress</td>
      <td>PG</td>
      <td>2001</td>
      <td>87</td>
    </tr>
    <tr>
      <th>20</th>
      <td>My Neighbor Totoro</td>
      <td>G</td>
      <td>1988</td>
      <td>86</td>
    </tr>
    <tr>
      <th>21</th>
      <td>Napoleon Dynamite</td>
      <td>PG</td>
      <td>2004</td>
      <td>96</td>
    </tr>
    <tr>
      <th>22</th>
      <td>Pee-wee's Big Adventure</td>
      <td>PG</td>
      <td>1985</td>
      <td>91</td>
    </tr>
    <tr>
      <th>23</th>
      <td>Pretty in Pink</td>
      <td>PG-13</td>
      <td>1986</td>
      <td>97</td>
    </tr>
    <tr>
      <th>24</th>
      <td>The Princess Bride</td>
      <td>PG</td>
      <td>1987</td>
      <td>98</td>
    </tr>
    <tr>
      <th>25</th>
      <td>Psycho</td>
      <td>R</td>
      <td>1960</td>
      <td>109</td>
    </tr>
    <tr>
      <th>26</th>
      <td>Stand by Me</td>
      <td>R</td>
      <td>1986</td>
      <td>89</td>
    </tr>
    <tr>
      <th>27</th>
      <td>Super 8</td>
      <td>PG-13</td>
      <td>2011</td>
      <td>112</td>
    </tr>
    <tr>
      <th>28</th>
      <td>superbad</td>
      <td>R</td>
      <td>2007</td>
      <td>113</td>
    </tr>
    <tr>
      <th>29</th>
      <td>WarGames</td>
      <td>PG</td>
      <td>1983</td>
      <td>114</td>
    </tr>
  </tbody>
</table>
</div>
      <button class="colab-df-convert" onclick="convertToInteractive('df-1fe0efd6-6153-47d1-be2a-92e7f64de713')"
              title="Convert this dataframe to an interactive table."
              style="display:none;">

  <svg xmlns="http://www.w3.org/2000/svg" height="24px"viewBox="0 0 24 24"
       width="24px">
    <path d="M0 0h24v24H0V0z" fill="none"/>
    <path d="M18.56 5.44l.94 2.06.94-2.06 2.06-.94-2.06-.94-.94-2.06-.94 2.06-2.06.94zm-11 1L8.5 8.5l.94-2.06 2.06-.94-2.06-.94L8.5 2.5l-.94 2.06-2.06.94zm10 10l.94 2.06.94-2.06 2.06-.94-2.06-.94-.94-2.06-.94 2.06-2.06.94z"/><path d="M17.41 7.96l-1.37-1.37c-.4-.4-.92-.59-1.43-.59-.52 0-1.04.2-1.43.59L10.3 9.45l-7.72 7.72c-.78.78-.78 2.05 0 2.83L4 21.41c.39.39.9.59 1.41.59.51 0 1.02-.2 1.41-.59l7.78-7.78 2.81-2.81c.8-.78.8-2.07 0-2.86zM5.41 20L4 18.59l7.72-7.72 1.47 1.35L5.41 20z"/>
  </svg>
      </button>

  <style>
    .colab-df-container {
      display:flex;
      flex-wrap:wrap;
      gap: 12px;
    }

    .colab-df-convert {
      background-color: #E8F0FE;
      border: none;
      border-radius: 50%;
      cursor: pointer;
      display: none;
      fill: #1967D2;
      height: 32px;
      padding: 0 0 0 0;
      width: 32px;
    }

    .colab-df-convert:hover {
      background-color: #E2EBFA;
      box-shadow: 0px 1px 2px rgba(60, 64, 67, 0.3), 0px 1px 3px 1px rgba(60, 64, 67, 0.15);
      fill: #174EA6;
    }

    [theme=dark] .colab-df-convert {
      background-color: #3B4455;
      fill: #D2E3FC;
    }

    [theme=dark] .colab-df-convert:hover {
      background-color: #434B5C;
      box-shadow: 0px 1px 3px 1px rgba(0, 0, 0, 0.15);
      filter: drop-shadow(0px 1px 2px rgba(0, 0, 0, 0.3));
      fill: #FFFFFF;
    }
  </style>

      <script>
        const buttonEl =
          document.querySelector('#df-1fe0efd6-6153-47d1-be2a-92e7f64de713 button.colab-df-convert');
        buttonEl.style.display =
          google.colab.kernel.accessAllowed ? 'block' : 'none';

        async function convertToInteractive(key) {
          const element = document.querySelector('#df-1fe0efd6-6153-47d1-be2a-92e7f64de713');
          const dataTable =
            await google.colab.kernel.invokeFunction('convertToInteractive',
                                                     [key], {});
          if (!dataTable) return;

          const docLinkHtml = 'Like what you see? Visit the ' +
            '<a target="_blank" href=https://colab.research.google.com/notebooks/data_table.ipynb>data table notebook</a>'
            + ' to learn more about interactive tables.';
          element.innerHTML = '';
          dataTable['output_type'] = 'display_data';
          await google.colab.output.renderOutput(dataTable, element);
          const docLink = document.createElement('div');
          docLink.innerHTML = docLinkHtml;
          element.appendChild(docLink);
        }
      </script>
    </div>
  </div>




### Building a Docker container (Optional)
Note* you can skip this section entirely and directly go to running on bacalhau

To use Bacalhau, you need to package your code in an appropriate format. The developers have already pushed a container for you to use, but if you want to build your own, you can follow the steps below. You can view a [dedicated container example](../custom-containers/index.md) in the documentation.

### Dockerfile

In this step, you will create a `Dockerfile` to create an image. The `Dockerfile` is a text document that contains the commands used to assemble the image. First, create the `Dockerfile`.

```
FROM python:3.8

RUN apt update && apt install git

RUN git clone https://github.com/js-ts/Sparkov_Data_Generation/

WORKDIR /Sparkov_Data_Generation/

RUN pip3 install -r requirements.txt
```

To Build the docker container run the docker build command

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Please replace

<hub-user> with your docker hub username, If you don’t have a docker hub account Follow these instructions to create docker account, and use the username of the account you created

<repo-name> This is the name of the container, you can name it anything you want

<tag> This is not required but you can use the latest tag

After you have build the container, the next step is to test it locally and then push it docker hub

Now you can push this repository to the registry designated by its name or tag.

```
 docker push <hub-user>/<repo-name>:<tag>
```


After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau

## Running on Bacalhau

After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau

This command is similar to what we have run locally but we change the output directory to the outputs folder so that the results are saved to IPFS

we will show you how you can mount the script from a IPFS as we as from an URL


```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.11 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.3.11
    Server Version: v0.3.11


Mounting the csv file from IPFS


```bash
bacalhau docker run \
-i QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W  \
--wait \
--id-only \
 jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/transactions.csv  ../outputs/transactions.parquet parquet
```

Mounting the csv file from an URL

```
bacalhau docker run \
-u https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv   jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/movies.csv  ../outputs/movies.parquet parquet
```

Running the commands will output a UUID that represents the job that was created. You can check the status of the job with the following command:


```bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 10:19:19 [0m[97;40m 94774248 [0m[97;40m Docker jsacex/csv-to... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmdHJaMmQHs9fE... [0m



Where it says "`Completed `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
bacalhau describe ${JOB_ID}
```

If you see that the job has completed and there are no errors, then you can download the results with the following command:


```bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job '94774248-1d07-4121-aac8-451aca4a636e'...
    Results for job '94774248-1d07-4121-aac8-451aca4a636e' have been written to...
    results


    2022/11/12 10:20:09 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


After the download has finished you should 
see the following contents in results directory


```bash
ls results/combined_results/outputs
```

    transactions.parquet


Viewing the output


```python
import pandas as pd
import os
pd.read_parquet('results/combined_results/outputs/transactions.parquet')
```





  <div id="df-41546dc6-fab7-40c0-ad66-dc7b41fc5400">
    <div class="colab-df-container">
      <div>
<style scoped>
    .dataframe tbody tr th:only-of-type {
        vertical-align: middle;
    }

    .dataframe tbody tr th {
        vertical-align: top;
    }

    .dataframe thead th {
        text-align: right;
    }
</style>
<table border="1" class="dataframe">
  <thead>
    <tr style="text-align: right;">
      <th></th>
      <th>hash</th>
      <th>nonce</th>
      <th>block_hash</th>
      <th>block_number</th>
      <th>transaction_index</th>
      <th>from_address</th>
      <th>to_address</th>
      <th>value</th>
      <th>gas</th>
      <th>gas_price</th>
      <th>input</th>
      <th>block_timestamp</th>
      <th>max_fee_per_gas</th>
      <th>max_priority_fee_per_gas</th>
      <th>transaction_type</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th>0</th>
      <td>0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...</td>
      <td>12</td>
      <td>0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c71...</td>
      <td>483920</td>
      <td>0</td>
      <td>0x1b63142628311395ceafeea5667e7c9026c862ca</td>
      <td>0xf4eced2f682ce333f96f2d8966c613ded8fc95dd</td>
      <td>0</td>
      <td>150853</td>
      <td>50000000000</td>
      <td>0xa9059cbb000000000000000000000000ac4df82fe37e...</td>
      <td>1446561880</td>
      <td>NaN</td>
      <td>NaN</td>
      <td>0</td>
    </tr>
    <tr>
      <th>1</th>
      <td>0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...</td>
      <td>84</td>
      <td>0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c71...</td>
      <td>483920</td>
      <td>1</td>
      <td>0x9b22a80d5c7b3374a05b446081f97d0a34079e7f</td>
      <td>0xf4eced2f682ce333f96f2d8966c613ded8fc95dd</td>
      <td>0</td>
      <td>150853</td>
      <td>50000000000</td>
      <td>0xa9059cbb00000000000000000000000066f183060253...</td>
      <td>1446561880</td>
      <td>NaN</td>
      <td>NaN</td>
      <td>0</td>
    </tr>
    <tr>
      <th>2</th>
      <td>0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...</td>
      <td>88</td>
      <td>0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c71...</td>
      <td>483920</td>
      <td>2</td>
      <td>0x9df428a91ff0f3635c8f0ce752933b9788926804</td>
      <td>0x9e669f970ec0f49bb735f20799a7e7c4a1c274e2</td>
      <td>11000440000000000</td>
      <td>90000</td>
      <td>50000000000</td>
      <td>0x</td>
      <td>1446561880</td>
      <td>NaN</td>
      <td>NaN</td>
      <td>0</td>
    </tr>
    <tr>
      <th>3</th>
      <td>0x05287a561f218418892ab053adfb3d919860988b1945...</td>
      <td>20085</td>
      <td>0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c71...</td>
      <td>483920</td>
      <td>3</td>
      <td>0x2a65aca4d5fc5b5c859090a6c34d164135398226</td>
      <td>0x743b8aeedc163c0e3a0fe9f3910d146c48e70da8</td>
      <td>1530219620000000000</td>
      <td>90000</td>
      <td>50000000000</td>
      <td>0x</td>
      <td>1446561880</td>
      <td>NaN</td>
      <td>NaN</td>
      <td>0</td>
    </tr>
  </tbody>
</table>
</div>
      <button class="colab-df-convert" onclick="convertToInteractive('df-41546dc6-fab7-40c0-ad66-dc7b41fc5400')"
              title="Convert this dataframe to an interactive table."
              style="display:none;">

  <svg xmlns="http://www.w3.org/2000/svg" height="24px"viewBox="0 0 24 24"
       width="24px">
    <path d="M0 0h24v24H0V0z" fill="none"/>
    <path d="M18.56 5.44l.94 2.06.94-2.06 2.06-.94-2.06-.94-.94-2.06-.94 2.06-2.06.94zm-11 1L8.5 8.5l.94-2.06 2.06-.94-2.06-.94L8.5 2.5l-.94 2.06-2.06.94zm10 10l.94 2.06.94-2.06 2.06-.94-2.06-.94-.94-2.06-.94 2.06-2.06.94z"/><path d="M17.41 7.96l-1.37-1.37c-.4-.4-.92-.59-1.43-.59-.52 0-1.04.2-1.43.59L10.3 9.45l-7.72 7.72c-.78.78-.78 2.05 0 2.83L4 21.41c.39.39.9.59 1.41.59.51 0 1.02-.2 1.41-.59l7.78-7.78 2.81-2.81c.8-.78.8-2.07 0-2.86zM5.41 20L4 18.59l7.72-7.72 1.47 1.35L5.41 20z"/>
  </svg>
      </button>

  <style>
    .colab-df-container {
      display:flex;
      flex-wrap:wrap;
      gap: 12px;
    }

    .colab-df-convert {
      background-color: #E8F0FE;
      border: none;
      border-radius: 50%;
      cursor: pointer;
      display: none;
      fill: #1967D2;
      height: 32px;
      padding: 0 0 0 0;
      width: 32px;
    }

    .colab-df-convert:hover {
      background-color: #E2EBFA;
      box-shadow: 0px 1px 2px rgba(60, 64, 67, 0.3), 0px 1px 3px 1px rgba(60, 64, 67, 0.15);
      fill: #174EA6;
    }

    [theme=dark] .colab-df-convert {
      background-color: #3B4455;
      fill: #D2E3FC;
    }

    [theme=dark] .colab-df-convert:hover {
      background-color: #434B5C;
      box-shadow: 0px 1px 3px 1px rgba(0, 0, 0, 0.15);
      filter: drop-shadow(0px 1px 2px rgba(0, 0, 0, 0.3));
      fill: #FFFFFF;
    }
  </style>

      <script>
        const buttonEl =
          document.querySelector('#df-41546dc6-fab7-40c0-ad66-dc7b41fc5400 button.colab-df-convert');
        buttonEl.style.display =
          google.colab.kernel.accessAllowed ? 'block' : 'none';

        async function convertToInteractive(key) {
          const element = document.querySelector('#df-41546dc6-fab7-40c0-ad66-dc7b41fc5400');
          const dataTable =
            await google.colab.kernel.invokeFunction('convertToInteractive',
                                                     [key], {});
          if (!dataTable) return;

          const docLinkHtml = 'Like what you see? Visit the ' +
            '<a target="_blank" href=https://colab.research.google.com/notebooks/data_table.ipynb>data table notebook</a>'
            + ' to learn more about interactive tables.';
          element.innerHTML = '';
          dataTable['output_type'] = 'display_data';
          await google.colab.output.renderOutput(dataTable, element);
          const docLink = document.createElement('div');
          docLink.innerHTML = docLinkHtml;
          element.appendChild(docLink);
        }
      </script>
    </div>
  </div>



