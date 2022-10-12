# Speech Recognition using Whisper




# **Introduction**

Whisper is an automatic speech recognition (ASR) system trained on 680,000 hours of multilingual and multitask supervised data collected from the web. We show that the use of such a large and diverse dataset leads to improved robustness to accents, background noise and technical language. Moreover, it enables transcription in multiple languages, as well as translation from those languages into English. We are open-sourcing models and inference code to serve as a foundation for building useful applications and for further research on robust speech processing.

In this example we will transcribe an audio clip locally, containerize the script and then 
Run the container on bacalhau

The advantages of using bacalhau over managed Automatic Speech Recognition services is that you can run your own containers which can scale to do batch process petabytes of Videos, Audio for automatic speech recognition, Using our sharding feature you can do distributed inference very easily and if you have the data stored on IPFS you don't need to move the data you can do compute where the data is located, and the cost of compute is much cheaper than managed services 



# Running whisper locally

Installing dependecies like Whisper, torch, pandas


```bash
pip install git+https://github.com/openai/whisper.git
pip install torch==1.10.1
pip install pandas
sudo apt update && sudo apt install ffmpeg
```

## Running the script

Before we create and run the script we need a sample audio file to test the code

for that we Download a sample audio clip




```bash
wget https://github.com/js-ts/hello/raw/main/hello.mp3
```

    --2022-10-10 03:16:18--  https://github.com/js-ts/hello/raw/main/hello.mp3
    Resolving github.com (github.com)... 20.205.243.166
    Connecting to github.com (github.com)|20.205.243.166|:443... connected.
    HTTP request sent, awaiting response... 302 Found
    Location: https://raw.githubusercontent.com/js-ts/hello/main/hello.mp3 [following]
    --2022-10-10 03:16:19--  https://raw.githubusercontent.com/js-ts/hello/main/hello.mp3
    Resolving raw.githubusercontent.com (raw.githubusercontent.com)... 185.199.108.133, 185.199.109.133, 185.199.110.133, ...
    Connecting to raw.githubusercontent.com (raw.githubusercontent.com)|185.199.108.133|:443... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 10063 (9.8K) [audio/mpeg]
    Saving to: ‘hello.mp3’
    
         0K .........                                             100% 52.3M=0s
    
    2022-10-10 03:16:19 (52.3 MB/s) - ‘hello.mp3’ saved [10063/10063]
    



We will create a script that accepts parameters like input file path, output file path, temperature etc. and set the default parameters, if provided a mp4 file convert it to a .wav file

and also save the transcript in various formats, after that we load the large model

then pass it the required parameters, this model is not only limited to english and transcription

It supports a lots of other languages and also does translation, to 

![](https://i.imgur.com/ALFe4qJ.png)



```python
%%writefile openai-whisper.py
import argparse
import os
import sys
import warnings
import whisper
from pathlib import Path
import subprocess
import torch
import shutil
import numpy as np
parser = argparse.ArgumentParser(description="OpenAI Whisper Automatic Speech Recognition")
parser.add_argument("-l",dest="audiolanguage", type=str,help="Language spoken in the audio, use Auto detection to let Whisper detect the language. Select from the following languages['Auto detection', 'Afrikaans', 'Albanian', 'Amharic', 'Arabic', 'Armenian', 'Assamese', 'Azerbaijani', 'Bashkir', 'Basque', 'Belarusian', 'Bengali', 'Bosnian', 'Breton', 'Bulgarian', 'Burmese', 'Castilian', 'Catalan', 'Chinese', 'Croatian', 'Czech', 'Danish', 'Dutch', 'English', 'Estonian', 'Faroese', 'Finnish', 'Flemish', 'French', 'Galician', 'Georgian', 'German', 'Greek', 'Gujarati', 'Haitian', 'Haitian Creole', 'Hausa', 'Hawaiian', 'Hebrew', 'Hindi', 'Hungarian', 'Icelandic', 'Indonesian', 'Italian', 'Japanese', 'Javanese', 'Kannada', 'Kazakh', 'Khmer', 'Korean', 'Lao', 'Latin', 'Latvian', 'Letzeburgesch', 'Lingala', 'Lithuanian', 'Luxembourgish', 'Macedonian', 'Malagasy', 'Malay', 'Malayalam', 'Maltese', 'Maori', 'Marathi', 'Moldavian', 'Moldovan', 'Mongolian', 'Myanmar', 'Nepali', 'Norwegian', 'Nynorsk', 'Occitan', 'Panjabi', 'Pashto', 'Persian', 'Polish', 'Portuguese', 'Punjabi', 'Pushto', 'Romanian', 'Russian', 'Sanskrit', 'Serbian', 'Shona', 'Sindhi', 'Sinhala', 'Sinhalese', 'Slovak', 'Slovenian', 'Somali', 'Spanish', 'Sundanese', 'Swahili', 'Swedish', 'Tagalog', 'Tajik', 'Tamil', 'Tatar', 'Telugu', 'Thai', 'Tibetan', 'Turkish', 'Turkmen', 'Ukrainian', 'Urdu', 'Uzbek', 'Valencian', 'Vietnamese', 'Welsh', 'Yiddish', 'Yoruba'] ",default="English")
parser.add_argument("-p",dest="inputpath", type=str,help="Path of the input file",default="/hello.mp3")
parser.add_argument("-v",dest="typeverbose", type=str,help="Whether to print out the progress and debug messages. ['Live transcription', 'Progress bar', 'None']",default="Live transcription")
parser.add_argument("-g",dest="outputtype", type=str,help="Type of file to generate to record the transcription. ['All', '.txt', '.vtt', '.srt']",default="All")
parser.add_argument("-s",dest="speechtask", type=str,help="Whether to perform X->X speech recognition (`transcribe`) or X->English translation (`translate`). ['transcribe', 'translate']",default="transcribe")
parser.add_argument("-n",dest="numSteps", type=int,help="Number of Steps",default=50)
parser.add_argument("-t",dest="decodingtemperature", type=int,help="Temperature to increase when falling back when the decoding fails to meet either of the thresholds below.",default=0.15 )
parser.add_argument("-b",dest="beamsize", type=int,help="Number of Images",default=5)
parser.add_argument("-o",dest="output", type=str,help="Output Folder where to store the ouputs",default="")

args=parser.parse_args()
device = torch.device('cuda:0')
print('Using device:', device, file=sys.stderr)

Model = 'large'
whisper_model =whisper.load_model(Model)
video_path_local = os.getcwd()+args.inputpath
file_name=os.path.basename(video_path_local)
output_file_path=args.output

if os.path.splitext(video_path_local)[1] == ".mp4":
    video_path_local_wav =os.path.splitext(file_name)[0]+".wav"
    result  = subprocess.run(["ffmpeg", "-i", str(video_path_local), "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", str(video_path_local_wav)])

# add language parameters
# Language spoken in the audio, use Auto detection to let Whisper detect the language.
#  ['Auto detection', 'Afrikaans', 'Albanian', 'Amharic', 'Arabic', 'Armenian', 'Assamese', 'Azerbaijani', 'Bashkir', 'Basque', 'Belarusian', 'Bengali', 'Bosnian', 'Breton', 'Bulgarian', 'Burmese', 'Castilian', 'Catalan', 'Chinese', 'Croatian', 'Czech', 'Danish', 'Dutch', 'English', 'Estonian', 'Faroese', 'Finnish', 'Flemish', 'French', 'Galician', 'Georgian', 'German', 'Greek', 'Gujarati', 'Haitian', 'Haitian Creole', 'Hausa', 'Hawaiian', 'Hebrew', 'Hindi', 'Hungarian', 'Icelandic', 'Indonesian', 'Italian', 'Japanese', 'Javanese', 'Kannada', 'Kazakh', 'Khmer', 'Korean', 'Lao', 'Latin', 'Latvian', 'Letzeburgesch', 'Lingala', 'Lithuanian', 'Luxembourgish', 'Macedonian', 'Malagasy', 'Malay', 'Malayalam', 'Maltese', 'Maori', 'Marathi', 'Moldavian', 'Moldovan', 'Mongolian', 'Myanmar', 'Nepali', 'Norwegian', 'Nynorsk', 'Occitan', 'Panjabi', 'Pashto', 'Persian', 'Polish', 'Portuguese', 'Punjabi', 'Pushto', 'Romanian', 'Russian', 'Sanskrit', 'Serbian', 'Shona', 'Sindhi', 'Sinhala', 'Sinhalese', 'Slovak', 'Slovenian', 'Somali', 'Spanish', 'Sundanese', 'Swahili', 'Swedish', 'Tagalog', 'Tajik', 'Tamil', 'Tatar', 'Telugu', 'Thai', 'Tibetan', 'Turkish', 'Turkmen', 'Ukrainian', 'Urdu', 'Uzbek', 'Valencian', 'Vietnamese', 'Welsh', 'Yiddish', 'Yoruba']
language = args.audiolanguage
# Whether to print out the progress and debug messages.
# ['Live transcription', 'Progress bar', 'None']
verbose = args.typeverbose
#  Type of file to generate to record the transcription.
# ['All', '.txt', '.vtt', '.srt']
output_type = args.outputtype
# Whether to perform X->X speech recognition (`transcribe`) or X->English translation (`translate`).
# ['transcribe', 'translate']
task = args.speechtask
# Temperature to use for sampling.
temperature = args.decodingtemperature
#  Temperature to increase when falling back when the decoding fails to meet either of the thresholds below.
temperature_increment_on_fallback = 0.2
#  Number of candidates when sampling with non-zero temperature.
best_of = 5
#  Number of beams in beam search, only applicable when temperature is zero.
beam_size = args.beamsize
# Optional patience value to use in beam decoding, as in [*Beam Decoding with Controlled Patience*](https://arxiv.org/abs/2204.05424), the default (1.0) is equivalent to conventional beam search.
patience = 1.0
# Optional token length penalty coefficient (alpha) as in [*Google's Neural Machine Translation System*](https://arxiv.org/abs/1609.08144), set to negative value to uses simple length normalization.
length_penalty = -0.05
# Comma-separated list of token ids to suppress during sampling; '-1' will suppress most special characters except common punctuations.
suppress_tokens = "-1"
# Optional text to provide as a prompt for the first window.
initial_prompt = ""
# if True, provide the previous output of the model as a prompt for the next window; disabling may make the text inconsistent across windows, but the model becomes less prone to getting stuck in a failure loop.
condition_on_previous_text = True
#  whether to perform inference in fp16.
fp16 = True
#  If the gzip compression ratio is higher than this value, treat the decoding as failed.
compression_ratio_threshold = 2.4
# If the average log probability is lower than this value, treat the decoding as failed.
logprob_threshold = -1.0
# If the probability of the <|nospeech|> token is higher than this value AND the decoding has failed due to `logprob_threshold`, consider the segment as silence.
no_speech_threshold = 0.6

verbose_lut = {
    'Live transcription': True,
    'Progress bar': False,
    'None': None
}

args = dict(
    language = (None if language == "Auto detection" else language),
    verbose = verbose_lut[verbose],
    task = task,
    temperature = temperature,
    temperature_increment_on_fallback = temperature_increment_on_fallback,
    best_of = best_of,
    beam_size = beam_size,
    patience=patience,
    length_penalty=(length_penalty if length_penalty>=0.0 else None),
    suppress_tokens=suppress_tokens,
    initial_prompt=(None if not initial_prompt else initial_prompt),
    condition_on_previous_text=condition_on_previous_text,
    fp16=fp16,
    compression_ratio_threshold=compression_ratio_threshold,
    logprob_threshold=logprob_threshold,
    no_speech_threshold=no_speech_threshold
)

temperature = args.pop("temperature")
temperature_increment_on_fallback = args.pop("temperature_increment_on_fallback")
if temperature_increment_on_fallback is not None:
    temperature = tuple(np.arange(temperature, 1.0 + 1e-6, temperature_increment_on_fallback))
else:
    temperature = [temperature]

if Model.endswith(".en") and args["language"] not in {"en", "English"}:
    warnings.warn(f"{Model} is an English-only model but receipted '{args['language']}'; using English instead.")
    args["language"] = "en"

video_transcription = whisper.transcribe(
    whisper_model,
    str(video_path_local),
    temperature=temperature,
    **args,
)

# Save output
writing_lut = {
    '.txt': whisper.utils.write_txt,
    '.vtt': whisper.utils.write_vtt,
    '.srt': whisper.utils.write_txt,
}

if output_type == "All":
    for suffix, write_suffix in writing_lut.items():
        transcript_local_path =os.getcwd()+output_file_path+'/'+os.path.splitext(file_name)[0] +suffix
        with open(transcript_local_path, "w", encoding="utf-8") as f:
            write_suffix(video_transcription["segments"], file=f)
        try:
            transcript_drive_path =file_name
        except:
            print(f"**Transcript file created: {transcript_local_path}**")
else:
    transcript_local_path =output_file_path+'/'+os.path.splitext(file_name)[0] +output_type

    with open(transcript_local_path, "w", encoding="utf-8") as f:
        writing_lut[output_type](video_transcription["segments"], file=f)
```

    Overwriting openai-whisper.py




Then run the script with the default parameters




```bash
python openai-whisper.py
```

    [00:00.000 --> 00:00.500]  Hello!


    Using device: cuda:0
    100%|█████████████████████████████████████| 2.87G/2.87G [00:57<00:00, 53.9MiB/s]tcmalloc: large alloc 3087007744 bytes == 0x708a000 @  0x7fb7d179e1e7 0x4b2150 0x5ac2ec 0x5dc6af 0x58ee9b 0x5901a3 0x5e3f6b 0x4d18aa 0x51b31c 0x58f2a7 0x51740e 0x5b41c5 0x58f49e 0x51b221 0x5b41c5 0x604133 0x606e06 0x606ecc 0x609aa6 0x64d332 0x64d4de 0x7fb7d139bc87 0x5b561a
    


```
usage: openai-whisper.py [-h] [-l AUDIOLANGUAGE] [-p INPUTPATH]
                         [-v TYPEVERBOSE] [-g OUTPUTTYPE] [-s SPEECHTASK]
                         [-n NUMSTEPS] [-t DECODINGTEMPERATURE] [-b BEAMSIZE]
                         [-o OUTPUT]

OpenAI Whisper Automatic Speech Recognition

optional arguments:
  -h, --help            show this help message and exit
  -l AUDIOLANGUAGE      Language spoken in the audio, use Auto detection to
                        let Whisper detect the language. Select from the
                        following languages['Auto detection', 'Afrikaans',
                        'Albanian', 'Amharic', 'Arabic', 'Armenian',
                        'Assamese', 'Azerbaijani', 'Bashkir', 'Basque',
                        'Belarusian', 'Bengali', 'Bosnian', 'Breton',
                        'Bulgarian', 'Burmese', 'Castilian', 'Catalan',
                        'Chinese', 'Croatian', 'Czech', 'Danish', 'Dutch',
                        'English', 'Estonian', 'Faroese', 'Finnish',
                        'Flemish', 'French', 'Galician', 'Georgian', 'German',
                        'Greek', 'Gujarati', 'Haitian', 'Haitian Creole',
                        'Hausa', 'Hawaiian', 'Hebrew', 'Hindi', 'Hungarian',
                        'Icelandic', 'Indonesian', 'Italian', 'Japanese',
                        'Javanese', 'Kannada', 'Kazakh', 'Khmer', 'Korean',
                        'Lao', 'Latin', 'Latvian', 'Letzeburgesch', 'Lingala',
                        'Lithuanian', 'Luxembourgish', 'Macedonian',
                        'Malagasy', 'Malay', 'Malayalam', 'Maltese', 'Maori',
                        'Marathi', 'Moldavian', 'Moldovan', 'Mongolian',
                        'Myanmar', 'Nepali', 'Norwegian', 'Nynorsk',
                        'Occitan', 'Panjabi', 'Pashto', 'Persian', 'Polish',
                        'Portuguese', 'Punjabi', 'Pushto', 'Romanian',
                        'Russian', 'Sanskrit', 'Serbian', 'Shona', 'Sindhi',
                        'Sinhala', 'Sinhalese', 'Slovak', 'Slovenian',
                        'Somali', 'Spanish', 'Sundanese', 'Swahili',
                        'Swedish', 'Tagalog', 'Tajik', 'Tamil', 'Tatar',
                        'Telugu', 'Thai', 'Tibetan', 'Turkish', 'Turkmen',
                        'Ukrainian', 'Urdu', 'Uzbek', 'Valencian',
                        'Vietnamese', 'Welsh', 'Yiddish', 'Yoruba']
  -p INPUTPATH          Path of the input file
  -v TYPEVERBOSE        Whether to print out the progress and debug messages.
                        ['Live transcription', 'Progress bar', 'None']
  -g OUTPUTTYPE         Type of file to generate to record the transcription.
                        ['All', '.txt', '.vtt', '.srt']
  -s SPEECHTASK         Whether to perform X->X speech recognition
                        (`transcribe`) or X->English translation
                        (`translate`). ['transcribe', 'translate']
  -n NUMSTEPS           Number of Steps
  -t DECODINGTEMPERATURE
                        Temperature to increase when falling back when the
                        decoding fails to meet either of the thresholds below.
  -b BEAMSIZE           Number of Images
  -o OUTPUT             Output Folder where to store the ouputs
```

Viewing the outputs



```bash
cat hello.srt
```

    Hello!


After that we will write a DOCKERFILE to containernize this script and then run it on bacalhau

# Building and Running on docker



In this step you will create a  `Dockerfile` to create your Docker deployment. The `Dockerfile` is a text document that contains the commands used to assemble the image.

First, create the `Dockerfile`.

Next, add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included.

 Dockerfile


```
FROM  pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime

WORKDIR /

RUN apt-get -y update

RUN apt-get -y install git

RUN python3 -m pip install --upgrade pip

RUN python -m pip install regex tqdm Pillow

RUN pip install git+https://github.com/openai/whisper.git

ADD hello.mp3 hello.mp3

ADD openai-whisper.py openai-whisper.py

RUN python openai-whisper.py
```


We choose pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime as our base image

And then install all the dependencies, after that we will add the test audio file and our openai-whisper script to the container, we will also run a test command to check whether our script works inside the container and if the container builds successfully



# Running whisper on bacalhau

We will transcribe the moon landing video, which can be found here

https://www.nasa.gov/multimedia/hd/apollo11_hdpage.html

Since the downloaded video is of .mov format we convert the video to .mp4 and then upload it to IPFS

 Uploading a sample dataset to IPFS

To upload the video we will be using [https://nft.storage/docs/how-to/nftup/](https://nft.storage/docs/how-to/nftup/)

![](https://i.imgur.com/xwZT3Pi.png)

After the dataset has been uploaded, copy the CID

bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu 

## **Running the container on bacalhau**

We use the --gpu flag to denote the no of GPU we are going to use


```
bacalhau docker run \
 jsacex/whisper \
 --gpu 1 \
-i bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu \
-- python openai-whisper.py -p inputs/Apollo_11_moonwalk_montage_720p.mp4 -o outputs
```
-i bafybeielf6z4cd2nuey5arckect5bjmelhouvn5r
here we use the -i flag to mount the CID
which contains our file to the container
at the path /inputs

python openai-whisper.py -p inputs/Apollo_11_moonwalk_montage_720p.mp4 -o outputs

-p we provide it the input path of our file and then in -o we provide the path where to store the outputs


Insalling bacalhau


```python
!curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    
    BACALHAU CLI is detected:
    Client Version: v0.2.5
    Server Version: v0.2.5
    Reinstalling BACALHAU CLI - /usr/local/bin/bacalhau...
    Getting the latest BACALHAU CLI...
    Installing v0.2.5 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.2.5
    Server Version: v0.2.5



```python
!echo $(bacalhau docker run --wait --wait-timeout-secs 1000 --id-only --gpu 1  jsacex/whisper -i bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu -- python openai-whisper.py -p inputs/Apollo_11_moonwalk_montage_720p.mp4 -o outputs) > job_id.txt
!cat job_id.txt
```

    4f758052-0543-40b5-bd86-6ab41e77389a



```python
!bacalhau list --id-filter $(cat job_id.txt)
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 17:33:46 [0m[97;40m 4f758052 [0m[97;40m Docker jsacex/stable... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmcQEQPg934Pow... [0m



Where it says "`Completed `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```python
!bacalhau describe $(cat job_id.txt)
```

Since there is no error we can’t see any error instead we see the state of our job to be complete, that means 
we can download the results!
we create a temporary directory to save our results


```python
!mkdir results
```

To Download the results of your job, run 

---

the following command:


```python
! bacalhau get  $(cat job_id.txt)  --output-dir results
```

    [90m17:38:25.343 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job '4f758052-0543-40b5-bd86-6ab41e77389a'...
    2022/09/29 17:38:25 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m17:38:35.851 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m17:38:37.1 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/results'


After the download has finished you should 
see the following contents in results directory


```python
! ls results/
```

    shards	stderr	stdout	volumes

