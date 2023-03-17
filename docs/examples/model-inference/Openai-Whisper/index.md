# Speech Recognition using Whisper




# **Introduction**

Whisper is an automatic speech recognition (ASR) system trained on 680,000 hours of multilingual and multitask supervised data collected from the web. We show that the use of such a large and diverse dataset leads to improved robustness to accents, background noise and technical language. Moreover, it enables transcription in multiple languages, as well as translation from those languages into English. We are open-sourcing models and inference code to serve as a foundation for building useful applications and for further research on robust speech processing.

In this example we will transcribe an audio clip locally, containerize the script and then 
Run the container on bacalhau

The advantages of using bacalhau over managed Automatic Speech Recognition services is that you can run your own containers which can scale to do batch process petabytes of Videos, Audio for automatic speech recognition, Using our sharding feature you can do distributed inference very easily and if you have the data stored on IPFS you don't need to move the data you can do compute where the data is located, and the cost of compute is much cheaper than managed services 



# Running whisper locally

Installing dependencies like Whisper, torch, pandas


```bash
%%bash
pip install git+https://github.com/openai/whisper.git
pip install torch==1.10.1
pip install pandas
sudo apt update && sudo apt install ffmpeg
```

## Running the script

Before we create and run the script we need a sample audio file to test the code

for that we Download a sample audio clip




```bash
%%bash
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
    



We will create a script that accepts parameters (input file path, output file path, temperature etc.) and set the default parameters. Also:
* If input file is in mp4 format, than the script converts it to wav format. 
* Save the transcript in various formats, 
* We load the large model
* Then pass it the required parameters.
This model is not only limited to english and transcription, it supports other languages and also does translation, to the following languages:

![](https://i.imgur.com/ALFe4qJ.png)

The graph above is sorted in [Word Error Rate (WER)](https://huggingface.co/spaces/evaluate-metric/wer) order.

Next, lets create a openai-whisper script:


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
parser.add_argument("-o",dest="output", type=str,help="Output Folder where to store the outputs",default="")

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



Let's run the script with the default parameters:




```bash
%%bash
python openai-whisper.py
```

    [00:00.000 --> 00:00.500]  Hello!


    Using device: cuda:0
    
      0%|                                              | 0.00/2.87G [00:00<?, ?iB/s]
      0%|                                     | 4.75M/2.87G [00:00<01:03, 48.7MiB/s]
      0%|                                     | 9.40M/2.87G [00:00<01:04, 48.0MiB/s]
      0%|▏                                    | 14.0M/2.87G [00:00<01:07, 45.8MiB/s]
      1%|▏                                    | 18.8M/2.87G [00:00<01:04, 47.4MiB/s]
      1%|▎                                    | 23.3M/2.87G [00:00<01:04, 47.3MiB/s]
      1%|▎                                    | 27.8M/2.87G [00:00<01:06, 45.8MiB/s]
      1%|▍                                    | 32.5M/2.87G [00:00<01:05, 46.6MiB/s]
      1%|▍                                    | 37.1M/2.87G [00:00<01:05, 46.9MiB/s]
      1%|▌                                    | 41.9M/2.87G [00:00<01:03, 47.7MiB/s]
      2%|▌                                    | 47.3M/2.87G [00:01<01:00, 50.2MiB/s]
      2%|▋                                    | 52.1M/2.87G [00:01<01:04, 47.2MiB/s]
      2%|▋                                    | 56.8M/2.87G [00:01<01:03, 47.7MiB/s]
      2%|▊                                    | 62.1M/2.87G [00:01<01:00, 50.1MiB/s]
      2%|▊                                    | 67.9M/2.87G [00:01<00:56, 53.1MiB/s]
      3%|▉                                    | 75.1M/2.87G [00:01<00:50, 59.8MiB/s]
      3%|█                                    | 82.5M/2.87G [00:01<00:46, 65.0MiB/s]
      3%|█                                    | 88.7M/2.87G [00:01<00:47, 63.1MiB/s]
      3%|█▏                                   | 94.8M/2.87G [00:01<00:48, 61.3MiB/s]
      4%|█▎                                    | 103M/2.87G [00:01<00:43, 69.1MiB/s]
      4%|█▍                                    | 110M/2.87G [00:02<00:53, 56.1MiB/s]
      4%|█▍                                    | 116M/2.87G [00:02<00:58, 50.6MiB/s]
      4%|█▌                                    | 121M/2.87G [00:02<01:00, 48.9MiB/s]
      4%|█▌                                    | 126M/2.87G [00:02<01:01, 47.8MiB/s]
      4%|█▋                                    | 130M/2.87G [00:02<01:01, 48.1MiB/s]
      5%|█▋                                    | 135M/2.87G [00:02<01:06, 44.4MiB/s]
      5%|█▊                                    | 140M/2.87G [00:02<01:04, 45.8MiB/s]
      5%|█▊                                    | 144M/2.87G [00:03<01:08, 43.1MiB/s]
      5%|█▉                                    | 150M/2.87G [00:03<01:01, 47.4MiB/s]
      5%|█▉                                    | 155M/2.87G [00:03<01:00, 48.0MiB/s]
      6%|██                                    | 162M/2.87G [00:03<00:52, 55.6MiB/s]
      6%|██▏                                   | 167M/2.87G [00:03<00:53, 54.2MiB/s]
      6%|██▏                                   | 173M/2.87G [00:03<00:55, 52.7MiB/s]
      6%|██▎                                   | 178M/2.87G [00:03<00:55, 52.6MiB/s]
      6%|██▎                                   | 183M/2.87G [00:03<00:54, 52.9MiB/s]
      6%|██▍                                   | 188M/2.87G [00:03<00:57, 50.4MiB/s]
      7%|██▍                                   | 193M/2.87G [00:03<01:01, 46.9MiB/s]
      7%|██▌                                   | 197M/2.87G [00:04<01:01, 47.1MiB/s]
      7%|██▌                                   | 203M/2.87G [00:04<00:58, 49.1MiB/s]
      7%|██▋                                   | 208M/2.87G [00:04<00:55, 52.0MiB/s]
      7%|██▊                                   | 213M/2.87G [00:04<00:57, 50.2MiB/s]
      7%|██▊                                   | 218M/2.87G [00:04<01:02, 45.5MiB/s]
      8%|██▉                                   | 224M/2.87G [00:04<00:58, 48.9MiB/s]
      8%|███                                   | 232M/2.87G [00:04<00:46, 60.6MiB/s]
      8%|███                                   | 238M/2.87G [00:04<00:49, 57.4MiB/s]
      8%|███▏                                  | 244M/2.87G [00:04<00:50, 56.3MiB/s]
      8%|███▏                                  | 250M/2.87G [00:05<00:49, 56.7MiB/s]
      9%|███▎                                  | 255M/2.87G [00:05<00:48, 57.6MiB/s]
      9%|███▎                                  | 261M/2.87G [00:05<00:48, 58.0MiB/s]
      9%|███▍                                  | 268M/2.87G [00:05<00:44, 62.6MiB/s]
      9%|███▌                                  | 275M/2.87G [00:05<00:43, 64.4MiB/s]
     10%|███▌                                  | 281M/2.87G [00:05<01:04, 43.0MiB/s]
     10%|███▋                                  | 286M/2.87G [00:05<01:04, 43.1MiB/s]
     10%|███▊                                  | 291M/2.87G [00:05<01:03, 43.5MiB/s]
     10%|███▊                                  | 297M/2.87G [00:06<00:57, 48.6MiB/s]
     10%|███▉                                  | 302M/2.87G [00:06<00:56, 49.1MiB/s]
     10%|███▉                                  | 307M/2.87G [00:06<00:55, 49.9MiB/s]
     11%|████                                  | 312M/2.87G [00:06<00:56, 48.9MiB/s]
     11%|████                                  | 317M/2.87G [00:06<00:57, 48.0MiB/s]
     11%|████▏                                 | 321M/2.87G [00:06<00:57, 48.2MiB/s]
     11%|████▏                                 | 326M/2.87G [00:06<00:56, 48.6MiB/s]
     11%|████▎                                 | 334M/2.87G [00:06<00:47, 57.5MiB/s]
     12%|████▍                                 | 345M/2.87G [00:06<00:36, 75.1MiB/s]
     12%|████▌                                 | 352M/2.87G [00:07<00:41, 65.3MiB/s]
     12%|████▋                                 | 359M/2.87G [00:07<00:42, 64.0MiB/s]
     12%|████▋                                 | 365M/2.87G [00:07<00:44, 61.0MiB/s]
     13%|████▊                                 | 371M/2.87G [00:07<00:46, 57.6MiB/s]
     13%|████▊                                 | 377M/2.87G [00:07<00:47, 56.8MiB/s]
     13%|████▉                                 | 382M/2.87G [00:07<00:49, 54.2MiB/s]
     13%|████▉                                 | 387M/2.87G [00:07<00:49, 53.7MiB/s]
     13%|█████                                 | 393M/2.87G [00:07<00:50, 52.8MiB/s]
     14%|█████▏                                | 398M/2.87G [00:07<00:51, 51.8MiB/s]
     14%|█████▏                                | 403M/2.87G [00:08<00:52, 50.3MiB/s]
     14%|█████▎                                | 407M/2.87G [00:08<00:52, 50.2MiB/s]
     14%|█████▎                                | 412M/2.87G [00:08<00:52, 50.2MiB/s]
     14%|█████▍                                | 417M/2.87G [00:08<00:51, 51.0MiB/s]
     14%|█████▍                                | 424M/2.87G [00:08<00:47, 55.4MiB/s]
     15%|█████▌                                | 429M/2.87G [00:08<00:48, 54.5MiB/s]
     15%|█████▌                                | 434M/2.87G [00:08<00:51, 51.4MiB/s]
     15%|█████▋                                | 440M/2.87G [00:08<00:47, 54.9MiB/s]
     15%|█████▊                                | 446M/2.87G [00:08<00:49, 52.8MiB/s]
     15%|█████▊                                | 451M/2.87G [00:09<00:53, 48.6MiB/s]
     15%|█████▉                                | 455M/2.87G [00:09<00:54, 48.3MiB/s]
     16%|█████▉                                | 460M/2.87G [00:09<00:53, 48.8MiB/s]
     16%|██████                                | 465M/2.87G [00:09<00:54, 48.1MiB/s]
     16%|██████                                | 471M/2.87G [00:09<00:49, 52.6MiB/s]
     16%|██████▏                               | 477M/2.87G [00:09<00:47, 54.2MiB/s]
     16%|██████▏                               | 484M/2.87G [00:09<00:42, 60.3MiB/s]
     17%|██████▎                               | 490M/2.87G [00:09<00:41, 62.7MiB/s]
     17%|██████▍                               | 496M/2.87G [00:09<00:44, 57.5MiB/s]
     17%|██████▍                               | 502M/2.87G [00:10<00:47, 54.4MiB/s]
     17%|██████▌                               | 507M/2.87G [00:10<00:47, 53.6MiB/s]
     17%|██████▌                               | 512M/2.87G [00:10<00:48, 53.0MiB/s]
     18%|██████▋                               | 518M/2.87G [00:10<00:46, 54.8MiB/s]
     18%|██████▊                               | 523M/2.87G [00:10<00:46, 54.1MiB/s]
     18%|██████▊                               | 529M/2.87G [00:10<00:48, 52.6MiB/s]
     18%|██████▉                               | 534M/2.87G [00:10<00:48, 52.1MiB/s]
     18%|██████▉                               | 539M/2.87G [00:10<00:52, 48.3MiB/s]
     18%|███████                               | 543M/2.87G [00:10<00:55, 45.7MiB/s]
     19%|███████                               | 548M/2.87G [00:11<00:55, 45.1MiB/s]
     19%|███████▏                              | 554M/2.87G [00:11<00:50, 49.1MiB/s]
     19%|███████▏                              | 560M/2.87G [00:11<00:47, 53.1MiB/s]
     19%|███████▎                              | 565M/2.87G [00:11<00:48, 51.3MiB/s]
     19%|███████▎                              | 570M/2.87G [00:11<00:49, 50.2MiB/s]
     20%|███████▍                              | 574M/2.87G [00:11<00:49, 49.8MiB/s]
     20%|███████▍                              | 579M/2.87G [00:11<00:51, 48.4MiB/s]
     20%|███████▌                              | 584M/2.87G [00:11<00:54, 45.7MiB/s]
     20%|███████▌                              | 588M/2.87G [00:11<01:13, 33.8MiB/s]
     20%|███████▋                              | 594M/2.87G [00:12<01:01, 39.9MiB/s]
     20%|███████▊                              | 603M/2.87G [00:12<00:47, 51.2MiB/s]
     21%|███████▊                              | 608M/2.87G [00:12<00:45, 53.6MiB/s]
     21%|███████▉                              | 614M/2.87G [00:12<00:48, 50.9MiB/s]
     21%|████████                              | 624M/2.87G [00:12<00:37, 65.7MiB/s]
     21%|████████▏                             | 631M/2.87G [00:12<00:37, 64.7MiB/s]
     22%|████████▏                             | 637M/2.87G [00:12<00:40, 60.4MiB/s]
     22%|████████▎                             | 643M/2.87G [00:12<00:42, 56.8MiB/s]
     22%|████████▍                             | 649M/2.87G [00:13<00:43, 55.4MiB/s]
     22%|████████▍                             | 654M/2.87G [00:13<00:44, 54.1MiB/s]
     22%|████████▌                             | 662M/2.87G [00:13<00:38, 61.7MiB/s]
     23%|████████▋                             | 668M/2.87G [00:13<00:40, 59.2MiB/s]
     23%|████████▋                             | 674M/2.87G [00:13<00:41, 56.7MiB/s]
     23%|████████▊                             | 680M/2.87G [00:13<00:42, 55.4MiB/s]
     23%|████████▊                             | 685M/2.87G [00:13<00:45, 51.5MiB/s]
     23%|████████▉                             | 691M/2.87G [00:13<00:44, 53.0MiB/s]
     24%|████████▉                             | 696M/2.87G [00:13<00:43, 53.7MiB/s]
     24%|█████████                             | 702M/2.87G [00:14<00:42, 55.6MiB/s]
     24%|█████████▏                            | 712M/2.87G [00:14<00:32, 70.9MiB/s]
     24%|█████████▎                            | 719M/2.87G [00:14<00:34, 68.6MiB/s]
     25%|█████████▎                            | 726M/2.87G [00:14<00:35, 66.1MiB/s]
     25%|█████████▍                            | 732M/2.87G [00:14<00:37, 62.4MiB/s]
     25%|█████████▌                            | 738M/2.87G [00:14<00:37, 61.7MiB/s]
     25%|█████████▌                            | 745M/2.87G [00:14<00:36, 63.3MiB/s]
     26%|█████████▋                            | 753M/2.87G [00:14<00:32, 70.1MiB/s]
     26%|█████████▊                            | 760M/2.87G [00:14<00:36, 63.2MiB/s]
     26%|█████████▉                            | 766M/2.87G [00:15<00:36, 62.5MiB/s]
     26%|█████████▉                            | 772M/2.87G [00:15<00:38, 58.9MiB/s]
     26%|██████████                            | 778M/2.87G [00:15<00:40, 56.2MiB/s]
     27%|██████████                            | 783M/2.87G [00:15<00:41, 55.3MiB/s]
     27%|██████████▏                           | 789M/2.87G [00:15<00:40, 55.8MiB/s]
     27%|██████████▎                           | 794M/2.87G [00:15<00:40, 55.8MiB/s]
     27%|██████████▎                           | 799M/2.87G [00:15<00:41, 53.8MiB/s]
     27%|██████████▍                           | 805M/2.87G [00:15<00:40, 55.1MiB/s]
     28%|██████████▍                           | 811M/2.87G [00:15<00:39, 56.9MiB/s]
     28%|██████████▌                           | 816M/2.87G [00:15<00:40, 55.5MiB/s]
     28%|██████████▌                           | 822M/2.87G [00:16<00:38, 57.2MiB/s]
     28%|██████████▋                           | 830M/2.87G [00:16<00:35, 62.4MiB/s]
     28%|██████████▊                           | 836M/2.87G [00:16<00:37, 59.6MiB/s]
     29%|██████████▊                           | 841M/2.87G [00:16<00:39, 55.8MiB/s]
     29%|██████████▉                           | 847M/2.87G [00:16<00:39, 55.9MiB/s]
     29%|██████████▉                           | 852M/2.87G [00:16<00:40, 54.3MiB/s]
     29%|███████████                           | 857M/2.87G [00:16<00:41, 52.6MiB/s]
     29%|███████████▏                          | 862M/2.87G [00:16<00:42, 51.8MiB/s]
     29%|███████████▏                          | 867M/2.87G [00:16<00:42, 51.7MiB/s]
     30%|███████████▎                          | 872M/2.87G [00:17<00:41, 51.7MiB/s]
     30%|███████████▎                          | 877M/2.87G [00:17<00:41, 52.0MiB/s]
     30%|███████████▍                          | 883M/2.87G [00:17<00:40, 52.7MiB/s]
     30%|███████████▍                          | 888M/2.87G [00:17<00:41, 51.9MiB/s]
     30%|███████████▌                          | 893M/2.87G [00:17<00:39, 54.3MiB/s]
     31%|███████████▌                          | 899M/2.87G [00:17<00:39, 54.0MiB/s]
     31%|███████████▋                          | 904M/2.87G [00:17<00:40, 53.2MiB/s]
     31%|███████████▋                          | 909M/2.87G [00:17<00:40, 52.5MiB/s]
     31%|███████████▊                          | 914M/2.87G [00:17<00:41, 51.5MiB/s]
     31%|███████████▊                          | 919M/2.87G [00:17<00:40, 52.0MiB/s]
     31%|███████████▉                          | 925M/2.87G [00:18<00:38, 54.7MiB/s]
     32%|████████████                          | 930M/2.87G [00:18<00:40, 52.0MiB/s]
     32%|████████████                          | 935M/2.87G [00:18<00:41, 50.4MiB/s]
     32%|████████████▏                         | 941M/2.87G [00:18<00:38, 53.9MiB/s]
     32%|████████████▏                         | 947M/2.87G [00:18<00:37, 55.3MiB/s]
     32%|████████████▎                         | 952M/2.87G [00:18<00:38, 54.1MiB/s]
     33%|████████████▎                         | 957M/2.87G [00:18<00:39, 52.7MiB/s]
     33%|████████████▍                         | 962M/2.87G [00:18<00:40, 51.9MiB/s]
     33%|████████████▍                         | 967M/2.87G [00:18<00:40, 51.3MiB/s]
     33%|████████████▌                         | 972M/2.87G [00:19<00:40, 51.2MiB/s]
     33%|████████████▌                         | 977M/2.87G [00:19<00:40, 50.9MiB/s]
     33%|████████████▋                         | 982M/2.87G [00:19<00:40, 51.0MiB/s]
     34%|████████████▊                         | 990M/2.87G [00:19<00:33, 61.2MiB/s]
     34%|████████████▊                         | 997M/2.87G [00:19<00:32, 63.2MiB/s]
     34%|████████████▌                        | 0.98G/2.87G [00:19<00:33, 60.6MiB/s]
     34%|████████████▋                        | 0.98G/2.87G [00:19<00:35, 57.0MiB/s]
     34%|████████████▋                        | 0.99G/2.87G [00:19<00:42, 47.4MiB/s]
     35%|████████████▊                        | 1.00G/2.87G [00:19<00:41, 49.1MiB/s]
     35%|████████████▊                        | 1.00G/2.87G [00:20<00:41, 48.7MiB/s]
     35%|████████████▉                        | 1.00G/2.87G [00:20<00:40, 49.4MiB/s]
     35%|█████████████                        | 1.01G/2.87G [00:20<00:39, 51.3MiB/s]
     35%|█████████████                        | 1.02G/2.87G [00:20<00:36, 54.9MiB/s]
     36%|█████████████▏                       | 1.02G/2.87G [00:20<00:34, 57.3MiB/s]
     36%|█████████████▎                       | 1.03G/2.87G [00:20<00:28, 68.4MiB/s]
     36%|█████████████▎                       | 1.04G/2.87G [00:20<00:32, 60.5MiB/s]
     36%|█████████████▍                       | 1.04G/2.87G [00:20<00:33, 58.6MiB/s]
     36%|█████████████▍                       | 1.05G/2.87G [00:20<00:34, 57.3MiB/s]
     37%|█████████████▌                       | 1.05G/2.87G [00:21<00:33, 58.6MiB/s]
     37%|█████████████▋                       | 1.06G/2.87G [00:21<00:34, 55.8MiB/s]
     37%|█████████████▋                       | 1.07G/2.87G [00:21<00:34, 55.8MiB/s]
     37%|█████████████▊                       | 1.07G/2.87G [00:21<00:33, 57.1MiB/s]
     37%|█████████████▊                       | 1.08G/2.87G [00:21<00:35, 54.6MiB/s]
     38%|█████████████▉                       | 1.08G/2.87G [00:21<00:36, 53.1MiB/s]
     38%|█████████████▉                       | 1.09G/2.87G [00:21<00:37, 51.8MiB/s]
     38%|██████████████                       | 1.09G/2.87G [00:21<00:34, 55.3MiB/s]
     38%|██████████████▏                      | 1.10G/2.87G [00:21<00:34, 55.2MiB/s]
     38%|██████████████▏                      | 1.10G/2.87G [00:22<00:38, 49.5MiB/s]
     39%|██████████████▎                      | 1.11G/2.87G [00:22<00:42, 44.4MiB/s]
     39%|██████████████▎                      | 1.11G/2.87G [00:22<00:41, 46.0MiB/s]
     39%|██████████████▍                      | 1.12G/2.87G [00:22<00:40, 47.0MiB/s]
     39%|██████████████▍                      | 1.12G/2.87G [00:22<00:36, 51.7MiB/s]
     39%|██████████████▌                      | 1.13G/2.87G [00:22<00:35, 52.5MiB/s]
     39%|██████████████▌                      | 1.13G/2.87G [00:22<00:36, 51.8MiB/s]
     40%|██████████████▋                      | 1.14G/2.87G [00:22<00:33, 56.0MiB/s]
     40%|██████████████▋                      | 1.15G/2.87G [00:22<00:31, 58.3MiB/s]
     40%|██████████████▊                      | 1.15G/2.87G [00:23<00:31, 58.1MiB/s]
     40%|██████████████▉                      | 1.16G/2.87G [00:23<00:34, 53.7MiB/s]
     40%|██████████████▉                      | 1.16G/2.87G [00:23<00:36, 50.3MiB/s]
     41%|███████████████                      | 1.17G/2.87G [00:23<00:39, 46.5MiB/s]
     41%|███████████████                      | 1.17G/2.87G [00:23<00:38, 47.4MiB/s]
     41%|███████████████▏                     | 1.18G/2.87G [00:23<00:35, 51.6MiB/s]
     41%|███████████████▏                     | 1.18G/2.87G [00:23<00:32, 55.3MiB/s]
     41%|███████████████▎                     | 1.19G/2.87G [00:23<00:31, 57.0MiB/s]
     42%|███████████████▎                     | 1.19G/2.87G [00:23<00:31, 56.5MiB/s]
     42%|███████████████▍                     | 1.20G/2.87G [00:24<00:30, 59.2MiB/s]
     42%|███████████████▌                     | 1.21G/2.87G [00:24<00:31, 56.0MiB/s]
     42%|███████████████▌                     | 1.21G/2.87G [00:24<00:32, 55.0MiB/s]
     42%|███████████████▋                     | 1.22G/2.87G [00:24<00:33, 53.6MiB/s]
     42%|███████████████▋                     | 1.22G/2.87G [00:24<00:34, 51.9MiB/s]
     43%|███████████████▊                     | 1.23G/2.87G [00:24<00:34, 50.9MiB/s]
     43%|███████████████▊                     | 1.23G/2.87G [00:24<00:32, 54.7MiB/s]
     43%|███████████████▉                     | 1.24G/2.87G [00:24<00:30, 57.1MiB/s]
     43%|████████████████                     | 1.24G/2.87G [00:24<00:33, 52.6MiB/s]
     43%|████████████████                     | 1.25G/2.87G [00:25<00:35, 49.9MiB/s]
     44%|████████████████▏                    | 1.25G/2.87G [00:25<00:33, 52.4MiB/s]
     44%|████████████████▏                    | 1.26G/2.87G [00:25<00:33, 51.8MiB/s]
     44%|████████████████▎                    | 1.26G/2.87G [00:25<00:33, 52.3MiB/s]
     44%|████████████████▎                    | 1.27G/2.87G [00:25<00:32, 52.4MiB/s]
     44%|████████████████▍                    | 1.27G/2.87G [00:25<00:33, 51.8MiB/s]
     44%|████████████████▍                    | 1.28G/2.87G [00:25<00:33, 50.9MiB/s]
     45%|████████████████▌                    | 1.28G/2.87G [00:25<00:34, 50.2MiB/s]
     45%|████████████████▌                    | 1.29G/2.87G [00:25<00:33, 50.2MiB/s]
     45%|████████████████▋                    | 1.29G/2.87G [00:25<00:32, 51.7MiB/s]
     45%|████████████████▋                    | 1.30G/2.87G [00:26<00:31, 53.4MiB/s]
     45%|████████████████▊                    | 1.30G/2.87G [00:26<00:31, 53.7MiB/s]
     46%|████████████████▊                    | 1.31G/2.87G [00:26<00:29, 57.0MiB/s]
     46%|████████████████▉                    | 1.32G/2.87G [00:26<00:30, 54.9MiB/s]
     46%|████████████████▉                    | 1.32G/2.87G [00:26<00:31, 52.5MiB/s]
     46%|█████████████████                    | 1.33G/2.87G [00:26<00:31, 52.4MiB/s]
     46%|█████████████████                    | 1.33G/2.87G [00:26<00:31, 52.4MiB/s]
     46%|█████████████████▏                   | 1.34G/2.87G [00:26<00:31, 52.2MiB/s]
     47%|█████████████████▏                   | 1.34G/2.87G [00:26<00:32, 50.5MiB/s]
     47%|█████████████████▎                   | 1.34G/2.87G [00:26<00:32, 50.6MiB/s]
     47%|█████████████████▎                   | 1.35G/2.87G [00:27<00:32, 49.7MiB/s]
     47%|█████████████████▍                   | 1.35G/2.87G [00:27<00:32, 49.8MiB/s]
     47%|█████████████████▍                   | 1.36G/2.87G [00:27<00:32, 50.3MiB/s]
     47%|█████████████████▌                   | 1.36G/2.87G [00:27<00:30, 53.9MiB/s]
     48%|█████████████████▋                   | 1.37G/2.87G [00:27<00:30, 53.9MiB/s]
     48%|█████████████████▋                   | 1.37G/2.87G [00:27<00:33, 47.7MiB/s]
     48%|█████████████████▊                   | 1.38G/2.87G [00:27<00:33, 47.8MiB/s]
     48%|█████████████████▊                   | 1.38G/2.87G [00:27<00:32, 48.5MiB/s]
     48%|█████████████████▉                   | 1.39G/2.87G [00:27<00:30, 52.1MiB/s]
     49%|█████████████████▉                   | 1.39G/2.87G [00:28<00:30, 52.3MiB/s]
     49%|██████████████████                   | 1.40G/2.87G [00:28<00:32, 49.3MiB/s]
     49%|██████████████████                   | 1.40G/2.87G [00:28<00:32, 49.0MiB/s]
     49%|██████████████████▏                  | 1.41G/2.87G [00:28<00:31, 50.4MiB/s]
     49%|██████████████████▏                  | 1.41G/2.87G [00:28<00:33, 47.2MiB/s]
     49%|██████████████████▎                  | 1.42G/2.87G [00:28<00:32, 47.6MiB/s]
     50%|██████████████████▎                  | 1.42G/2.87G [00:28<00:36, 42.8MiB/s]
     50%|██████████████████▎                  | 1.43G/2.87G [00:28<00:35, 43.8MiB/s]
     50%|██████████████████▍                  | 1.43G/2.87G [00:28<00:34, 44.3MiB/s]
     50%|██████████████████▍                  | 1.44G/2.87G [00:29<00:33, 45.8MiB/s]
     50%|██████████████████▌                  | 1.44G/2.87G [00:29<00:30, 49.7MiB/s]
     50%|██████████████████▌                  | 1.45G/2.87G [00:29<00:30, 50.5MiB/s]
     51%|██████████████████▋                  | 1.45G/2.87G [00:29<00:30, 50.0MiB/s]
     51%|██████████████████▋                  | 1.46G/2.87G [00:29<00:30, 49.6MiB/s]
     51%|██████████████████▊                  | 1.46G/2.87G [00:29<00:30, 49.7MiB/s]
     51%|██████████████████▉                  | 1.47G/2.87G [00:29<00:28, 53.8MiB/s]
     51%|██████████████████▉                  | 1.47G/2.87G [00:29<00:26, 56.6MiB/s]
     51%|███████████████████                  | 1.48G/2.87G [00:29<00:26, 57.1MiB/s]
     52%|███████████████████                  | 1.48G/2.87G [00:29<00:27, 54.9MiB/s]
     52%|███████████████████▏                 | 1.49G/2.87G [00:30<00:27, 53.3MiB/s]
     52%|███████████████████▏                 | 1.49G/2.87G [00:30<00:31, 47.3MiB/s]
     52%|███████████████████▎                 | 1.50G/2.87G [00:30<00:31, 47.5MiB/s]
     52%|███████████████████▎                 | 1.50G/2.87G [00:30<00:30, 48.5MiB/s]
     52%|███████████████████▍                 | 1.51G/2.87G [00:30<00:28, 51.2MiB/s]
     53%|███████████████████▍                 | 1.52G/2.87G [00:30<00:26, 55.2MiB/s]
     53%|███████████████████▌                 | 1.52G/2.87G [00:30<00:25, 57.0MiB/s]
     53%|███████████████████▋                 | 1.53G/2.87G [00:30<00:24, 58.5MiB/s]
     53%|███████████████████▋                 | 1.53G/2.87G [00:30<00:24, 58.7MiB/s]
     53%|███████████████████▊                 | 1.54G/2.87G [00:31<00:25, 56.5MiB/s]
     54%|███████████████████▊                 | 1.54G/2.87G [00:31<00:26, 54.8MiB/s]
     54%|███████████████████▉                 | 1.55G/2.87G [00:31<00:26, 54.3MiB/s]
     54%|███████████████████▉                 | 1.55G/2.87G [00:31<00:25, 55.7MiB/s]
     54%|████████████████████                 | 1.56G/2.87G [00:31<00:27, 52.3MiB/s]
     54%|████████████████████▏                | 1.56G/2.87G [00:31<00:28, 50.2MiB/s]
     55%|████████████████████▏                | 1.57G/2.87G [00:31<00:27, 51.4MiB/s]
     55%|████████████████████▎                | 1.57G/2.87G [00:31<00:27, 51.1MiB/s]
     55%|████████████████████▎                | 1.58G/2.87G [00:31<00:28, 49.7MiB/s]
     55%|████████████████████▍                | 1.58G/2.87G [00:32<00:27, 50.7MiB/s]
     55%|████████████████████▍                | 1.59G/2.87G [00:32<00:27, 50.0MiB/s]
     55%|████████████████████▌                | 1.59G/2.87G [00:32<00:29, 46.8MiB/s]
     56%|████████████████████▌                | 1.60G/2.87G [00:32<00:29, 46.0MiB/s]
     56%|████████████████████▌                | 1.60G/2.87G [00:32<00:29, 46.0MiB/s]
     56%|████████████████████▋                | 1.61G/2.87G [00:32<00:28, 47.5MiB/s]
     56%|████████████████████▊                | 1.61G/2.87G [00:32<00:24, 55.7MiB/s]
     56%|████████████████████▊                | 1.62G/2.87G [00:32<00:22, 60.2MiB/s]
     57%|████████████████████▉                | 1.63G/2.87G [00:32<00:22, 58.3MiB/s]
     57%|████████████████████▉                | 1.63G/2.87G [00:32<00:22, 58.3MiB/s]
     57%|█████████████████████                | 1.64G/2.87G [00:33<00:22, 58.5MiB/s]
     57%|█████████████████████▏               | 1.64G/2.87G [00:33<00:22, 59.3MiB/s]
     57%|█████████████████████▏               | 1.65G/2.87G [00:33<00:21, 60.7MiB/s]
     58%|█████████████████████▎               | 1.65G/2.87G [00:33<00:21, 61.5MiB/s]
     58%|█████████████████████▎               | 1.66G/2.87G [00:33<00:23, 55.8MiB/s]
     58%|█████████████████████▍               | 1.67G/2.87G [00:33<00:24, 53.8MiB/s]
     58%|█████████████████████▌               | 1.67G/2.87G [00:33<00:25, 50.0MiB/s]
     58%|█████████████████████▌               | 1.68G/2.87G [00:33<00:25, 50.2MiB/s]
     58%|█████████████████████▋               | 1.68G/2.87G [00:33<00:24, 51.8MiB/s]
     59%|█████████████████████▋               | 1.69G/2.87G [00:34<00:20, 61.1MiB/s]
     59%|█████████████████████▊               | 1.69G/2.87G [00:34<00:21, 60.2MiB/s]
     59%|█████████████████████▉               | 1.70G/2.87G [00:34<00:21, 58.6MiB/s]
     59%|█████████████████████▉               | 1.71G/2.87G [00:34<00:21, 59.1MiB/s]
     60%|██████████████████████               | 1.71G/2.87G [00:34<00:20, 60.0MiB/s]
     60%|██████████████████████               | 1.72G/2.87G [00:34<00:20, 59.7MiB/s]
     60%|██████████████████████▏              | 1.72G/2.87G [00:34<00:20, 60.2MiB/s]
     60%|██████████████████████▏              | 1.73G/2.87G [00:34<00:20, 60.4MiB/s]
     60%|██████████████████████▎              | 1.73G/2.87G [00:34<00:21, 57.9MiB/s]
     61%|██████████████████████▍              | 1.74G/2.87G [00:35<00:21, 55.6MiB/s]
     61%|██████████████████████▍              | 1.75G/2.87G [00:35<00:21, 55.5MiB/s]
     61%|██████████████████████▌              | 1.75G/2.87G [00:35<00:21, 56.0MiB/s]
     61%|██████████████████████▌              | 1.76G/2.87G [00:35<00:21, 56.0MiB/s]
     61%|██████████████████████▋              | 1.76G/2.87G [00:35<00:23, 51.7MiB/s]
     61%|██████████████████████▋              | 1.77G/2.87G [00:35<00:23, 50.3MiB/s]
     62%|██████████████████████▊              | 1.77G/2.87G [00:35<00:24, 49.3MiB/s]
     62%|██████████████████████▊              | 1.78G/2.87G [00:35<00:21, 54.0MiB/s]
     62%|██████████████████████▉              | 1.78G/2.87G [00:35<00:19, 61.5MiB/s]
     62%|███████████████████████              | 1.79G/2.87G [00:35<00:19, 60.1MiB/s]
     62%|███████████████████████              | 1.80G/2.87G [00:36<00:20, 57.8MiB/s]
     63%|███████████████████████▏             | 1.80G/2.87G [00:36<00:19, 57.8MiB/s]
     63%|███████████████████████▏             | 1.81G/2.87G [00:36<00:20, 56.5MiB/s]
     63%|███████████████████████▎             | 1.81G/2.87G [00:36<00:20, 54.9MiB/s]
     63%|███████████████████████▍             | 1.82G/2.87G [00:36<00:21, 53.5MiB/s]
     63%|███████████████████████▍             | 1.82G/2.87G [00:36<00:21, 52.6MiB/s]
     64%|███████████████████████▌             | 1.83G/2.87G [00:36<00:19, 56.6MiB/s]
     64%|███████████████████████▌             | 1.83G/2.87G [00:36<00:19, 56.8MiB/s]
     64%|███████████████████████▋             | 1.84G/2.87G [00:36<00:19, 56.4MiB/s]
     64%|███████████████████████▋             | 1.84G/2.87G [00:37<00:20, 53.2MiB/s]
     64%|███████████████████████▊             | 1.85G/2.87G [00:37<00:19, 56.7MiB/s]
     65%|███████████████████████▉             | 1.86G/2.87G [00:37<00:18, 58.2MiB/s]
     65%|███████████████████████▉             | 1.86G/2.87G [00:37<00:19, 56.4MiB/s]
     65%|████████████████████████             | 1.87G/2.87G [00:37<00:19, 54.8MiB/s]
     65%|████████████████████████             | 1.87G/2.87G [00:37<00:18, 57.2MiB/s]
     65%|████████████████████████▏            | 1.88G/2.87G [00:37<00:18, 56.6MiB/s]
     66%|████████████████████████▏            | 1.88G/2.87G [00:37<00:19, 53.3MiB/s]
     66%|████████████████████████▎            | 1.89G/2.87G [00:37<00:18, 56.0MiB/s]
     66%|████████████████████████▍            | 1.90G/2.87G [00:38<00:18, 58.3MiB/s]
     66%|████████████████████████▍            | 1.90G/2.87G [00:38<00:17, 58.3MiB/s]
     66%|████████████████████████▌            | 1.91G/2.87G [00:38<00:18, 56.4MiB/s]
     66%|████████████████████████▌            | 1.91G/2.87G [00:38<00:19, 53.2MiB/s]
     67%|████████████████████████▋            | 1.92G/2.87G [00:38<00:19, 52.5MiB/s]
     67%|████████████████████████▋            | 1.92G/2.87G [00:38<00:19, 52.2MiB/s]
     67%|████████████████████████▊            | 1.93G/2.87G [00:38<00:19, 51.9MiB/s]
     67%|████████████████████████▊            | 1.93G/2.87G [00:38<00:19, 51.0MiB/s]
     67%|████████████████████████▉            | 1.94G/2.87G [00:38<00:20, 50.1MiB/s]
     67%|████████████████████████▉            | 1.94G/2.87G [00:38<00:20, 48.4MiB/s]
     68%|█████████████████████████            | 1.95G/2.87G [00:39<00:19, 50.6MiB/s]
     68%|█████████████████████████            | 1.95G/2.87G [00:39<00:19, 50.3MiB/s]
     68%|█████████████████████████▏           | 1.96G/2.87G [00:39<00:19, 51.7MiB/s]
     68%|█████████████████████████▏           | 1.96G/2.87G [00:39<00:17, 55.8MiB/s]
     69%|█████████████████████████▎           | 1.97G/2.87G [00:39<00:14, 66.6MiB/s]
     69%|█████████████████████████▍           | 1.98G/2.87G [00:39<00:14, 66.2MiB/s]
     69%|█████████████████████████▌           | 1.98G/2.87G [00:39<00:16, 57.2MiB/s]
     69%|█████████████████████████▌           | 1.99G/2.87G [00:39<00:16, 58.0MiB/s]
     69%|█████████████████████████▋           | 1.99G/2.87G [00:40<00:18, 50.3MiB/s]
     70%|█████████████████████████▋           | 2.00G/2.87G [00:40<00:18, 50.4MiB/s]
     70%|█████████████████████████▊           | 2.00G/2.87G [00:40<00:18, 50.1MiB/s]
     70%|█████████████████████████▊           | 2.01G/2.87G [00:40<00:19, 48.0MiB/s]
     70%|█████████████████████████▉           | 2.01G/2.87G [00:40<00:18, 50.8MiB/s]
     70%|█████████████████████████▉           | 2.02G/2.87G [00:40<00:16, 54.1MiB/s]
     70%|██████████████████████████           | 2.03G/2.87G [00:40<00:16, 56.1MiB/s]
     71%|██████████████████████████▏          | 2.03G/2.87G [00:40<00:15, 57.3MiB/s]
     71%|██████████████████████████▏          | 2.04G/2.87G [00:40<00:18, 47.7MiB/s]
     71%|██████████████████████████▎          | 2.04G/2.87G [00:41<00:17, 51.1MiB/s]
     71%|██████████████████████████▎          | 2.05G/2.87G [00:41<00:16, 52.6MiB/s]
     71%|██████████████████████████▍          | 2.05G/2.87G [00:41<00:16, 52.2MiB/s]
     72%|██████████████████████████▍          | 2.06G/2.87G [00:41<00:16, 54.2MiB/s]
     72%|██████████████████████████▌          | 2.06G/2.87G [00:41<00:15, 55.8MiB/s]
     72%|██████████████████████████▋          | 2.07G/2.87G [00:41<00:15, 55.2MiB/s]
     72%|██████████████████████████▋          | 2.08G/2.87G [00:41<00:13, 65.2MiB/s]
     72%|██████████████████████████▊          | 2.08G/2.87G [00:41<00:12, 65.4MiB/s]
     73%|██████████████████████████▉          | 2.09G/2.87G [00:41<00:13, 62.6MiB/s]
     73%|██████████████████████████▉          | 2.10G/2.87G [00:41<00:13, 60.1MiB/s]
     73%|███████████████████████████          | 2.10G/2.87G [00:42<00:13, 59.8MiB/s]
     73%|███████████████████████████▏         | 2.11G/2.87G [00:42<00:13, 61.3MiB/s]
     74%|███████████████████████████▏         | 2.11G/2.87G [00:42<00:13, 62.1MiB/s]
     74%|███████████████████████████▎         | 2.12G/2.87G [00:42<00:12, 63.0MiB/s]
     74%|███████████████████████████▎         | 2.13G/2.87G [00:42<00:12, 62.5MiB/s]
     74%|███████████████████████████▍         | 2.13G/2.87G [00:42<00:13, 58.8MiB/s]
     74%|███████████████████████████▌         | 2.14G/2.87G [00:42<00:13, 59.3MiB/s]
     75%|███████████████████████████▌         | 2.14G/2.87G [00:42<00:13, 59.4MiB/s]
     75%|███████████████████████████▋         | 2.15G/2.87G [00:42<00:13, 59.6MiB/s]
     75%|███████████████████████████▋         | 2.15G/2.87G [00:43<00:13, 55.4MiB/s]
     75%|███████████████████████████▊         | 2.16G/2.87G [00:43<00:14, 54.5MiB/s]
     75%|███████████████████████████▊         | 2.16G/2.87G [00:43<00:14, 53.1MiB/s]
     75%|███████████████████████████▉         | 2.17G/2.87G [00:43<00:14, 51.3MiB/s]
     76%|███████████████████████████▉         | 2.17G/2.87G [00:43<00:15, 49.3MiB/s]
     76%|████████████████████████████         | 2.18G/2.87G [00:43<00:15, 48.8MiB/s]
     76%|████████████████████████████         | 2.18G/2.87G [00:43<00:15, 49.3MiB/s]
     76%|████████████████████████████▏        | 2.19G/2.87G [00:43<00:15, 46.2MiB/s]
     76%|████████████████████████████▏        | 2.19G/2.87G [00:43<00:16, 45.5MiB/s]
     76%|████████████████████████████▎        | 2.20G/2.87G [00:43<00:16, 45.1MiB/s]
     77%|████████████████████████████▎        | 2.20G/2.87G [00:44<00:15, 46.8MiB/s]
     77%|████████████████████████████▍        | 2.21G/2.87G [00:44<00:13, 51.3MiB/s]
     77%|████████████████████████████▍        | 2.21G/2.87G [00:44<00:13, 54.2MiB/s]
     77%|████████████████████████████▌        | 2.22G/2.87G [00:44<00:12, 55.4MiB/s]
     77%|████████████████████████████▋        | 2.22G/2.87G [00:44<00:12, 56.0MiB/s]
     78%|████████████████████████████▋        | 2.23G/2.87G [00:44<00:12, 55.8MiB/s]
     78%|████████████████████████████▊        | 2.23G/2.87G [00:44<00:12, 53.5MiB/s]
     78%|████████████████████████████▊        | 2.24G/2.87G [00:44<00:12, 55.7MiB/s]
     78%|████████████████████████████▉        | 2.25G/2.87G [00:44<00:11, 57.5MiB/s]
     78%|████████████████████████████▉        | 2.25G/2.87G [00:45<00:11, 59.2MiB/s]
     79%|█████████████████████████████        | 2.26G/2.87G [00:45<00:11, 59.0MiB/s]
     79%|█████████████████████████████▏       | 2.26G/2.87G [00:45<00:11, 59.4MiB/s]
     79%|█████████████████████████████▏       | 2.27G/2.87G [00:45<00:11, 57.9MiB/s]
     79%|█████████████████████████████▎       | 2.27G/2.87G [00:45<00:11, 56.3MiB/s]
     79%|█████████████████████████████▎       | 2.28G/2.87G [00:45<00:10, 58.2MiB/s]
     80%|█████████████████████████████▍       | 2.29G/2.87G [00:45<00:11, 56.0MiB/s]
     80%|█████████████████████████████▍       | 2.29G/2.87G [00:45<00:11, 54.4MiB/s]
     80%|█████████████████████████████▌       | 2.30G/2.87G [00:45<00:10, 57.6MiB/s]
     80%|█████████████████████████████▋       | 2.30G/2.87G [00:45<00:11, 54.8MiB/s]
     80%|█████████████████████████████▋       | 2.31G/2.87G [00:46<00:11, 54.7MiB/s]
     80%|█████████████████████████████▊       | 2.31G/2.87G [00:46<00:11, 53.9MiB/s]
     81%|█████████████████████████████▊       | 2.32G/2.87G [00:46<00:09, 63.0MiB/s]
     81%|█████████████████████████████▉       | 2.33G/2.87G [00:46<00:09, 59.6MiB/s]
     81%|██████████████████████████████       | 2.33G/2.87G [00:46<00:10, 55.6MiB/s]
     81%|██████████████████████████████       | 2.34G/2.87G [00:46<00:10, 54.1MiB/s]
     81%|██████████████████████████████▏      | 2.34G/2.87G [00:46<00:10, 53.2MiB/s]
     82%|██████████████████████████████▏      | 2.35G/2.87G [00:46<00:10, 53.3MiB/s]
     82%|██████████████████████████████▎      | 2.35G/2.87G [00:46<00:10, 54.8MiB/s]
     82%|██████████████████████████████▎      | 2.36G/2.87G [00:47<00:09, 57.0MiB/s]
     82%|██████████████████████████████▍      | 2.37G/2.87G [00:47<00:08, 66.6MiB/s]
     83%|██████████████████████████████▌      | 2.37G/2.87G [00:47<00:08, 63.1MiB/s]
     83%|██████████████████████████████▋      | 2.38G/2.87G [00:47<00:08, 61.2MiB/s]
     83%|██████████████████████████████▋      | 2.39G/2.87G [00:47<00:09, 57.6MiB/s]
     83%|██████████████████████████████▊      | 2.39G/2.87G [00:47<00:09, 54.8MiB/s]
     83%|██████████████████████████████▊      | 2.40G/2.87G [00:47<00:09, 54.1MiB/s]
     84%|██████████████████████████████▉      | 2.40G/2.87G [00:47<00:09, 53.6MiB/s]
     84%|██████████████████████████████▉      | 2.41G/2.87G [00:47<00:09, 53.1MiB/s]
     84%|███████████████████████████████      | 2.41G/2.87G [00:48<00:09, 51.9MiB/s]
     84%|███████████████████████████████      | 2.42G/2.87G [00:48<00:09, 51.8MiB/s]
     84%|███████████████████████████████▏     | 2.42G/2.87G [00:48<00:08, 54.4MiB/s]
     84%|███████████████████████████████▏     | 2.43G/2.87G [00:48<00:09, 53.5MiB/s]
     85%|███████████████████████████████▎     | 2.43G/2.87G [00:48<00:08, 54.8MiB/s]
     85%|███████████████████████████████▎     | 2.44G/2.87G [00:48<00:08, 55.8MiB/s]
     85%|███████████████████████████████▍     | 2.44G/2.87G [00:48<00:08, 57.8MiB/s]
     85%|███████████████████████████████▌     | 2.45G/2.87G [00:48<00:08, 56.2MiB/s]
     85%|███████████████████████████████▌     | 2.45G/2.87G [00:48<00:08, 54.9MiB/s]
     86%|███████████████████████████████▋     | 2.46G/2.87G [00:49<00:08, 51.4MiB/s]
     86%|███████████████████████████████▋     | 2.47G/2.87G [00:49<00:08, 54.7MiB/s]
     86%|███████████████████████████████▊     | 2.47G/2.87G [00:49<00:07, 57.3MiB/s]
     86%|███████████████████████████████▉     | 2.48G/2.87G [00:49<00:07, 57.9MiB/s]
     86%|███████████████████████████████▉     | 2.48G/2.87G [00:49<00:07, 56.0MiB/s]
     87%|████████████████████████████████     | 2.49G/2.87G [00:49<00:07, 54.4MiB/s]
     87%|████████████████████████████████     | 2.49G/2.87G [00:49<00:07, 53.2MiB/s]
     87%|████████████████████████████████▏    | 2.50G/2.87G [00:49<00:07, 51.7MiB/s]
     87%|████████████████████████████████▏    | 2.50G/2.87G [00:49<00:07, 52.6MiB/s]
     87%|████████████████████████████████▎    | 2.51G/2.87G [00:49<00:07, 53.4MiB/s]
     87%|████████████████████████████████▎    | 2.51G/2.87G [00:50<00:07, 53.1MiB/s]
     88%|████████████████████████████████▍    | 2.52G/2.87G [00:50<00:07, 54.0MiB/s]
     88%|████████████████████████████████▍    | 2.52G/2.87G [00:50<00:06, 55.3MiB/s]
     88%|████████████████████████████████▌    | 2.53G/2.87G [00:50<00:06, 54.2MiB/s]
     88%|████████████████████████████████▋    | 2.54G/2.87G [00:50<00:05, 61.2MiB/s]
     88%|████████████████████████████████▋    | 2.54G/2.87G [00:50<00:05, 63.3MiB/s]
     89%|████████████████████████████████▊    | 2.55G/2.87G [00:50<00:05, 59.9MiB/s]
     89%|████████████████████████████████▊    | 2.55G/2.87G [00:50<00:06, 56.7MiB/s]
     89%|████████████████████████████████▉    | 2.56G/2.87G [00:50<00:06, 50.0MiB/s]
     89%|█████████████████████████████████    | 2.56G/2.87G [00:51<00:06, 51.0MiB/s]
     89%|█████████████████████████████████    | 2.57G/2.87G [00:51<00:06, 53.8MiB/s]
     90%|█████████████████████████████████▏   | 2.58G/2.87G [00:51<00:06, 49.5MiB/s]
     90%|█████████████████████████████████▏   | 2.58G/2.87G [00:51<00:06, 49.2MiB/s]
     90%|█████████████████████████████████▎   | 2.58G/2.87G [00:51<00:06, 48.8MiB/s]
     90%|█████████████████████████████████▎   | 2.59G/2.87G [00:51<00:05, 51.6MiB/s]
     90%|█████████████████████████████████▍   | 2.60G/2.87G [00:51<00:05, 54.5MiB/s]
     90%|█████████████████████████████████▍   | 2.60G/2.87G [00:51<00:05, 56.2MiB/s]
     91%|█████████████████████████████████▌   | 2.61G/2.87G [00:51<00:04, 57.5MiB/s]
     91%|█████████████████████████████████▋   | 2.61G/2.87G [00:52<00:04, 59.0MiB/s]
     91%|█████████████████████████████████▋   | 2.62G/2.87G [00:52<00:04, 59.5MiB/s]
     91%|█████████████████████████████████▊   | 2.62G/2.87G [00:52<00:04, 59.6MiB/s]
     91%|█████████████████████████████████▊   | 2.63G/2.87G [00:52<00:04, 56.5MiB/s]
     92%|█████████████████████████████████▉   | 2.64G/2.87G [00:52<00:04, 56.4MiB/s]
     92%|█████████████████████████████████▉   | 2.64G/2.87G [00:52<00:04, 57.2MiB/s]
     92%|██████████████████████████████████   | 2.65G/2.87G [00:52<00:04, 51.3MiB/s]
     92%|██████████████████████████████████   | 2.65G/2.87G [00:52<00:04, 49.8MiB/s]
     92%|██████████████████████████████████▏  | 2.66G/2.87G [00:52<00:04, 50.2MiB/s]
     93%|██████████████████████████████████▏  | 2.66G/2.87G [00:52<00:04, 50.3MiB/s]
     93%|██████████████████████████████████▎  | 2.67G/2.87G [00:53<00:04, 50.4MiB/s]
     93%|██████████████████████████████████▍  | 2.67G/2.87G [00:53<00:04, 53.0MiB/s]
     93%|██████████████████████████████████▍  | 2.68G/2.87G [00:53<00:03, 54.7MiB/s]
     93%|██████████████████████████████████▌  | 2.68G/2.87G [00:53<00:03, 54.8MiB/s]
     93%|██████████████████████████████████▌  | 2.69G/2.87G [00:53<00:03, 53.8MiB/s]
     94%|██████████████████████████████████▋  | 2.69G/2.87G [00:53<00:03, 54.7MiB/s]
     94%|██████████████████████████████████▋  | 2.70G/2.87G [00:53<00:03, 52.4MiB/s]
     94%|██████████████████████████████████▊  | 2.70G/2.87G [00:53<00:03, 47.7MiB/s]
     94%|██████████████████████████████████▊  | 2.71G/2.87G [00:53<00:03, 49.3MiB/s]
     94%|██████████████████████████████████▉  | 2.71G/2.87G [00:54<00:03, 50.1MiB/s]
     95%|██████████████████████████████████▉  | 2.72G/2.87G [00:54<00:03, 47.9MiB/s]
     95%|███████████████████████████████████  | 2.72G/2.87G [00:54<00:03, 47.2MiB/s]
     95%|███████████████████████████████████  | 2.73G/2.87G [00:54<00:03, 48.3MiB/s]
     95%|███████████████████████████████████▏ | 2.73G/2.87G [00:54<00:03, 51.2MiB/s]
     95%|███████████████████████████████████▏ | 2.74G/2.87G [00:54<00:02, 51.0MiB/s]
     95%|███████████████████████████████████▎ | 2.74G/2.87G [00:54<00:02, 50.7MiB/s]
     96%|███████████████████████████████████▎ | 2.75G/2.87G [00:54<00:02, 48.7MiB/s]
     96%|███████████████████████████████████▍ | 2.75G/2.87G [00:54<00:02, 50.0MiB/s]
     96%|███████████████████████████████████▍ | 2.76G/2.87G [00:54<00:02, 54.1MiB/s]
     96%|███████████████████████████████████▌ | 2.76G/2.87G [00:55<00:02, 53.6MiB/s]
     96%|███████████████████████████████████▌ | 2.77G/2.87G [00:55<00:02, 52.6MiB/s]
     96%|███████████████████████████████████▋ | 2.77G/2.87G [00:55<00:02, 51.7MiB/s]
     97%|███████████████████████████████████▋ | 2.78G/2.87G [00:55<00:02, 49.8MiB/s]
     97%|███████████████████████████████████▊ | 2.78G/2.87G [00:55<00:01, 50.4MiB/s]
     97%|███████████████████████████████████▊ | 2.79G/2.87G [00:55<00:01, 49.8MiB/s]
     97%|███████████████████████████████████▉ | 2.79G/2.87G [00:55<00:01, 50.2MiB/s]
     97%|███████████████████████████████████▉ | 2.80G/2.87G [00:55<00:01, 52.2MiB/s]
     98%|████████████████████████████████████ | 2.81G/2.87G [00:55<00:01, 64.3MiB/s]
     98%|████████████████████████████████████▏| 2.81G/2.87G [00:56<00:01, 60.0MiB/s]
     98%|████████████████████████████████████▎| 2.82G/2.87G [00:56<00:01, 59.4MiB/s]
     98%|████████████████████████████████████▎| 2.82G/2.87G [00:56<00:00, 56.9MiB/s]
     98%|████████████████████████████████████▍| 2.83G/2.87G [00:56<00:00, 58.2MiB/s]
     99%|████████████████████████████████████▍| 2.83G/2.87G [00:56<00:00, 54.7MiB/s]
     99%|████████████████████████████████████▌| 2.84G/2.87G [00:56<00:00, 50.6MiB/s]
     99%|████████████████████████████████████▌| 2.84G/2.87G [00:56<00:00, 49.7MiB/s]
     99%|████████████████████████████████████▋| 2.85G/2.87G [00:56<00:00, 49.2MiB/s]
     99%|████████████████████████████████████▋| 2.85G/2.87G [00:56<00:00, 49.9MiB/s]
     99%|████████████████████████████████████▊| 2.86G/2.87G [00:57<00:00, 53.2MiB/s]
    100%|████████████████████████████████████▊| 2.86G/2.87G [00:57<00:00, 54.9MiB/s]
    100%|████████████████████████████████████▉| 2.87G/2.87G [00:57<00:00, 62.2MiB/s]
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
  -o OUTPUT             Output Folder where to store the outputs
```

Viewing the outputs



```bash
%%bash
cat hello.srt
```

    Hello!


# Building and Running on docker



In this step we will create a `Dockerfile` to create your Docker deployment. 

Next, add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included.


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


We choose `pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime` as our base image

And then install all the dependencies, after that we will add the test audio file and our openai-whisper script to the container, we will also run a test command to check whether our script works inside the container and if the container builds successfully



```python

```


# Running whisper on Bacalhau

We will transcribe the moon landing video, which can be found here

https://www.nasa.gov/multimedia/hd/apollo11_hdpage.html

Since the downloaded video is in mov format we convert the video to mp4 format, and then upload it to IPFS.

To upload the video we will be using [https://nft.storage/docs/how-to/nftup/](https://nft.storage/docs/how-to/nftup/)

![](https://i.imgur.com/xwZT3Pi.png)

After the dataset has been uploaded, copy the CID:

`bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu` 

Let's run the container on Bacalhau. We use the `--gpu` flag to denote the no of GPU we are going to use:


```
bacalhau docker run \
jsacex/whisper \
--gpu 1 \
-i bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu \
-- python openai-whisper.py -p inputs/Apollo_11_moonwalk_montage_720p.mp4 -o outputs
```

In the command above we use:

* `-i bafybeielf6z4cd2nuey5arckect5bjmelhouvn5r` flag to mount the CID which contains our file to the container at the path `/inputs`
* `-p` we provide it the input path of our file
* `-o` we provide the path where to store the outputs


Let's install Bacalhau:


```python
!curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.8 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.8/bacalhau_v0.3.8_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.8/bacalhau_v0.3.8_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.3.8
    Server Version: v0.3.8



```bash
%%bash --out job_id
bacalhau docker run \ 
--wait \
--id-only \
--gpu 1 \
--timeout 3600 \
--wait-timeout-secs 3600 \
jsacex/whisper \
-i bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu \
-- python openai-whisper.py -p inputs/Apollo_11_moonwalk_montage_720p.mp4 -o outputs
```

    215dc3ca-e59a-4a06-9272-0be8304f1e1d



```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 08:04:36 [0m[97;40m d4ae780f [0m[97;40m Docker jsacex/whispe... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmabtzjaAj94sG... [0m



Where it says `Completed `, that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
%%bash
bacalhau describe ${JOB_ID}
```

    APIVersion: V1alpha1
    ClientID: 65f7e03a4abefc46b3ebcccfc84877fb15e9912fe541146996ce0b8279e51847
    CreatedAt: "2022-10-28T08:06:30.723682632Z"
    Deal:
      Concurrency: 1
    ExecutionPlan:
      ShardsTotal: 1
    ID: 215dc3ca-e59a-4a06-9272-0be8304f1e1d
    JobState:
      Nodes:
        QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86:
          Shards:
            "0":
              NodeId: QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86
              PublishedResults: {}
              State: Cancelled
              VerificationResult: {}
        QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT:
          Shards:
            "0":
              NodeId: QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT
              PublishedResults: {}
              State: Cancelled
              VerificationResult: {}
        QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG:
          Shards:
            "0":
              NodeId: QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
              PublishedResults:
                CID: bafybeievylhpwyuwegmbbozhsnf5rgeul4uyxmretjp5dybnze5h23opzu
                Name: job-215dc3ca-e59a-4a06-9272-0be8304f1e1d-shard-0-host-QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
                StorageSource: Estuary
              RunOutput:
                exitCode: 0
                runnerError: ""
                stderr: "e-libopenh264 --enable-pic --enable-pthreads --enable-shared
                  --disable-static --enable-version3 --enable-zlib --enable-libmp3lame\n
                  \ libavutil      56. 51.100 / 56. 51.100\n  libavcodec     58. 91.100
                  / 58. 91.100\n  libavformat    58. 45.100 / 58. 45.100\n  libavdevice
                  \   58. 10.100 / 58. 10.100\n  libavfilter     7. 85.100 /  7. 85.100\n
                  \ libavresample   4.  0.  0 /  4.  0.  0\n  libswscale      5.  7.100
                  /  5.  7.100\n  libswresample   3.  7.100 /  3.  7.100\nInput #0, mov,mp4,m4a,3gp,3g2,mj2,
                  from '/inputs/Apollo_11_moonwalk_montage_720p.mp4':\n  Metadata:\n    major_brand
                  \    : isom\n    minor_version   : 512\n    compatible_brands: isomiso2avc1mp41\n
                  \   encoder         : Lavf59.27.100\n  Duration: 00:02:00.17, start:
                  0.000000, bitrate: 660 kb/s\n    Stream #0:0(und): Video: h264 (High)
                  (avc1 / 0x31637661), yuv420p(tv, bt709), 1280x720, 523 kb/s, 30 fps,
                  30 tbr, 15360 tbn, 60 tbc (default)\n    Metadata:\n      handler_name
                  \   : VideoHandler\n      encoder         : Lavc59.37.100 libx264\n
                  \     timecode        : 00:00:00:00\n    Stream #0:1(eng): Audio: aac
                  (LC) (mp4a / 0x6134706D), 48000 Hz, stereo, fltp, 128 kb/s (default)\n
                  \   Metadata:\n      handler_name    : Apple Sound Media Handler\n    Stream
                  #0:2(eng): Data: none (tmcd / 0x64636D74)\n    Metadata:\n      handler_name
                  \   : TimeCodeHandler\n      timecode        : 00:00:00:00\nStream mapping:\n
                  \ Stream #0:1 -> #0:0 (aac (native) -> pcm_s16le (native))\nPress [q]
                  to stop, [?] for help\nOutput #0, wav, to 'Apollo_11_moonwalk_montage_720p.wav':\n
                  \ Metadata:\n    major_brand     : isom\n    minor_version   : 512\n
                  \   compatible_brands: isomiso2avc1mp41\n    ISFT            : Lavf58.45.100\n
                  \   Stream #0:0(eng): Audio: pcm_s16le ([1][0][0][0] / 0x0001), 16000
                  Hz, mono, s16, 256 kb/s (default)\n    Metadata:\n      handler_name
                  \   : Apple Sound Media Handler\n      encoder         : Lavc58.91.100
                  pcm_s16le\nsize=    3755kB time=00:02:00.17 bitrate= 256.0kbits/s speed=
                  358x    \nvideo:0kB audio:3755kB subtitle:0kB other streams:0kB global
                  headers:0kB muxing overhead: 0.002028%\nUsing device: cuda:0"
                stderrtruncated: true
                stdout: |-
                  [00:00.000 --> 00:07.000]  As the foot of the ladder, the lamb foot beds are only depressed in the surface about one
                  [00:14.760 --> 00:21.760]  or two inches, although the surface appears to be very, very fine grained as you get close
                  [00:23.360 --> 00:28.360]  to it. It's almost like a powder. The ground mass is very fine.
                  [00:28.360 --> 00:35.360]  Okay, I'm going to leave that one foot up there and both hands down about the fourth
                  [00:47.760 --> 00:48.760]  rung up.
                  [00:48.760 --> 00:49.760]  There you go.
                  [00:49.760 --> 00:52.760]  Okay, now I think I'll do the same.
                  [00:52.760 --> 00:59.760]  For those who haven't read the plaque, we'll read the plaque that's on the front landing
                  [01:06.840 --> 01:13.840]  gear of this lamb. There's two hemispheres, one showing each of the two hemispheres of
                  [01:13.840 --> 01:20.840]  Earth. Underneath it says, Airman from the planet Earth, first set foot upon the moon
                  [01:25.280 --> 01:28.280]  July 1969, D.C. We came in deep.
                  [01:28.280 --> 01:35.280]  I guess you're about the only person around that doesn't have TV coverage of the scene.
                  [01:35.280 --> 01:42.280]  That's all right. I don't mind a bit.
                  [01:42.280 --> 01:47.280]  How is the quality of the TV?
                  [01:47.280 --> 01:50.280]  Oh, it's beautiful, Mike. It really is.
                  [01:50.280 --> 01:55.280]  Oh, geez, that's great. Is the lighting halfway decent?
                  [01:55.280 --> 02:00.280]  Yes, indeed. They've got the flag up now, and you can see the stars and stripes on the
                stdouttruncated: false
              State: Completed
              Status: 'Got results proposal of length: 0'
              VerificationResult:
                Complete: true
                Result: true
    RequesterNodeID: QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
    RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVRKPgCfY2fgfrkHkFjeWcqno+MDpmp8DgVaY672BqJl/dZFNU9lBg2P8Znh8OTtHPPBUBk566vU3KchjW7m3uK4OudXrYEfSfEPnCGmL6GuLiZjLf+eXGEez7qPaoYqo06gD8ROdD8VVse27E96LlrpD1xKshHhqQTxKoq1y6Rx4DpbkSt966BumovWJ70w+Nt9ZkPPydRCxVnyWS1khECFQxp5Ep3NbbKtxHNX5HeULzXN5q0EQO39UN6iBhiI34eZkH7PoAm3Vk5xns//FjTAvQw6wZUu8LwvZTaihs+upx2zZysq6CEBKoeNZqed9+Tf+qHow0P5pxmiu+or+DAgMBAAE=
    Spec:
      Docker:
        Entrypoint:
        - python
        - openai-whisper.py
        - -p
        - inputs/Apollo_11_moonwalk_montage_720p.mp4
        - -o
        - outputs
        Image: jsacex/whisper
      Engine: Docker
      Language:
        JobContext: {}
      Publisher: Estuary
      Resources:
        GPU: "1"
      Sharding:
        BatchSize: 1
        GlobPatternBasePath: /inputs
      Verifier: Noop
      Wasm: {}
      inputs:
      - CID: bafybeielf6z4cd2nuey5arckect5bjmelhouvn5rhbjlvpvhp7erkrc4nu
        StorageSource: IPFS
        path: /inputs
      outputs:
      - Name: outputs
        StorageSource: IPFS
        path: /outputs


To download the results of your job, run the following command:


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job '215dc3ca-e59a-4a06-9272-0be8304f1e1d'...2022/10/28 08:13:02 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


After the download has finished we can see the following contents in results directory:


```bash
%%bash
ls results/
```

    job-215dc3ca-e59a-4a06-9272-0be8304f1e1d-shard-0-host-QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
    shards
    stderr
    stdout
    volumes


We can view the outputs:


```bash
%%bash
cat results/combined_results/outputs/Apollo_11_moonwalk_montage_720p.vtt
```

    WEBVTT
    
    00:00.000 --> 00:07.000
    As the foot of the ladder, the lamb foot beds are only depressed in the surface about one
    
    00:14.760 --> 00:21.760
    or two inches, although the surface appears to be very, very fine grained as you get close
    
    00:23.360 --> 00:28.360
    to it. It's almost like a powder. The ground mass is very fine.
    
    00:28.360 --> 00:35.360
    Okay, I'm going to leave that one foot up there and both hands down about the fourth
    
    00:47.760 --> 00:48.760
    rung up.
    
    00:48.760 --> 00:49.760
    There you go.
    
    00:49.760 --> 00:52.760
    Okay, now I think I'll do the same.
    
    00:52.760 --> 00:59.760
    For those who haven't read the plaque, we'll read the plaque that's on the front landing
    
    01:06.840 --> 01:13.840
    gear of this lamb. There's two hemispheres, one showing each of the two hemispheres of
    
    01:13.840 --> 01:20.840
    Earth. Underneath it says, Airman from the planet Earth, first set foot upon the moon
    
    01:25.280 --> 01:28.280
    July 1969, D.C. We came in deep.
    
    01:28.280 --> 01:35.280
    I guess you're about the only person around that doesn't have TV coverage of the scene.
    
    01:35.280 --> 01:42.280
    That's all right. I don't mind a bit.
    
    01:42.280 --> 01:47.280
    How is the quality of the TV?
    
    01:47.280 --> 01:50.280
    Oh, it's beautiful, Mike. It really is.
    
    01:50.280 --> 01:55.280
    Oh, geez, that's great. Is the lighting halfway decent?
    
    01:55.280 --> 02:00.280
    Yes, indeed. They've got the flag up now, and you can see the stars and stripes on the
    

