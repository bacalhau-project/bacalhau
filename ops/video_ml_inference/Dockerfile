# This base does NOT support GPUs and its 3GB instead of 18GB
FROM ultralytics/yolov5:v7.0-cpu

# Install Google Cloud SDK for accessing the storage bucket
RUN curl -sSL https://sdk.cloud.google.com | bash
ENV PATH $PATH:/root/google-cloud-sdk/bin

# reguired to fetch videos via python
RUN pip install --no-cache-dir google-cloud-storage

# dirs the scripts are copied to and videos download to
RUN mkdir /scripts /input_videos

# downloads videos into input_videos
COPY fetch_script.py /scripts/fetch_script.py

# Embed a default model that we may override via env var
RUN mkdir /model
ADD https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5s.pt /model

##########################
# fetch_script arguments #
##########################
# the gcp bucket containing videos to perform inference over
ENV VIDEO_BUCKET_NAME="k400-sample-video-data"
# the directory the vidoes will be downloaded to and the dir yolo will use as input to perform inference over
ENV VIDEO_DOWNLOAD_DIR="/input_videos"


############################
# yolo inference arguments #
############################
# Class list here: https://github.com/ultralytics/yolov5/blob/master/data/coco.yaml, numbers map to objects e.g. YOLO_CLASSES="0 1 2 3" would be person, bicycle, car, motorcycle
# person = 0
ENV YOLO_CLASSES="0"
# will only extract frame if there is a 70% or greater confidence level
ENV YOLO_CONF_THRES="0.7"
# the weights to use, default is small
ENV YOLO_WEIGHTS_PATH="/model/yolov5s.pt"
# the directory containing results of inference
ENV YOLO_PROJECT_DIR="/outputs"

# Custom entry point script that first fetches the data, then runs detection
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
