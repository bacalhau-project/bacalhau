---
sidebar_label: "Training-Pytorch-Model"
sidebar_position: 2
---
# Training Pytorch Model

## Introduction

PyTorch is a framework developed by Facebook AI Research for deep learning, featuring both beginner-friendly debugging tools and a high-level of customization for advanced users, with researchers and practitioners using it across companies like Facebook and Tesla. Applications include computer vision, natural language processing, cryptography, and more

In this example we will train a RNN MNIST neural network model

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/model-training/Training-Tensorflow-Model/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=model-training/Training-Tensorflow-Model/index.ipynb)

## Training the model locally

Prerequisites
- python
- torch
- torchvision
- NVIDIA GPU

Cloning the pytorch examples


```bash
%%bash
git clone https://github.com/pytorch/examples
```

    Cloning into 'examples'...
    remote: Enumerating objects: 3718, done.[K
    remote: Counting objects: 100% (40/40), done.[K
    remote: Compressing objects: 100% (33/33), done.[K
    remote: Total 3718 (delta 11), reused 32 (delta 7), pack-reused 3678
    Receiving objects: 100% (3718/3718), 40.95 MiB | 21.46 MiB/s, done.
    Resolving deltas: 100% (1831/1831), done.


Training a mnist_rnn model

we add the --save-model flag to save the model


```bash
%%bash
python ./examples/mnist_rnn/main.py --save-model
```

    /usr/local/lib/python3.7/dist-packages/torch/nn/functional.py:1331: UserWarning: dropout2d: Received a 2-D input to dropout2d, which is deprecated and will result in an error in a future release. To retain the behavior and silence this warning, please use dropout instead. Note that dropout2d exists to provide channel-wise dropout on inputs with 2 spatial dimensions, a channel dimension, and an optional batch dimension (i.e. 3D or 4D inputs).
      warnings.warn(warn_msg)
    Train Epoch: 1 [0/60000 (0%)]	Loss: 2.257103
    Train Epoch: 1 [640/60000 (1%)]	Loss: 2.343541
    Train Epoch: 1 [1280/60000 (2%)]	Loss: 2.286971
    Train Epoch: 1 [1920/60000 (3%)]	Loss: 2.278690
    Train Epoch: 1 [2560/60000 (4%)]	Loss: 2.325279
    Train Epoch: 1 [3200/60000 (5%)]	Loss: 2.156002
    Train Epoch: 1 [3840/60000 (6%)]	Loss: 2.213600
    Train Epoch: 1 [4480/60000 (7%)]	Loss: 2.205997
    Train Epoch: 1 [5120/60000 (9%)]	Loss: 2.104978
    Train Epoch: 1 [5760/60000 (10%)]	Loss: 2.133132
    Train Epoch: 1 [6400/60000 (11%)]	Loss: 2.141112
    Train Epoch: 1 [7040/60000 (12%)]	Loss: 2.029041
    Train Epoch: 1 [7680/60000 (13%)]	Loss: 2.038753
    Train Epoch: 1 [8320/60000 (14%)]	Loss: 1.982695
    Train Epoch: 1 [8960/60000 (15%)]	Loss: 2.027745
    Train Epoch: 1 [9600/60000 (16%)]	Loss: 1.933618
    Train Epoch: 1 [10240/60000 (17%)]	Loss: 2.001938
    Train Epoch: 1 [10880/60000 (18%)]	Loss: 1.990632
    Train Epoch: 1 [11520/60000 (19%)]	Loss: 1.903336
    Train Epoch: 1 [12160/60000 (20%)]	Loss: 1.927148
    Train Epoch: 1 [12800/60000 (21%)]	Loss: 1.932347
    Train Epoch: 1 [13440/60000 (22%)]	Loss: 1.768175
    Train Epoch: 1 [14080/60000 (23%)]	Loss: 1.793582
    Train Epoch: 1 [14720/60000 (25%)]	Loss: 1.698625
    Train Epoch: 1 [15360/60000 (26%)]	Loss: 1.919402
    Train Epoch: 1 [16000/60000 (27%)]	Loss: 1.819005
    Train Epoch: 1 [16640/60000 (28%)]	Loss: 1.798551
    Train Epoch: 1 [17280/60000 (29%)]	Loss: 1.752450
    Train Epoch: 1 [17920/60000 (30%)]	Loss: 1.580650
    Train Epoch: 1 [18560/60000 (31%)]	Loss: 1.669491
    Train Epoch: 1 [19200/60000 (32%)]	Loss: 1.666683
    Train Epoch: 1 [19840/60000 (33%)]	Loss: 1.746461
    Train Epoch: 1 [20480/60000 (34%)]	Loss: 1.750646
    Train Epoch: 1 [21120/60000 (35%)]	Loss: 1.704663
    Train Epoch: 1 [21760/60000 (36%)]	Loss: 1.545694
    Train Epoch: 1 [22400/60000 (37%)]	Loss: 1.800772
    Train Epoch: 1 [23040/60000 (38%)]	Loss: 1.807309
    Train Epoch: 1 [23680/60000 (39%)]	Loss: 1.531073
    Train Epoch: 1 [24320/60000 (41%)]	Loss: 1.644449
    Train Epoch: 1 [24960/60000 (42%)]	Loss: 1.440658
    Train Epoch: 1 [25600/60000 (43%)]	Loss: 1.572379
    Train Epoch: 1 [26240/60000 (44%)]	Loss: 1.542954
    Train Epoch: 1 [26880/60000 (45%)]	Loss: 1.636800
    Train Epoch: 1 [27520/60000 (46%)]	Loss: 1.732645
    Train Epoch: 1 [28160/60000 (47%)]	Loss: 1.556232
    Train Epoch: 1 [28800/60000 (48%)]	Loss: 1.797165
    Train Epoch: 1 [29440/60000 (49%)]	Loss: 1.550112
    Train Epoch: 1 [30080/60000 (50%)]	Loss: 1.513264
    Train Epoch: 1 [30720/60000 (51%)]	Loss: 1.349926
    Train Epoch: 1 [31360/60000 (52%)]	Loss: 1.168647
    Train Epoch: 1 [32000/60000 (53%)]	Loss: 1.371591
    Train Epoch: 1 [32640/60000 (54%)]	Loss: 1.360642
    Train Epoch: 1 [33280/60000 (55%)]	Loss: 1.319583
    Train Epoch: 1 [33920/60000 (57%)]	Loss: 1.470899
    Train Epoch: 1 [34560/60000 (58%)]	Loss: 1.229612
    Train Epoch: 1 [35200/60000 (59%)]	Loss: 1.355430
    Train Epoch: 1 [35840/60000 (60%)]	Loss: 1.162910
    Train Epoch: 1 [36480/60000 (61%)]	Loss: 1.264161
    Train Epoch: 1 [37120/60000 (62%)]	Loss: 1.304694
    Train Epoch: 1 [37760/60000 (63%)]	Loss: 1.245098
    Train Epoch: 1 [38400/60000 (64%)]	Loss: 1.276992
    Train Epoch: 1 [39040/60000 (65%)]	Loss: 1.224096
    Train Epoch: 1 [39680/60000 (66%)]	Loss: 1.017790
    Train Epoch: 1 [40320/60000 (67%)]	Loss: 1.265200
    Train Epoch: 1 [40960/60000 (68%)]	Loss: 1.095893
    Train Epoch: 1 [41600/60000 (69%)]	Loss: 1.253011
    Train Epoch: 1 [42240/60000 (70%)]	Loss: 1.309954
    Train Epoch: 1 [42880/60000 (71%)]	Loss: 1.072964
    Train Epoch: 1 [43520/60000 (72%)]	Loss: 1.278133
    Train Epoch: 1 [44160/60000 (74%)]	Loss: 1.042409
    Train Epoch: 1 [44800/60000 (75%)]	Loss: 1.204304
    Train Epoch: 1 [45440/60000 (76%)]	Loss: 1.224481
    Train Epoch: 1 [46080/60000 (77%)]	Loss: 1.168465
    Train Epoch: 1 [46720/60000 (78%)]	Loss: 1.225616
    Train Epoch: 1 [47360/60000 (79%)]	Loss: 1.107115
    Train Epoch: 1 [48000/60000 (80%)]	Loss: 0.964020
    Train Epoch: 1 [48640/60000 (81%)]	Loss: 1.150630
    Train Epoch: 1 [49280/60000 (82%)]	Loss: 1.298064
    Train Epoch: 1 [49920/60000 (83%)]	Loss: 1.385769
    Train Epoch: 1 [50560/60000 (84%)]	Loss: 1.130490
    Train Epoch: 1 [51200/60000 (85%)]	Loss: 0.967750
    Train Epoch: 1 [51840/60000 (86%)]	Loss: 1.239161
    Train Epoch: 1 [52480/60000 (87%)]	Loss: 0.985015
    Train Epoch: 1 [53120/60000 (88%)]	Loss: 1.048505
    Train Epoch: 1 [53760/60000 (90%)]	Loss: 0.928015
    Train Epoch: 1 [54400/60000 (91%)]	Loss: 1.156546
    Train Epoch: 1 [55040/60000 (92%)]	Loss: 1.117476
    Train Epoch: 1 [55680/60000 (93%)]	Loss: 1.082589
    Train Epoch: 1 [56320/60000 (94%)]	Loss: 1.037969
    Train Epoch: 1 [56960/60000 (95%)]	Loss: 0.901225
    Train Epoch: 1 [57600/60000 (96%)]	Loss: 0.939105
    Train Epoch: 1 [58240/60000 (97%)]	Loss: 0.977517
    Train Epoch: 1 [58880/60000 (98%)]	Loss: 1.061300
    Train Epoch: 1 [59520/60000 (99%)]	Loss: 1.161198
    
    Test set: Average loss: 0.7476, Accuracy: 7615/10000 (76%)
    
    Train Epoch: 2 [0/60000 (0%)]	Loss: 1.074720
    Train Epoch: 2 [640/60000 (1%)]	Loss: 1.031572
    Train Epoch: 2 [1280/60000 (2%)]	Loss: 0.896288
    Train Epoch: 2 [1920/60000 (3%)]	Loss: 1.111214
    Train Epoch: 2 [2560/60000 (4%)]	Loss: 1.075807
    Train Epoch: 2 [3200/60000 (5%)]	Loss: 0.896091
    Train Epoch: 2 [3840/60000 (6%)]	Loss: 0.898205
    Train Epoch: 2 [4480/60000 (7%)]	Loss: 0.909036
    Train Epoch: 2 [5120/60000 (9%)]	Loss: 0.871763
    Train Epoch: 2 [5760/60000 (10%)]	Loss: 0.809469
    Train Epoch: 2 [6400/60000 (11%)]	Loss: 1.018834
    Train Epoch: 2 [7040/60000 (12%)]	Loss: 0.893395
    Train Epoch: 2 [7680/60000 (13%)]	Loss: 0.832215
    Train Epoch: 2 [8320/60000 (14%)]	Loss: 0.942631
    Train Epoch: 2 [8960/60000 (15%)]	Loss: 0.899457
    Train Epoch: 2 [9600/60000 (16%)]	Loss: 1.078218
    Train Epoch: 2 [10240/60000 (17%)]	Loss: 0.860738
    Train Epoch: 2 [10880/60000 (18%)]	Loss: 0.742847
    Train Epoch: 2 [11520/60000 (19%)]	Loss: 1.037842
    Train Epoch: 2 [12160/60000 (20%)]	Loss: 1.066162
    Train Epoch: 2 [12800/60000 (21%)]	Loss: 0.885088
    Train Epoch: 2 [13440/60000 (22%)]	Loss: 0.996853
    Train Epoch: 2 [14080/60000 (23%)]	Loss: 0.822172
    Train Epoch: 2 [14720/60000 (25%)]	Loss: 0.993543
    Train Epoch: 2 [15360/60000 (26%)]	Loss: 0.810572
    Train Epoch: 2 [16000/60000 (27%)]	Loss: 1.058691
    Train Epoch: 2 [16640/60000 (28%)]	Loss: 0.866646
    Train Epoch: 2 [17280/60000 (29%)]	Loss: 0.772441
    Train Epoch: 2 [17920/60000 (30%)]	Loss: 0.720767
    Train Epoch: 2 [18560/60000 (31%)]	Loss: 0.866728
    Train Epoch: 2 [19200/60000 (32%)]	Loss: 0.705710
    Train Epoch: 2 [19840/60000 (33%)]	Loss: 0.890331
    Train Epoch: 2 [20480/60000 (34%)]	Loss: 0.834183
    Train Epoch: 2 [21120/60000 (35%)]	Loss: 0.774839
    Train Epoch: 2 [21760/60000 (36%)]	Loss: 0.879249
    Train Epoch: 2 [22400/60000 (37%)]	Loss: 0.861507
    Train Epoch: 2 [23040/60000 (38%)]	Loss: 0.725026
    Train Epoch: 2 [23680/60000 (39%)]	Loss: 0.870410
    Train Epoch: 2 [24320/60000 (41%)]	Loss: 0.694554
    Train Epoch: 2 [24960/60000 (42%)]	Loss: 0.808239
    Train Epoch: 2 [25600/60000 (43%)]	Loss: 0.807047
    Train Epoch: 2 [26240/60000 (44%)]	Loss: 0.861262
    Train Epoch: 2 [26880/60000 (45%)]	Loss: 0.760611
    Train Epoch: 2 [27520/60000 (46%)]	Loss: 0.723064
    Train Epoch: 2 [28160/60000 (47%)]	Loss: 0.645913
    Train Epoch: 2 [28800/60000 (48%)]	Loss: 0.794883
    Train Epoch: 2 [29440/60000 (49%)]	Loss: 1.018256
    Train Epoch: 2 [30080/60000 (50%)]	Loss: 0.897736
    Train Epoch: 2 [30720/60000 (51%)]	Loss: 1.036487
    Train Epoch: 2 [31360/60000 (52%)]	Loss: 0.957585
    Train Epoch: 2 [32000/60000 (53%)]	Loss: 0.648525
    Train Epoch: 2 [32640/60000 (54%)]	Loss: 0.908357
    Train Epoch: 2 [33280/60000 (55%)]	Loss: 0.844382
    Train Epoch: 2 [33920/60000 (57%)]	Loss: 0.492543
    Train Epoch: 2 [34560/60000 (58%)]	Loss: 0.767534
    Train Epoch: 2 [35200/60000 (59%)]	Loss: 0.583981
    Train Epoch: 2 [35840/60000 (60%)]	Loss: 0.670485
    Train Epoch: 2 [36480/60000 (61%)]	Loss: 0.812931
    Train Epoch: 2 [37120/60000 (62%)]	Loss: 0.675360
    Train Epoch: 2 [37760/60000 (63%)]	Loss: 0.719999
    Train Epoch: 2 [38400/60000 (64%)]	Loss: 0.733326
    Train Epoch: 2 [39040/60000 (65%)]	Loss: 0.595985
    Train Epoch: 2 [39680/60000 (66%)]	Loss: 0.761033
    Train Epoch: 2 [40320/60000 (67%)]	Loss: 0.547535
    Train Epoch: 2 [40960/60000 (68%)]	Loss: 0.713409
    Train Epoch: 2 [41600/60000 (69%)]	Loss: 0.774444
    Train Epoch: 2 [42240/60000 (70%)]	Loss: 0.536494
    Train Epoch: 2 [42880/60000 (71%)]	Loss: 0.678178
    Train Epoch: 2 [43520/60000 (72%)]	Loss: 0.612846
    Train Epoch: 2 [44160/60000 (74%)]	Loss: 0.596894
    Train Epoch: 2 [44800/60000 (75%)]	Loss: 0.629905
    Train Epoch: 2 [45440/60000 (76%)]	Loss: 0.812533
    Train Epoch: 2 [46080/60000 (77%)]	Loss: 0.749563
    Train Epoch: 2 [46720/60000 (78%)]	Loss: 0.686619
    Train Epoch: 2 [47360/60000 (79%)]	Loss: 0.817192
    Train Epoch: 2 [48000/60000 (80%)]	Loss: 0.521638
    Train Epoch: 2 [48640/60000 (81%)]	Loss: 0.948533
    Train Epoch: 2 [49280/60000 (82%)]	Loss: 0.807676
    Train Epoch: 2 [49920/60000 (83%)]	Loss: 0.609730
    Train Epoch: 2 [50560/60000 (84%)]	Loss: 0.624522
    Train Epoch: 2 [51200/60000 (85%)]	Loss: 0.688772
    Train Epoch: 2 [51840/60000 (86%)]	Loss: 0.576913
    Train Epoch: 2 [52480/60000 (87%)]	Loss: 0.583184
    Train Epoch: 2 [53120/60000 (88%)]	Loss: 0.739166
    Train Epoch: 2 [53760/60000 (90%)]	Loss: 0.768429
    Train Epoch: 2 [54400/60000 (91%)]	Loss: 0.767366
    Train Epoch: 2 [55040/60000 (92%)]	Loss: 0.739564
    Train Epoch: 2 [55680/60000 (93%)]	Loss: 0.969297
    Train Epoch: 2 [56320/60000 (94%)]	Loss: 0.545870
    Train Epoch: 2 [56960/60000 (95%)]	Loss: 0.490728
    Train Epoch: 2 [57600/60000 (96%)]	Loss: 0.738210
    Train Epoch: 2 [58240/60000 (97%)]	Loss: 0.649949
    Train Epoch: 2 [58880/60000 (98%)]	Loss: 0.534231
    Train Epoch: 2 [59520/60000 (99%)]	Loss: 0.701677
    
    Test set: Average loss: 0.4355, Accuracy: 8636/10000 (86%)
    
    Train Epoch: 3 [0/60000 (0%)]	Loss: 0.436861
    Train Epoch: 3 [640/60000 (1%)]	Loss: 0.613573
    Train Epoch: 3 [1280/60000 (2%)]	Loss: 0.751559
    Train Epoch: 3 [1920/60000 (3%)]	Loss: 0.518953
    Train Epoch: 3 [2560/60000 (4%)]	Loss: 0.706350
    Train Epoch: 3 [3200/60000 (5%)]	Loss: 0.463392
    Train Epoch: 3 [3840/60000 (6%)]	Loss: 0.637765
    Train Epoch: 3 [4480/60000 (7%)]	Loss: 0.707880
    Train Epoch: 3 [5120/60000 (9%)]	Loss: 0.705076
    Train Epoch: 3 [5760/60000 (10%)]	Loss: 0.473644
    Train Epoch: 3 [6400/60000 (11%)]	Loss: 0.566550
    Train Epoch: 3 [7040/60000 (12%)]	Loss: 0.554120
    Train Epoch: 3 [7680/60000 (13%)]	Loss: 0.735059
    Train Epoch: 3 [8320/60000 (14%)]	Loss: 0.492775
    Train Epoch: 3 [8960/60000 (15%)]	Loss: 0.705045
    Train Epoch: 3 [9600/60000 (16%)]	Loss: 0.723935
    Train Epoch: 3 [10240/60000 (17%)]	Loss: 0.657871
    Train Epoch: 3 [10880/60000 (18%)]	Loss: 0.546103
    Train Epoch: 3 [11520/60000 (19%)]	Loss: 0.576000
    Train Epoch: 3 [12160/60000 (20%)]	Loss: 0.762758
    Train Epoch: 3 [12800/60000 (21%)]	Loss: 0.672853
    Train Epoch: 3 [13440/60000 (22%)]	Loss: 0.690244
    Train Epoch: 3 [14080/60000 (23%)]	Loss: 0.491185
    Train Epoch: 3 [14720/60000 (25%)]	Loss: 0.819045
    Train Epoch: 3 [15360/60000 (26%)]	Loss: 0.633367
    Train Epoch: 3 [16000/60000 (27%)]	Loss: 0.631507
    Train Epoch: 3 [16640/60000 (28%)]	Loss: 0.742323
    Train Epoch: 3 [17280/60000 (29%)]	Loss: 0.769272
    Train Epoch: 3 [17920/60000 (30%)]	Loss: 0.547987
    Train Epoch: 3 [18560/60000 (31%)]	Loss: 0.726344
    Train Epoch: 3 [19200/60000 (32%)]	Loss: 0.500911
    Train Epoch: 3 [19840/60000 (33%)]	Loss: 0.609957
    Train Epoch: 3 [20480/60000 (34%)]	Loss: 0.567650
    Train Epoch: 3 [21120/60000 (35%)]	Loss: 0.592656
    Train Epoch: 3 [21760/60000 (36%)]	Loss: 0.659012
    Train Epoch: 3 [22400/60000 (37%)]	Loss: 0.792519
    Train Epoch: 3 [23040/60000 (38%)]	Loss: 0.649515
    Train Epoch: 3 [23680/60000 (39%)]	Loss: 0.535163
    Train Epoch: 3 [24320/60000 (41%)]	Loss: 0.510494
    Train Epoch: 3 [24960/60000 (42%)]	Loss: 0.753702
    Train Epoch: 3 [25600/60000 (43%)]	Loss: 0.588570
    Train Epoch: 3 [26240/60000 (44%)]	Loss: 0.524773
    Train Epoch: 3 [26880/60000 (45%)]	Loss: 0.654642
    Train Epoch: 3 [27520/60000 (46%)]	Loss: 0.464091
    Train Epoch: 3 [28160/60000 (47%)]	Loss: 0.517499
    Train Epoch: 3 [28800/60000 (48%)]	Loss: 0.743199
    Train Epoch: 3 [29440/60000 (49%)]	Loss: 0.712906
    Train Epoch: 3 [30080/60000 (50%)]	Loss: 0.898138
    Train Epoch: 3 [30720/60000 (51%)]	Loss: 0.471215
    Train Epoch: 3 [31360/60000 (52%)]	Loss: 0.586351
    Train Epoch: 3 [32000/60000 (53%)]	Loss: 0.619581
    Train Epoch: 3 [32640/60000 (54%)]	Loss: 0.431174
    Train Epoch: 3 [33280/60000 (55%)]	Loss: 0.805528
    Train Epoch: 3 [33920/60000 (57%)]	Loss: 0.434236
    Train Epoch: 3 [34560/60000 (58%)]	Loss: 0.833718
    Train Epoch: 3 [35200/60000 (59%)]	Loss: 0.737563
    Train Epoch: 3 [35840/60000 (60%)]	Loss: 0.814904
    Train Epoch: 3 [36480/60000 (61%)]	Loss: 0.658190
    Train Epoch: 3 [37120/60000 (62%)]	Loss: 0.642526
    Train Epoch: 3 [37760/60000 (63%)]	Loss: 0.528397
    Train Epoch: 3 [38400/60000 (64%)]	Loss: 0.401048
    Train Epoch: 3 [39040/60000 (65%)]	Loss: 0.638031
    Train Epoch: 3 [39680/60000 (66%)]	Loss: 0.885019
    Train Epoch: 3 [40320/60000 (67%)]	Loss: 0.639517
    Train Epoch: 3 [40960/60000 (68%)]	Loss: 0.777474
    Train Epoch: 3 [41600/60000 (69%)]	Loss: 0.529243
    Train Epoch: 3 [42240/60000 (70%)]	Loss: 0.383692
    Train Epoch: 3 [42880/60000 (71%)]	Loss: 0.399004
    Train Epoch: 3 [43520/60000 (72%)]	Loss: 0.602193
    Train Epoch: 3 [44160/60000 (74%)]	Loss: 0.728852
    Train Epoch: 3 [44800/60000 (75%)]	Loss: 0.605767
    Train Epoch: 3 [45440/60000 (76%)]	Loss: 1.022341
    Train Epoch: 3 [46080/60000 (77%)]	Loss: 0.670445
    Train Epoch: 3 [46720/60000 (78%)]	Loss: 0.567436
    Train Epoch: 3 [47360/60000 (79%)]	Loss: 0.486619
    Train Epoch: 3 [48000/60000 (80%)]	Loss: 0.636935
    Train Epoch: 3 [48640/60000 (81%)]	Loss: 0.501475
    Train Epoch: 3 [49280/60000 (82%)]	Loss: 0.448360
    Train Epoch: 3 [49920/60000 (83%)]	Loss: 0.548112
    Train Epoch: 3 [50560/60000 (84%)]	Loss: 0.518546
    Train Epoch: 3 [51200/60000 (85%)]	Loss: 0.460728
    Train Epoch: 3 [51840/60000 (86%)]	Loss: 0.566899
    Train Epoch: 3 [52480/60000 (87%)]	Loss: 0.455567
    Train Epoch: 3 [53120/60000 (88%)]	Loss: 0.590804
    Train Epoch: 3 [53760/60000 (90%)]	Loss: 0.655986
    Train Epoch: 3 [54400/60000 (91%)]	Loss: 0.603358
    Train Epoch: 3 [55040/60000 (92%)]	Loss: 0.498249
    Train Epoch: 3 [55680/60000 (93%)]	Loss: 0.582818
    Train Epoch: 3 [56320/60000 (94%)]	Loss: 0.671843
    Train Epoch: 3 [56960/60000 (95%)]	Loss: 0.562645
    Train Epoch: 3 [57600/60000 (96%)]	Loss: 0.710898
    Train Epoch: 3 [58240/60000 (97%)]	Loss: 0.704995
    Train Epoch: 3 [58880/60000 (98%)]	Loss: 0.426514
    Train Epoch: 3 [59520/60000 (99%)]	Loss: 0.586657
    
    Test set: Average loss: 0.3266, Accuracy: 9035/10000 (90%)
    
    Train Epoch: 4 [0/60000 (0%)]	Loss: 0.555241
    Train Epoch: 4 [640/60000 (1%)]	Loss: 0.414488
    Train Epoch: 4 [1280/60000 (2%)]	Loss: 0.423981
    Train Epoch: 4 [1920/60000 (3%)]	Loss: 0.458799
    Train Epoch: 4 [2560/60000 (4%)]	Loss: 0.526234
    Train Epoch: 4 [3200/60000 (5%)]	Loss: 0.502130
    Train Epoch: 4 [3840/60000 (6%)]	Loss: 0.572711
    Train Epoch: 4 [4480/60000 (7%)]	Loss: 0.768068
    Train Epoch: 4 [5120/60000 (9%)]	Loss: 0.552236
    Train Epoch: 4 [5760/60000 (10%)]	Loss: 0.413747
    Train Epoch: 4 [6400/60000 (11%)]	Loss: 0.495317
    Train Epoch: 4 [7040/60000 (12%)]	Loss: 0.513442
    Train Epoch: 4 [7680/60000 (13%)]	Loss: 0.371071
    Train Epoch: 4 [8320/60000 (14%)]	Loss: 0.537922
    Train Epoch: 4 [8960/60000 (15%)]	Loss: 0.550542
    Train Epoch: 4 [9600/60000 (16%)]	Loss: 0.492354
    Train Epoch: 4 [10240/60000 (17%)]	Loss: 0.430003
    Train Epoch: 4 [10880/60000 (18%)]	Loss: 0.676727
    Train Epoch: 4 [11520/60000 (19%)]	Loss: 0.522242
    Train Epoch: 4 [12160/60000 (20%)]	Loss: 0.323046
    Train Epoch: 4 [12800/60000 (21%)]	Loss: 0.413817
    Train Epoch: 4 [13440/60000 (22%)]	Loss: 0.493616
    Train Epoch: 4 [14080/60000 (23%)]	Loss: 0.482043
    Train Epoch: 4 [14720/60000 (25%)]	Loss: 0.598020
    Train Epoch: 4 [15360/60000 (26%)]	Loss: 0.698045
    Train Epoch: 4 [16000/60000 (27%)]	Loss: 0.464924
    Train Epoch: 4 [16640/60000 (28%)]	Loss: 0.598145
    Train Epoch: 4 [17280/60000 (29%)]	Loss: 0.513251
    Train Epoch: 4 [17920/60000 (30%)]	Loss: 0.383759
    Train Epoch: 4 [18560/60000 (31%)]	Loss: 0.451445
    Train Epoch: 4 [19200/60000 (32%)]	Loss: 0.298578
    Train Epoch: 4 [19840/60000 (33%)]	Loss: 0.724677
    Train Epoch: 4 [20480/60000 (34%)]	Loss: 0.648704
    Train Epoch: 4 [21120/60000 (35%)]	Loss: 0.417878
    Train Epoch: 4 [21760/60000 (36%)]	Loss: 0.587597
    Train Epoch: 4 [22400/60000 (37%)]	Loss: 0.650825
    Train Epoch: 4 [23040/60000 (38%)]	Loss: 0.461850
    Train Epoch: 4 [23680/60000 (39%)]	Loss: 0.498996
    Train Epoch: 4 [24320/60000 (41%)]	Loss: 0.272354
    Train Epoch: 4 [24960/60000 (42%)]	Loss: 0.552614
    Train Epoch: 4 [25600/60000 (43%)]	Loss: 0.559007
    Train Epoch: 4 [26240/60000 (44%)]	Loss: 0.514660
    Train Epoch: 4 [26880/60000 (45%)]	Loss: 0.449900
    Train Epoch: 4 [27520/60000 (46%)]	Loss: 0.459001
    Train Epoch: 4 [28160/60000 (47%)]	Loss: 0.510848
    Train Epoch: 4 [28800/60000 (48%)]	Loss: 0.376767
    Train Epoch: 4 [29440/60000 (49%)]	Loss: 0.663157
    Train Epoch: 4 [30080/60000 (50%)]	Loss: 0.380203
    Train Epoch: 4 [30720/60000 (51%)]	Loss: 0.487593
    Train Epoch: 4 [31360/60000 (52%)]	Loss: 0.368228
    Train Epoch: 4 [32000/60000 (53%)]	Loss: 0.531883
    Train Epoch: 4 [32640/60000 (54%)]	Loss: 0.514747
    Train Epoch: 4 [33280/60000 (55%)]	Loss: 0.413709
    Train Epoch: 4 [33920/60000 (57%)]	Loss: 0.466322
    Train Epoch: 4 [34560/60000 (58%)]	Loss: 0.481781
    Train Epoch: 4 [35200/60000 (59%)]	Loss: 0.332192
    Train Epoch: 4 [35840/60000 (60%)]	Loss: 0.535552
    Train Epoch: 4 [36480/60000 (61%)]	Loss: 0.701525
    Train Epoch: 4 [37120/60000 (62%)]	Loss: 0.472824
    Train Epoch: 4 [37760/60000 (63%)]	Loss: 0.506161
    Train Epoch: 4 [38400/60000 (64%)]	Loss: 0.434092
    Train Epoch: 4 [39040/60000 (65%)]	Loss: 0.458589
    Train Epoch: 4 [39680/60000 (66%)]	Loss: 0.571874
    Train Epoch: 4 [40320/60000 (67%)]	Loss: 0.417427
    Train Epoch: 4 [40960/60000 (68%)]	Loss: 0.562599
    Train Epoch: 4 [41600/60000 (69%)]	Loss: 0.595764
    Train Epoch: 4 [42240/60000 (70%)]	Loss: 0.763261
    Train Epoch: 4 [42880/60000 (71%)]	Loss: 0.449961
    Train Epoch: 4 [43520/60000 (72%)]	Loss: 0.504707
    Train Epoch: 4 [44160/60000 (74%)]	Loss: 0.518068
    Train Epoch: 4 [44800/60000 (75%)]	Loss: 0.457749
    Train Epoch: 4 [45440/60000 (76%)]	Loss: 0.556885
    Train Epoch: 4 [46080/60000 (77%)]	Loss: 0.407525
    Train Epoch: 4 [46720/60000 (78%)]	Loss: 0.627192
    Train Epoch: 4 [47360/60000 (79%)]	Loss: 0.640685
    Train Epoch: 4 [48000/60000 (80%)]	Loss: 0.461735
    Train Epoch: 4 [48640/60000 (81%)]	Loss: 0.440985
    Train Epoch: 4 [49280/60000 (82%)]	Loss: 0.617622
    Train Epoch: 4 [49920/60000 (83%)]	Loss: 0.502659
    Train Epoch: 4 [50560/60000 (84%)]	Loss: 0.525112
    Train Epoch: 4 [51200/60000 (85%)]	Loss: 0.530759
    Train Epoch: 4 [51840/60000 (86%)]	Loss: 0.327249
    Train Epoch: 4 [52480/60000 (87%)]	Loss: 0.392866
    Train Epoch: 4 [53120/60000 (88%)]	Loss: 0.716493
    Train Epoch: 4 [53760/60000 (90%)]	Loss: 0.916052
    Train Epoch: 4 [54400/60000 (91%)]	Loss: 0.398534
    Train Epoch: 4 [55040/60000 (92%)]	Loss: 0.514750
    Train Epoch: 4 [55680/60000 (93%)]	Loss: 0.466898
    Train Epoch: 4 [56320/60000 (94%)]	Loss: 0.446999
    Train Epoch: 4 [56960/60000 (95%)]	Loss: 0.575152
    Train Epoch: 4 [57600/60000 (96%)]	Loss: 0.578759
    Train Epoch: 4 [58240/60000 (97%)]	Loss: 0.473566
    Train Epoch: 4 [58880/60000 (98%)]	Loss: 0.520567
    Train Epoch: 4 [59520/60000 (99%)]	Loss: 0.242124
    
    Test set: Average loss: 0.2797, Accuracy: 9146/10000 (91%)
    
    Train Epoch: 5 [0/60000 (0%)]	Loss: 0.509088
    Train Epoch: 5 [640/60000 (1%)]	Loss: 0.581982
    Train Epoch: 5 [1280/60000 (2%)]	Loss: 0.393443
    Train Epoch: 5 [1920/60000 (3%)]	Loss: 0.635975
    Train Epoch: 5 [2560/60000 (4%)]	Loss: 0.359194
    Train Epoch: 5 [3200/60000 (5%)]	Loss: 0.446414
    Train Epoch: 5 [3840/60000 (6%)]	Loss: 0.638958
    Train Epoch: 5 [4480/60000 (7%)]	Loss: 0.456178
    Train Epoch: 5 [5120/60000 (9%)]	Loss: 0.676889
    Train Epoch: 5 [5760/60000 (10%)]	Loss: 0.725724
    Train Epoch: 5 [6400/60000 (11%)]	Loss: 0.758731
    Train Epoch: 5 [7040/60000 (12%)]	Loss: 0.298136
    Train Epoch: 5 [7680/60000 (13%)]	Loss: 0.498484
    Train Epoch: 5 [8320/60000 (14%)]	Loss: 0.781466
    Train Epoch: 5 [8960/60000 (15%)]	Loss: 0.372765
    Train Epoch: 5 [9600/60000 (16%)]	Loss: 0.551780
    Train Epoch: 5 [10240/60000 (17%)]	Loss: 0.671177
    Train Epoch: 5 [10880/60000 (18%)]	Loss: 0.386135
    Train Epoch: 5 [11520/60000 (19%)]	Loss: 0.429770
    Train Epoch: 5 [12160/60000 (20%)]	Loss: 0.351372
    Train Epoch: 5 [12800/60000 (21%)]	Loss: 0.712960
    Train Epoch: 5 [13440/60000 (22%)]	Loss: 0.696320
    Train Epoch: 5 [14080/60000 (23%)]	Loss: 0.242317
    Train Epoch: 5 [14720/60000 (25%)]	Loss: 0.757244
    Train Epoch: 5 [15360/60000 (26%)]	Loss: 0.641723
    Train Epoch: 5 [16000/60000 (27%)]	Loss: 0.303923
    Train Epoch: 5 [16640/60000 (28%)]	Loss: 0.451922
    Train Epoch: 5 [17280/60000 (29%)]	Loss: 0.546510
    Train Epoch: 5 [17920/60000 (30%)]	Loss: 0.449047
    Train Epoch: 5 [18560/60000 (31%)]	Loss: 0.497757
    Train Epoch: 5 [19200/60000 (32%)]	Loss: 0.590393
    Train Epoch: 5 [19840/60000 (33%)]	Loss: 0.591735
    Train Epoch: 5 [20480/60000 (34%)]	Loss: 0.422177
    Train Epoch: 5 [21120/60000 (35%)]	Loss: 0.596936
    Train Epoch: 5 [21760/60000 (36%)]	Loss: 0.533217
    Train Epoch: 5 [22400/60000 (37%)]	Loss: 0.441300
    Train Epoch: 5 [23040/60000 (38%)]	Loss: 0.472163
    Train Epoch: 5 [23680/60000 (39%)]	Loss: 0.565845
    Train Epoch: 5 [24320/60000 (41%)]	Loss: 0.585979
    Train Epoch: 5 [24960/60000 (42%)]	Loss: 0.654992
    Train Epoch: 5 [25600/60000 (43%)]	Loss: 0.646540
    Train Epoch: 5 [26240/60000 (44%)]	Loss: 0.327594
    Train Epoch: 5 [26880/60000 (45%)]	Loss: 0.361460
    Train Epoch: 5 [27520/60000 (46%)]	Loss: 0.527023
    Train Epoch: 5 [28160/60000 (47%)]	Loss: 0.510980
    Train Epoch: 5 [28800/60000 (48%)]	Loss: 0.596273
    Train Epoch: 5 [29440/60000 (49%)]	Loss: 0.641761
    Train Epoch: 5 [30080/60000 (50%)]	Loss: 0.352163
    Train Epoch: 5 [30720/60000 (51%)]	Loss: 0.477677
    Train Epoch: 5 [31360/60000 (52%)]	Loss: 0.331182
    Train Epoch: 5 [32000/60000 (53%)]	Loss: 0.546108
    Train Epoch: 5 [32640/60000 (54%)]	Loss: 0.691826
    Train Epoch: 5 [33280/60000 (55%)]	Loss: 0.432296
    Train Epoch: 5 [33920/60000 (57%)]	Loss: 0.293409
    Train Epoch: 5 [34560/60000 (58%)]	Loss: 0.461841
    Train Epoch: 5 [35200/60000 (59%)]	Loss: 0.441172
    Train Epoch: 5 [35840/60000 (60%)]	Loss: 0.450768
    Train Epoch: 5 [36480/60000 (61%)]	Loss: 0.479811
    Train Epoch: 5 [37120/60000 (62%)]	Loss: 0.368302
    Train Epoch: 5 [37760/60000 (63%)]	Loss: 0.714117
    Train Epoch: 5 [38400/60000 (64%)]	Loss: 0.512306
    Train Epoch: 5 [39040/60000 (65%)]	Loss: 0.353668
    Train Epoch: 5 [39680/60000 (66%)]	Loss: 0.634520
    Train Epoch: 5 [40320/60000 (67%)]	Loss: 0.508755
    Train Epoch: 5 [40960/60000 (68%)]	Loss: 0.574378
    Train Epoch: 5 [41600/60000 (69%)]	Loss: 0.515621
    Train Epoch: 5 [42240/60000 (70%)]	Loss: 0.340576
    Train Epoch: 5 [42880/60000 (71%)]	Loss: 0.285466
    Train Epoch: 5 [43520/60000 (72%)]	Loss: 0.502436
    Train Epoch: 5 [44160/60000 (74%)]	Loss: 0.399609
    Train Epoch: 5 [44800/60000 (75%)]	Loss: 0.348736
    Train Epoch: 5 [45440/60000 (76%)]	Loss: 0.346850
    Train Epoch: 5 [46080/60000 (77%)]	Loss: 0.276397
    Train Epoch: 5 [46720/60000 (78%)]	Loss: 0.838089
    Train Epoch: 5 [47360/60000 (79%)]	Loss: 0.402147
    Train Epoch: 5 [48000/60000 (80%)]	Loss: 0.303684
    Train Epoch: 5 [48640/60000 (81%)]	Loss: 0.553139
    Train Epoch: 5 [49280/60000 (82%)]	Loss: 0.497246
    Train Epoch: 5 [49920/60000 (83%)]	Loss: 0.535975
    Train Epoch: 5 [50560/60000 (84%)]	Loss: 0.429838
    Train Epoch: 5 [51200/60000 (85%)]	Loss: 0.462401
    Train Epoch: 5 [51840/60000 (86%)]	Loss: 0.443050
    Train Epoch: 5 [52480/60000 (87%)]	Loss: 0.449190
    Train Epoch: 5 [53120/60000 (88%)]	Loss: 0.407580
    Train Epoch: 5 [53760/60000 (90%)]	Loss: 0.709944
    Train Epoch: 5 [54400/60000 (91%)]	Loss: 0.663002
    Train Epoch: 5 [55040/60000 (92%)]	Loss: 0.664517
    Train Epoch: 5 [55680/60000 (93%)]	Loss: 0.559338
    Train Epoch: 5 [56320/60000 (94%)]	Loss: 0.369790
    Train Epoch: 5 [56960/60000 (95%)]	Loss: 0.673157
    Train Epoch: 5 [57600/60000 (96%)]	Loss: 0.338669
    Train Epoch: 5 [58240/60000 (97%)]	Loss: 0.492030
    Train Epoch: 5 [58880/60000 (98%)]	Loss: 0.344072
    Train Epoch: 5 [59520/60000 (99%)]	Loss: 0.422336
    
    Test set: Average loss: 0.2519, Accuracy: 9238/10000 (92%)
    
    Train Epoch: 6 [0/60000 (0%)]	Loss: 0.386451
    Train Epoch: 6 [640/60000 (1%)]	Loss: 0.457663
    Train Epoch: 6 [1280/60000 (2%)]	Loss: 0.515761
    Train Epoch: 6 [1920/60000 (3%)]	Loss: 0.612987
    Train Epoch: 6 [2560/60000 (4%)]	Loss: 0.787487
    Train Epoch: 6 [3200/60000 (5%)]	Loss: 0.491761
    Train Epoch: 6 [3840/60000 (6%)]	Loss: 0.454228
    Train Epoch: 6 [4480/60000 (7%)]	Loss: 0.359811
    Train Epoch: 6 [5120/60000 (9%)]	Loss: 0.368992
    Train Epoch: 6 [5760/60000 (10%)]	Loss: 0.442591
    Train Epoch: 6 [6400/60000 (11%)]	Loss: 0.597941
    Train Epoch: 6 [7040/60000 (12%)]	Loss: 0.383115
    Train Epoch: 6 [7680/60000 (13%)]	Loss: 0.362788
    Train Epoch: 6 [8320/60000 (14%)]	Loss: 0.514896
    Train Epoch: 6 [8960/60000 (15%)]	Loss: 0.774907
    Train Epoch: 6 [9600/60000 (16%)]	Loss: 0.390481
    Train Epoch: 6 [10240/60000 (17%)]	Loss: 0.584314
    Train Epoch: 6 [10880/60000 (18%)]	Loss: 0.288985
    Train Epoch: 6 [11520/60000 (19%)]	Loss: 0.426987
    Train Epoch: 6 [12160/60000 (20%)]	Loss: 0.278613
    Train Epoch: 6 [12800/60000 (21%)]	Loss: 0.499849
    Train Epoch: 6 [13440/60000 (22%)]	Loss: 0.431185
    Train Epoch: 6 [14080/60000 (23%)]	Loss: 0.689421
    Train Epoch: 6 [14720/60000 (25%)]	Loss: 0.337867
    Train Epoch: 6 [15360/60000 (26%)]	Loss: 0.626685
    Train Epoch: 6 [16000/60000 (27%)]	Loss: 0.497805
    Train Epoch: 6 [16640/60000 (28%)]	Loss: 0.441194
    Train Epoch: 6 [17280/60000 (29%)]	Loss: 0.561231
    Train Epoch: 6 [17920/60000 (30%)]	Loss: 0.401973
    Train Epoch: 6 [18560/60000 (31%)]	Loss: 0.561977
    Train Epoch: 6 [19200/60000 (32%)]	Loss: 0.410717
    Train Epoch: 6 [19840/60000 (33%)]	Loss: 0.770685
    Train Epoch: 6 [20480/60000 (34%)]	Loss: 0.639804
    Train Epoch: 6 [21120/60000 (35%)]	Loss: 0.302792
    Train Epoch: 6 [21760/60000 (36%)]	Loss: 0.529687
    Train Epoch: 6 [22400/60000 (37%)]	Loss: 0.717906
    Train Epoch: 6 [23040/60000 (38%)]	Loss: 0.498945
    Train Epoch: 6 [23680/60000 (39%)]	Loss: 0.429929
    Train Epoch: 6 [24320/60000 (41%)]	Loss: 0.435225
    Train Epoch: 6 [24960/60000 (42%)]	Loss: 0.320319
    Train Epoch: 6 [25600/60000 (43%)]	Loss: 0.590387
    Train Epoch: 6 [26240/60000 (44%)]	Loss: 0.265355
    Train Epoch: 6 [26880/60000 (45%)]	Loss: 0.454373
    Train Epoch: 6 [27520/60000 (46%)]	Loss: 0.790875
    Train Epoch: 6 [28160/60000 (47%)]	Loss: 0.486921
    Train Epoch: 6 [28800/60000 (48%)]	Loss: 0.462753
    Train Epoch: 6 [29440/60000 (49%)]	Loss: 0.813337
    Train Epoch: 6 [30080/60000 (50%)]	Loss: 0.308712
    Train Epoch: 6 [30720/60000 (51%)]	Loss: 0.476948
    Train Epoch: 6 [31360/60000 (52%)]	Loss: 0.649331
    Train Epoch: 6 [32000/60000 (53%)]	Loss: 0.337972
    Train Epoch: 6 [32640/60000 (54%)]	Loss: 0.552407
    Train Epoch: 6 [33280/60000 (55%)]	Loss: 0.584259
    Train Epoch: 6 [33920/60000 (57%)]	Loss: 0.682539
    Train Epoch: 6 [34560/60000 (58%)]	Loss: 0.472495
    Train Epoch: 6 [35200/60000 (59%)]	Loss: 0.581826
    Train Epoch: 6 [35840/60000 (60%)]	Loss: 0.430555
    Train Epoch: 6 [36480/60000 (61%)]	Loss: 0.408301
    Train Epoch: 6 [37120/60000 (62%)]	Loss: 0.544223
    Train Epoch: 6 [37760/60000 (63%)]	Loss: 0.276037
    Train Epoch: 6 [38400/60000 (64%)]	Loss: 0.383866
    Train Epoch: 6 [39040/60000 (65%)]	Loss: 0.486723
    Train Epoch: 6 [39680/60000 (66%)]	Loss: 0.401154
    Train Epoch: 6 [40320/60000 (67%)]	Loss: 0.501817
    Train Epoch: 6 [40960/60000 (68%)]	Loss: 0.514987
    Train Epoch: 6 [41600/60000 (69%)]	Loss: 0.501832
    Train Epoch: 6 [42240/60000 (70%)]	Loss: 0.471297
    Train Epoch: 6 [42880/60000 (71%)]	Loss: 0.467299
    Train Epoch: 6 [43520/60000 (72%)]	Loss: 0.421591
    Train Epoch: 6 [44160/60000 (74%)]	Loss: 0.485595
    Train Epoch: 6 [44800/60000 (75%)]	Loss: 0.450339
    Train Epoch: 6 [45440/60000 (76%)]	Loss: 0.339639
    Train Epoch: 6 [46080/60000 (77%)]	Loss: 0.386934
    Train Epoch: 6 [46720/60000 (78%)]	Loss: 0.288079
    Train Epoch: 6 [47360/60000 (79%)]	Loss: 0.448822
    Train Epoch: 6 [48000/60000 (80%)]	Loss: 0.774343
    Train Epoch: 6 [48640/60000 (81%)]	Loss: 0.379256
    Train Epoch: 6 [49280/60000 (82%)]	Loss: 0.430138
    Train Epoch: 6 [49920/60000 (83%)]	Loss: 0.486228
    Train Epoch: 6 [50560/60000 (84%)]	Loss: 0.548016
    Train Epoch: 6 [51200/60000 (85%)]	Loss: 0.312752
    Train Epoch: 6 [51840/60000 (86%)]	Loss: 0.405820
    Train Epoch: 6 [52480/60000 (87%)]	Loss: 0.346440
    Train Epoch: 6 [53120/60000 (88%)]	Loss: 0.289083
    Train Epoch: 6 [53760/60000 (90%)]	Loss: 0.595599
    Train Epoch: 6 [54400/60000 (91%)]	Loss: 0.303218
    Train Epoch: 6 [55040/60000 (92%)]	Loss: 0.461978
    Train Epoch: 6 [55680/60000 (93%)]	Loss: 0.425981
    Train Epoch: 6 [56320/60000 (94%)]	Loss: 0.318439
    Train Epoch: 6 [56960/60000 (95%)]	Loss: 0.555305
    Train Epoch: 6 [57600/60000 (96%)]	Loss: 0.662117
    Train Epoch: 6 [58240/60000 (97%)]	Loss: 0.489319
    Train Epoch: 6 [58880/60000 (98%)]	Loss: 0.406899
    Train Epoch: 6 [59520/60000 (99%)]	Loss: 0.385348
    
    Test set: Average loss: 0.2355, Accuracy: 9277/10000 (93%)
    
    Train Epoch: 7 [0/60000 (0%)]	Loss: 0.717746
    Train Epoch: 7 [640/60000 (1%)]	Loss: 0.469850
    Train Epoch: 7 [1280/60000 (2%)]	Loss: 0.594132
    Train Epoch: 7 [1920/60000 (3%)]	Loss: 0.475334
    Train Epoch: 7 [2560/60000 (4%)]	Loss: 0.430496
    Train Epoch: 7 [3200/60000 (5%)]	Loss: 0.294112
    Train Epoch: 7 [3840/60000 (6%)]	Loss: 0.312968
    Train Epoch: 7 [4480/60000 (7%)]	Loss: 0.362220
    Train Epoch: 7 [5120/60000 (9%)]	Loss: 0.429730
    Train Epoch: 7 [5760/60000 (10%)]	Loss: 0.357846
    Train Epoch: 7 [6400/60000 (11%)]	Loss: 0.336342
    Train Epoch: 7 [7040/60000 (12%)]	Loss: 0.553371
    Train Epoch: 7 [7680/60000 (13%)]	Loss: 0.517778
    Train Epoch: 7 [8320/60000 (14%)]	Loss: 0.441374
    Train Epoch: 7 [8960/60000 (15%)]	Loss: 0.242141
    Train Epoch: 7 [9600/60000 (16%)]	Loss: 0.288597
    Train Epoch: 7 [10240/60000 (17%)]	Loss: 0.355948
    Train Epoch: 7 [10880/60000 (18%)]	Loss: 0.225561
    Train Epoch: 7 [11520/60000 (19%)]	Loss: 0.556643
    Train Epoch: 7 [12160/60000 (20%)]	Loss: 0.426134
    Train Epoch: 7 [12800/60000 (21%)]	Loss: 0.408436
    Train Epoch: 7 [13440/60000 (22%)]	Loss: 0.452091
    Train Epoch: 7 [14080/60000 (23%)]	Loss: 0.417876
    Train Epoch: 7 [14720/60000 (25%)]	Loss: 0.312885
    Train Epoch: 7 [15360/60000 (26%)]	Loss: 0.513127
    Train Epoch: 7 [16000/60000 (27%)]	Loss: 0.371684
    Train Epoch: 7 [16640/60000 (28%)]	Loss: 0.347489
    Train Epoch: 7 [17280/60000 (29%)]	Loss: 0.463195
    Train Epoch: 7 [17920/60000 (30%)]	Loss: 0.391325
    Train Epoch: 7 [18560/60000 (31%)]	Loss: 0.483347
    Train Epoch: 7 [19200/60000 (32%)]	Loss: 0.341747
    Train Epoch: 7 [19840/60000 (33%)]	Loss: 0.484753
    Train Epoch: 7 [20480/60000 (34%)]	Loss: 0.342775
    Train Epoch: 7 [21120/60000 (35%)]	Loss: 0.680683
    Train Epoch: 7 [21760/60000 (36%)]	Loss: 0.297526
    Train Epoch: 7 [22400/60000 (37%)]	Loss: 0.473823
    Train Epoch: 7 [23040/60000 (38%)]	Loss: 0.535452
    Train Epoch: 7 [23680/60000 (39%)]	Loss: 0.457003
    Train Epoch: 7 [24320/60000 (41%)]	Loss: 0.428764
    Train Epoch: 7 [24960/60000 (42%)]	Loss: 0.437032
    Train Epoch: 7 [25600/60000 (43%)]	Loss: 0.626992
    Train Epoch: 7 [26240/60000 (44%)]	Loss: 0.401498
    Train Epoch: 7 [26880/60000 (45%)]	Loss: 0.341814
    Train Epoch: 7 [27520/60000 (46%)]	Loss: 0.347058
    Train Epoch: 7 [28160/60000 (47%)]	Loss: 0.592646
    Train Epoch: 7 [28800/60000 (48%)]	Loss: 0.486121
    Train Epoch: 7 [29440/60000 (49%)]	Loss: 0.521025
    Train Epoch: 7 [30080/60000 (50%)]	Loss: 0.396132
    Train Epoch: 7 [30720/60000 (51%)]	Loss: 0.568312
    Train Epoch: 7 [31360/60000 (52%)]	Loss: 0.475081
    Train Epoch: 7 [32000/60000 (53%)]	Loss: 0.496030
    Train Epoch: 7 [32640/60000 (54%)]	Loss: 0.321438
    Train Epoch: 7 [33280/60000 (55%)]	Loss: 0.361846
    Train Epoch: 7 [33920/60000 (57%)]	Loss: 0.436478
    Train Epoch: 7 [34560/60000 (58%)]	Loss: 0.532364
    Train Epoch: 7 [35200/60000 (59%)]	Loss: 0.510952
    Train Epoch: 7 [35840/60000 (60%)]	Loss: 0.645716
    Train Epoch: 7 [36480/60000 (61%)]	Loss: 0.459234
    Train Epoch: 7 [37120/60000 (62%)]	Loss: 0.372446
    Train Epoch: 7 [37760/60000 (63%)]	Loss: 0.232452
    Train Epoch: 7 [38400/60000 (64%)]	Loss: 0.349685
    Train Epoch: 7 [39040/60000 (65%)]	Loss: 0.594316
    Train Epoch: 7 [39680/60000 (66%)]	Loss: 0.716787
    Train Epoch: 7 [40320/60000 (67%)]	Loss: 0.736326
    Train Epoch: 7 [40960/60000 (68%)]	Loss: 0.434927
    Train Epoch: 7 [41600/60000 (69%)]	Loss: 0.504802
    Train Epoch: 7 [42240/60000 (70%)]	Loss: 0.458648
    Train Epoch: 7 [42880/60000 (71%)]	Loss: 0.433149
    Train Epoch: 7 [43520/60000 (72%)]	Loss: 0.291753
    Train Epoch: 7 [44160/60000 (74%)]	Loss: 0.414159
    Train Epoch: 7 [44800/60000 (75%)]	Loss: 0.387175
    Train Epoch: 7 [45440/60000 (76%)]	Loss: 0.412587
    Train Epoch: 7 [46080/60000 (77%)]	Loss: 0.396877
    Train Epoch: 7 [46720/60000 (78%)]	Loss: 0.497912
    Train Epoch: 7 [47360/60000 (79%)]	Loss: 0.428156
    Train Epoch: 7 [48000/60000 (80%)]	Loss: 0.457888
    Train Epoch: 7 [48640/60000 (81%)]	Loss: 0.519679
    Train Epoch: 7 [49280/60000 (82%)]	Loss: 0.357949
    Train Epoch: 7 [49920/60000 (83%)]	Loss: 0.349140
    Train Epoch: 7 [50560/60000 (84%)]	Loss: 0.389948
    Train Epoch: 7 [51200/60000 (85%)]	Loss: 0.426888
    Train Epoch: 7 [51840/60000 (86%)]	Loss: 0.348459
    Train Epoch: 7 [52480/60000 (87%)]	Loss: 0.596195
    Train Epoch: 7 [53120/60000 (88%)]	Loss: 0.567125
    Train Epoch: 7 [53760/60000 (90%)]	Loss: 0.301156
    Train Epoch: 7 [54400/60000 (91%)]	Loss: 0.650556
    Train Epoch: 7 [55040/60000 (92%)]	Loss: 0.716237
    Train Epoch: 7 [55680/60000 (93%)]	Loss: 0.478880
    Train Epoch: 7 [56320/60000 (94%)]	Loss: 0.421738
    Train Epoch: 7 [56960/60000 (95%)]	Loss: 0.435452
    Train Epoch: 7 [57600/60000 (96%)]	Loss: 0.639110
    Train Epoch: 7 [58240/60000 (97%)]	Loss: 0.387537
    Train Epoch: 7 [58880/60000 (98%)]	Loss: 0.839672
    Train Epoch: 7 [59520/60000 (99%)]	Loss: 0.409901
    
    Test set: Average loss: 0.2244, Accuracy: 9333/10000 (93%)
    
    Train Epoch: 8 [0/60000 (0%)]	Loss: 0.469116
    Train Epoch: 8 [640/60000 (1%)]	Loss: 0.369547
    Train Epoch: 8 [1280/60000 (2%)]	Loss: 0.205326
    Train Epoch: 8 [1920/60000 (3%)]	Loss: 0.377605
    Train Epoch: 8 [2560/60000 (4%)]	Loss: 0.759715
    Train Epoch: 8 [3200/60000 (5%)]	Loss: 0.435700
    Train Epoch: 8 [3840/60000 (6%)]	Loss: 0.496598
    Train Epoch: 8 [4480/60000 (7%)]	Loss: 0.382843
    Train Epoch: 8 [5120/60000 (9%)]	Loss: 0.572180
    Train Epoch: 8 [5760/60000 (10%)]	Loss: 0.510329
    Train Epoch: 8 [6400/60000 (11%)]	Loss: 0.479855
    Train Epoch: 8 [7040/60000 (12%)]	Loss: 0.630407
    Train Epoch: 8 [7680/60000 (13%)]	Loss: 0.418155
    Train Epoch: 8 [8320/60000 (14%)]	Loss: 0.401250
    Train Epoch: 8 [8960/60000 (15%)]	Loss: 0.618375
    Train Epoch: 8 [9600/60000 (16%)]	Loss: 0.614910
    Train Epoch: 8 [10240/60000 (17%)]	Loss: 0.318959
    Train Epoch: 8 [10880/60000 (18%)]	Loss: 0.337133
    Train Epoch: 8 [11520/60000 (19%)]	Loss: 0.797270
    Train Epoch: 8 [12160/60000 (20%)]	Loss: 0.405077
    Train Epoch: 8 [12800/60000 (21%)]	Loss: 0.660094
    Train Epoch: 8 [13440/60000 (22%)]	Loss: 0.607702
    Train Epoch: 8 [14080/60000 (23%)]	Loss: 0.496708
    Train Epoch: 8 [14720/60000 (25%)]	Loss: 0.288580
    Train Epoch: 8 [15360/60000 (26%)]	Loss: 0.542240
    Train Epoch: 8 [16000/60000 (27%)]	Loss: 0.460526
    Train Epoch: 8 [16640/60000 (28%)]	Loss: 0.513786
    Train Epoch: 8 [17280/60000 (29%)]	Loss: 0.357062
    Train Epoch: 8 [17920/60000 (30%)]	Loss: 0.301969
    Train Epoch: 8 [18560/60000 (31%)]	Loss: 0.418003
    Train Epoch: 8 [19200/60000 (32%)]	Loss: 0.445466
    Train Epoch: 8 [19840/60000 (33%)]	Loss: 0.381778
    Train Epoch: 8 [20480/60000 (34%)]	Loss: 0.454850
    Train Epoch: 8 [21120/60000 (35%)]	Loss: 0.311810
    Train Epoch: 8 [21760/60000 (36%)]	Loss: 0.547684
    Train Epoch: 8 [22400/60000 (37%)]	Loss: 0.196216
    Train Epoch: 8 [23040/60000 (38%)]	Loss: 0.286038
    Train Epoch: 8 [23680/60000 (39%)]	Loss: 0.477280
    Train Epoch: 8 [24320/60000 (41%)]	Loss: 0.818387
    Train Epoch: 8 [24960/60000 (42%)]	Loss: 0.514256
    Train Epoch: 8 [25600/60000 (43%)]	Loss: 0.455588
    Train Epoch: 8 [26240/60000 (44%)]	Loss: 0.365949
    Train Epoch: 8 [26880/60000 (45%)]	Loss: 0.358122
    Train Epoch: 8 [27520/60000 (46%)]	Loss: 0.453270
    Train Epoch: 8 [28160/60000 (47%)]	Loss: 0.543010
    Train Epoch: 8 [28800/60000 (48%)]	Loss: 0.643081
    Train Epoch: 8 [29440/60000 (49%)]	Loss: 0.510997
    Train Epoch: 8 [30080/60000 (50%)]	Loss: 0.316055
    Train Epoch: 8 [30720/60000 (51%)]	Loss: 0.675488
    Train Epoch: 8 [31360/60000 (52%)]	Loss: 0.303624
    Train Epoch: 8 [32000/60000 (53%)]	Loss: 0.449534
    Train Epoch: 8 [32640/60000 (54%)]	Loss: 0.451440
    Train Epoch: 8 [33280/60000 (55%)]	Loss: 0.478363
    Train Epoch: 8 [33920/60000 (57%)]	Loss: 0.425090
    Train Epoch: 8 [34560/60000 (58%)]	Loss: 0.211939
    Train Epoch: 8 [35200/60000 (59%)]	Loss: 0.356067
    Train Epoch: 8 [35840/60000 (60%)]	Loss: 0.646257
    Train Epoch: 8 [36480/60000 (61%)]	Loss: 0.643568
    Train Epoch: 8 [37120/60000 (62%)]	Loss: 0.322013
    Train Epoch: 8 [37760/60000 (63%)]	Loss: 0.407144
    Train Epoch: 8 [38400/60000 (64%)]	Loss: 0.543189
    Train Epoch: 8 [39040/60000 (65%)]	Loss: 0.287051
    Train Epoch: 8 [39680/60000 (66%)]	Loss: 0.351675
    Train Epoch: 8 [40320/60000 (67%)]	Loss: 0.288524
    Train Epoch: 8 [40960/60000 (68%)]	Loss: 0.453518
    Train Epoch: 8 [41600/60000 (69%)]	Loss: 0.253906
    Train Epoch: 8 [42240/60000 (70%)]	Loss: 0.512110
    Train Epoch: 8 [42880/60000 (71%)]	Loss: 0.590715
    Train Epoch: 8 [43520/60000 (72%)]	Loss: 0.325584
    Train Epoch: 8 [44160/60000 (74%)]	Loss: 0.482525
    Train Epoch: 8 [44800/60000 (75%)]	Loss: 0.337738
    Train Epoch: 8 [45440/60000 (76%)]	Loss: 0.318561
    Train Epoch: 8 [46080/60000 (77%)]	Loss: 0.341067
    Train Epoch: 8 [46720/60000 (78%)]	Loss: 0.545488
    Train Epoch: 8 [47360/60000 (79%)]	Loss: 0.402002
    Train Epoch: 8 [48000/60000 (80%)]	Loss: 0.231705
    Train Epoch: 8 [48640/60000 (81%)]	Loss: 0.242957
    Train Epoch: 8 [49280/60000 (82%)]	Loss: 0.426707
    Train Epoch: 8 [49920/60000 (83%)]	Loss: 0.341219
    Train Epoch: 8 [50560/60000 (84%)]	Loss: 0.422939
    Train Epoch: 8 [51200/60000 (85%)]	Loss: 0.410271
    Train Epoch: 8 [51840/60000 (86%)]	Loss: 0.443087
    Train Epoch: 8 [52480/60000 (87%)]	Loss: 0.273087
    Train Epoch: 8 [53120/60000 (88%)]	Loss: 0.300433
    Train Epoch: 8 [53760/60000 (90%)]	Loss: 0.408493
    Train Epoch: 8 [54400/60000 (91%)]	Loss: 0.410628
    Train Epoch: 8 [55040/60000 (92%)]	Loss: 0.481743
    Train Epoch: 8 [55680/60000 (93%)]	Loss: 0.532843
    Train Epoch: 8 [56320/60000 (94%)]	Loss: 0.255752
    Train Epoch: 8 [56960/60000 (95%)]	Loss: 0.287013
    Train Epoch: 8 [57600/60000 (96%)]	Loss: 0.429710
    Train Epoch: 8 [58240/60000 (97%)]	Loss: 0.377912
    Train Epoch: 8 [58880/60000 (98%)]	Loss: 0.560696
    Train Epoch: 8 [59520/60000 (99%)]	Loss: 0.380459
    
    Test set: Average loss: 0.2163, Accuracy: 9362/10000 (94%)
    
    Train Epoch: 9 [0/60000 (0%)]	Loss: 0.585349
    Train Epoch: 9 [640/60000 (1%)]	Loss: 0.493247
    Train Epoch: 9 [1280/60000 (2%)]	Loss: 0.391806
    Train Epoch: 9 [1920/60000 (3%)]	Loss: 0.493008
    Train Epoch: 9 [2560/60000 (4%)]	Loss: 0.448494
    Train Epoch: 9 [3200/60000 (5%)]	Loss: 0.325095
    Train Epoch: 9 [3840/60000 (6%)]	Loss: 0.695937
    Train Epoch: 9 [4480/60000 (7%)]	Loss: 0.266650
    Train Epoch: 9 [5120/60000 (9%)]	Loss: 0.420215
    Train Epoch: 9 [5760/60000 (10%)]	Loss: 0.353440
    Train Epoch: 9 [6400/60000 (11%)]	Loss: 0.341078
    Train Epoch: 9 [7040/60000 (12%)]	Loss: 0.439247
    Train Epoch: 9 [7680/60000 (13%)]	Loss: 0.214538
    Train Epoch: 9 [8320/60000 (14%)]	Loss: 0.469013
    Train Epoch: 9 [8960/60000 (15%)]	Loss: 0.341292
    Train Epoch: 9 [9600/60000 (16%)]	Loss: 0.785742
    Train Epoch: 9 [10240/60000 (17%)]	Loss: 0.466753
    Train Epoch: 9 [10880/60000 (18%)]	Loss: 0.418933
    Train Epoch: 9 [11520/60000 (19%)]	Loss: 0.352861
    Train Epoch: 9 [12160/60000 (20%)]	Loss: 0.330622
    Train Epoch: 9 [12800/60000 (21%)]	Loss: 0.394191
    Train Epoch: 9 [13440/60000 (22%)]	Loss: 0.304991
    Train Epoch: 9 [14080/60000 (23%)]	Loss: 0.291812
    Train Epoch: 9 [14720/60000 (25%)]	Loss: 0.460314
    Train Epoch: 9 [15360/60000 (26%)]	Loss: 0.462962
    Train Epoch: 9 [16000/60000 (27%)]	Loss: 0.573508
    Train Epoch: 9 [16640/60000 (28%)]	Loss: 0.424545
    Train Epoch: 9 [17280/60000 (29%)]	Loss: 0.314216
    Train Epoch: 9 [17920/60000 (30%)]	Loss: 0.399477
    Train Epoch: 9 [18560/60000 (31%)]	Loss: 0.281409
    Train Epoch: 9 [19200/60000 (32%)]	Loss: 0.491287
    Train Epoch: 9 [19840/60000 (33%)]	Loss: 0.478374
    Train Epoch: 9 [20480/60000 (34%)]	Loss: 0.580464
    Train Epoch: 9 [21120/60000 (35%)]	Loss: 0.456699
    Train Epoch: 9 [21760/60000 (36%)]	Loss: 0.328621
    Train Epoch: 9 [22400/60000 (37%)]	Loss: 0.444201
    Train Epoch: 9 [23040/60000 (38%)]	Loss: 0.337673
    Train Epoch: 9 [23680/60000 (39%)]	Loss: 0.385429
    Train Epoch: 9 [24320/60000 (41%)]	Loss: 0.408061
    Train Epoch: 9 [24960/60000 (42%)]	Loss: 0.261543
    Train Epoch: 9 [25600/60000 (43%)]	Loss: 0.307577
    Train Epoch: 9 [26240/60000 (44%)]	Loss: 0.340200
    Train Epoch: 9 [26880/60000 (45%)]	Loss: 0.251913
    Train Epoch: 9 [27520/60000 (46%)]	Loss: 0.269230
    Train Epoch: 9 [28160/60000 (47%)]	Loss: 0.456552
    Train Epoch: 9 [28800/60000 (48%)]	Loss: 0.598232
    Train Epoch: 9 [29440/60000 (49%)]	Loss: 0.418178
    Train Epoch: 9 [30080/60000 (50%)]	Loss: 0.356407
    Train Epoch: 9 [30720/60000 (51%)]	Loss: 0.392345
    Train Epoch: 9 [31360/60000 (52%)]	Loss: 0.379441
    Train Epoch: 9 [32000/60000 (53%)]	Loss: 0.465714
    Train Epoch: 9 [32640/60000 (54%)]	Loss: 0.367991
    Train Epoch: 9 [33280/60000 (55%)]	Loss: 0.285676
    Train Epoch: 9 [33920/60000 (57%)]	Loss: 0.243431
    Train Epoch: 9 [34560/60000 (58%)]	Loss: 0.355942
    Train Epoch: 9 [35200/60000 (59%)]	Loss: 0.374828
    Train Epoch: 9 [35840/60000 (60%)]	Loss: 0.277245
    Train Epoch: 9 [36480/60000 (61%)]	Loss: 0.273998
    Train Epoch: 9 [37120/60000 (62%)]	Loss: 0.406776
    Train Epoch: 9 [37760/60000 (63%)]	Loss: 0.651791
    Train Epoch: 9 [38400/60000 (64%)]	Loss: 0.417006
    Train Epoch: 9 [39040/60000 (65%)]	Loss: 0.287786
    Train Epoch: 9 [39680/60000 (66%)]	Loss: 0.592247
    Train Epoch: 9 [40320/60000 (67%)]	Loss: 0.317201
    Train Epoch: 9 [40960/60000 (68%)]	Loss: 0.324063
    Train Epoch: 9 [41600/60000 (69%)]	Loss: 0.393426
    Train Epoch: 9 [42240/60000 (70%)]	Loss: 0.413506
    Train Epoch: 9 [42880/60000 (71%)]	Loss: 0.633300
    Train Epoch: 9 [43520/60000 (72%)]	Loss: 0.276478
    Train Epoch: 9 [44160/60000 (74%)]	Loss: 0.473216
    Train Epoch: 9 [44800/60000 (75%)]	Loss: 0.327980
    Train Epoch: 9 [45440/60000 (76%)]	Loss: 0.727830
    Train Epoch: 9 [46080/60000 (77%)]	Loss: 0.416605
    Train Epoch: 9 [46720/60000 (78%)]	Loss: 0.407100
    Train Epoch: 9 [47360/60000 (79%)]	Loss: 0.375050
    Train Epoch: 9 [48000/60000 (80%)]	Loss: 0.488991
    Train Epoch: 9 [48640/60000 (81%)]	Loss: 0.413114
    Train Epoch: 9 [49280/60000 (82%)]	Loss: 0.520725
    Train Epoch: 9 [49920/60000 (83%)]	Loss: 0.420221
    Train Epoch: 9 [50560/60000 (84%)]	Loss: 0.599522
    Train Epoch: 9 [51200/60000 (85%)]	Loss: 0.490780
    Train Epoch: 9 [51840/60000 (86%)]	Loss: 0.228232
    Train Epoch: 9 [52480/60000 (87%)]	Loss: 0.347773
    Train Epoch: 9 [53120/60000 (88%)]	Loss: 0.476633
    Train Epoch: 9 [53760/60000 (90%)]	Loss: 0.256655
    Train Epoch: 9 [54400/60000 (91%)]	Loss: 0.396474
    Train Epoch: 9 [55040/60000 (92%)]	Loss: 0.328017
    Train Epoch: 9 [55680/60000 (93%)]	Loss: 0.355085
    Train Epoch: 9 [56320/60000 (94%)]	Loss: 0.354232
    Train Epoch: 9 [56960/60000 (95%)]	Loss: 0.360218
    Train Epoch: 9 [57600/60000 (96%)]	Loss: 0.332372
    Train Epoch: 9 [58240/60000 (97%)]	Loss: 0.364290
    Train Epoch: 9 [58880/60000 (98%)]	Loss: 0.261339
    Train Epoch: 9 [59520/60000 (99%)]	Loss: 0.250586
    
    Test set: Average loss: 0.2151, Accuracy: 9366/10000 (94%)
    
    Train Epoch: 10 [0/60000 (0%)]	Loss: 0.438674
    Train Epoch: 10 [640/60000 (1%)]	Loss: 0.447094
    Train Epoch: 10 [1280/60000 (2%)]	Loss: 0.303145
    Train Epoch: 10 [1920/60000 (3%)]	Loss: 0.327250
    Train Epoch: 10 [2560/60000 (4%)]	Loss: 0.238297
    Train Epoch: 10 [3200/60000 (5%)]	Loss: 0.383331
    Train Epoch: 10 [3840/60000 (6%)]	Loss: 0.382009
    Train Epoch: 10 [4480/60000 (7%)]	Loss: 0.389430
    Train Epoch: 10 [5120/60000 (9%)]	Loss: 0.295570
    Train Epoch: 10 [5760/60000 (10%)]	Loss: 0.259864
    Train Epoch: 10 [6400/60000 (11%)]	Loss: 0.495971
    Train Epoch: 10 [7040/60000 (12%)]	Loss: 0.361642
    Train Epoch: 10 [7680/60000 (13%)]	Loss: 0.765770
    Train Epoch: 10 [8320/60000 (14%)]	Loss: 0.403898
    Train Epoch: 10 [8960/60000 (15%)]	Loss: 0.209247
    Train Epoch: 10 [9600/60000 (16%)]	Loss: 0.482393
    Train Epoch: 10 [10240/60000 (17%)]	Loss: 0.459047
    Train Epoch: 10 [10880/60000 (18%)]	Loss: 0.505761
    Train Epoch: 10 [11520/60000 (19%)]	Loss: 0.433308
    Train Epoch: 10 [12160/60000 (20%)]	Loss: 0.354521
    Train Epoch: 10 [12800/60000 (21%)]	Loss: 0.233018
    Train Epoch: 10 [13440/60000 (22%)]	Loss: 0.390475
    Train Epoch: 10 [14080/60000 (23%)]	Loss: 0.245935
    Train Epoch: 10 [14720/60000 (25%)]	Loss: 0.398529
    Train Epoch: 10 [15360/60000 (26%)]	Loss: 0.393017
    Train Epoch: 10 [16000/60000 (27%)]	Loss: 0.364165
    Train Epoch: 10 [16640/60000 (28%)]	Loss: 0.657179
    Train Epoch: 10 [17280/60000 (29%)]	Loss: 0.199565
    Train Epoch: 10 [17920/60000 (30%)]	Loss: 0.373812
    Train Epoch: 10 [18560/60000 (31%)]	Loss: 0.395341
    Train Epoch: 10 [19200/60000 (32%)]	Loss: 0.367142
    Train Epoch: 10 [19840/60000 (33%)]	Loss: 0.420444
    Train Epoch: 10 [20480/60000 (34%)]	Loss: 0.411721
    Train Epoch: 10 [21120/60000 (35%)]	Loss: 0.406184
    Train Epoch: 10 [21760/60000 (36%)]	Loss: 0.309357
    Train Epoch: 10 [22400/60000 (37%)]	Loss: 0.397584
    Train Epoch: 10 [23040/60000 (38%)]	Loss: 0.699485
    Train Epoch: 10 [23680/60000 (39%)]	Loss: 0.672688
    Train Epoch: 10 [24320/60000 (41%)]	Loss: 0.383668
    Train Epoch: 10 [24960/60000 (42%)]	Loss: 0.443057
    Train Epoch: 10 [25600/60000 (43%)]	Loss: 0.409219
    Train Epoch: 10 [26240/60000 (44%)]	Loss: 0.311079
    Train Epoch: 10 [26880/60000 (45%)]	Loss: 0.367074
    Train Epoch: 10 [27520/60000 (46%)]	Loss: 0.279823
    Train Epoch: 10 [28160/60000 (47%)]	Loss: 0.337272
    Train Epoch: 10 [28800/60000 (48%)]	Loss: 0.485712
    Train Epoch: 10 [29440/60000 (49%)]	Loss: 0.345926
    Train Epoch: 10 [30080/60000 (50%)]	Loss: 0.424248
    Train Epoch: 10 [30720/60000 (51%)]	Loss: 0.322441
    Train Epoch: 10 [31360/60000 (52%)]	Loss: 0.283901
    Train Epoch: 10 [32000/60000 (53%)]	Loss: 0.640330
    Train Epoch: 10 [32640/60000 (54%)]	Loss: 0.342491
    Train Epoch: 10 [33280/60000 (55%)]	Loss: 0.343811
    Train Epoch: 10 [33920/60000 (57%)]	Loss: 0.392110
    Train Epoch: 10 [34560/60000 (58%)]	Loss: 0.433466
    Train Epoch: 10 [35200/60000 (59%)]	Loss: 0.341572
    Train Epoch: 10 [35840/60000 (60%)]	Loss: 0.394995
    Train Epoch: 10 [36480/60000 (61%)]	Loss: 0.332045
    Train Epoch: 10 [37120/60000 (62%)]	Loss: 0.276502
    Train Epoch: 10 [37760/60000 (63%)]	Loss: 0.292657
    Train Epoch: 10 [38400/60000 (64%)]	Loss: 0.455167
    Train Epoch: 10 [39040/60000 (65%)]	Loss: 0.297509
    Train Epoch: 10 [39680/60000 (66%)]	Loss: 0.640905
    Train Epoch: 10 [40320/60000 (67%)]	Loss: 0.422916
    Train Epoch: 10 [40960/60000 (68%)]	Loss: 0.473346
    Train Epoch: 10 [41600/60000 (69%)]	Loss: 0.491301
    Train Epoch: 10 [42240/60000 (70%)]	Loss: 0.346930
    Train Epoch: 10 [42880/60000 (71%)]	Loss: 0.572828
    Train Epoch: 10 [43520/60000 (72%)]	Loss: 0.365607
    Train Epoch: 10 [44160/60000 (74%)]	Loss: 0.317555
    Train Epoch: 10 [44800/60000 (75%)]	Loss: 0.468911
    Train Epoch: 10 [45440/60000 (76%)]	Loss: 0.496311
    Train Epoch: 10 [46080/60000 (77%)]	Loss: 0.696476
    Train Epoch: 10 [46720/60000 (78%)]	Loss: 0.359581
    Train Epoch: 10 [47360/60000 (79%)]	Loss: 0.419243
    Train Epoch: 10 [48000/60000 (80%)]	Loss: 0.303316
    Train Epoch: 10 [48640/60000 (81%)]	Loss: 0.383326
    Train Epoch: 10 [49280/60000 (82%)]	Loss: 0.268373
    Train Epoch: 10 [49920/60000 (83%)]	Loss: 0.413617
    Train Epoch: 10 [50560/60000 (84%)]	Loss: 0.454594
    Train Epoch: 10 [51200/60000 (85%)]	Loss: 0.359162
    Train Epoch: 10 [51840/60000 (86%)]	Loss: 0.630098
    Train Epoch: 10 [52480/60000 (87%)]	Loss: 0.521164
    Train Epoch: 10 [53120/60000 (88%)]	Loss: 0.247818
    Train Epoch: 10 [53760/60000 (90%)]	Loss: 0.330510
    Train Epoch: 10 [54400/60000 (91%)]	Loss: 0.343167
    Train Epoch: 10 [55040/60000 (92%)]	Loss: 0.380157
    Train Epoch: 10 [55680/60000 (93%)]	Loss: 0.395422
    Train Epoch: 10 [56320/60000 (94%)]	Loss: 0.687743
    Train Epoch: 10 [56960/60000 (95%)]	Loss: 0.470193
    Train Epoch: 10 [57600/60000 (96%)]	Loss: 0.473724
    Train Epoch: 10 [58240/60000 (97%)]	Loss: 0.361690
    Train Epoch: 10 [58880/60000 (98%)]	Loss: 0.349370
    Train Epoch: 10 [59520/60000 (99%)]	Loss: 0.385800
    
    Test set: Average loss: 0.2124, Accuracy: 9367/10000 (94%)
    
    Train Epoch: 11 [0/60000 (0%)]	Loss: 0.426175
    Train Epoch: 11 [640/60000 (1%)]	Loss: 0.170051
    Train Epoch: 11 [1280/60000 (2%)]	Loss: 0.250144
    Train Epoch: 11 [1920/60000 (3%)]	Loss: 0.172225
    Train Epoch: 11 [2560/60000 (4%)]	Loss: 0.421107
    Train Epoch: 11 [3200/60000 (5%)]	Loss: 0.380877
    Train Epoch: 11 [3840/60000 (6%)]	Loss: 0.230398
    Train Epoch: 11 [4480/60000 (7%)]	Loss: 0.477564
    Train Epoch: 11 [5120/60000 (9%)]	Loss: 0.395525
    Train Epoch: 11 [5760/60000 (10%)]	Loss: 0.270284
    Train Epoch: 11 [6400/60000 (11%)]	Loss: 0.310442
    Train Epoch: 11 [7040/60000 (12%)]	Loss: 0.285872
    Train Epoch: 11 [7680/60000 (13%)]	Loss: 0.333100
    Train Epoch: 11 [8320/60000 (14%)]	Loss: 0.269915
    Train Epoch: 11 [8960/60000 (15%)]	Loss: 0.340484
    Train Epoch: 11 [9600/60000 (16%)]	Loss: 0.433937
    Train Epoch: 11 [10240/60000 (17%)]	Loss: 0.552323
    Train Epoch: 11 [10880/60000 (18%)]	Loss: 0.532913
    Train Epoch: 11 [11520/60000 (19%)]	Loss: 0.495746
    Train Epoch: 11 [12160/60000 (20%)]	Loss: 0.303816
    Train Epoch: 11 [12800/60000 (21%)]	Loss: 0.264450
    Train Epoch: 11 [13440/60000 (22%)]	Loss: 0.436694
    Train Epoch: 11 [14080/60000 (23%)]	Loss: 0.440698
    Train Epoch: 11 [14720/60000 (25%)]	Loss: 0.422328
    Train Epoch: 11 [15360/60000 (26%)]	Loss: 0.415076
    Train Epoch: 11 [16000/60000 (27%)]	Loss: 0.595344
    Train Epoch: 11 [16640/60000 (28%)]	Loss: 0.246912
    Train Epoch: 11 [17280/60000 (29%)]	Loss: 0.261348
    Train Epoch: 11 [17920/60000 (30%)]	Loss: 0.420687
    Train Epoch: 11 [18560/60000 (31%)]	Loss: 0.309478
    Train Epoch: 11 [19200/60000 (32%)]	Loss: 0.351695
    Train Epoch: 11 [19840/60000 (33%)]	Loss: 0.521406
    Train Epoch: 11 [20480/60000 (34%)]	Loss: 0.290906
    Train Epoch: 11 [21120/60000 (35%)]	Loss: 0.364633
    Train Epoch: 11 [21760/60000 (36%)]	Loss: 0.324598
    Train Epoch: 11 [22400/60000 (37%)]	Loss: 0.504305
    Train Epoch: 11 [23040/60000 (38%)]	Loss: 0.565828
    Train Epoch: 11 [23680/60000 (39%)]	Loss: 0.530418
    Train Epoch: 11 [24320/60000 (41%)]	Loss: 0.394786
    Train Epoch: 11 [24960/60000 (42%)]	Loss: 0.360259
    Train Epoch: 11 [25600/60000 (43%)]	Loss: 0.332048
    Train Epoch: 11 [26240/60000 (44%)]	Loss: 0.277467
    Train Epoch: 11 [26880/60000 (45%)]	Loss: 0.392917
    Train Epoch: 11 [27520/60000 (46%)]	Loss: 0.343030
    Train Epoch: 11 [28160/60000 (47%)]	Loss: 0.575351
    Train Epoch: 11 [28800/60000 (48%)]	Loss: 0.234557
    Train Epoch: 11 [29440/60000 (49%)]	Loss: 0.345107
    Train Epoch: 11 [30080/60000 (50%)]	Loss: 0.250498
    Train Epoch: 11 [30720/60000 (51%)]	Loss: 0.252944
    Train Epoch: 11 [31360/60000 (52%)]	Loss: 0.339441
    Train Epoch: 11 [32000/60000 (53%)]	Loss: 0.419631
    Train Epoch: 11 [32640/60000 (54%)]	Loss: 0.299459
    Train Epoch: 11 [33280/60000 (55%)]	Loss: 0.496848
    Train Epoch: 11 [33920/60000 (57%)]	Loss: 0.298093
    Train Epoch: 11 [34560/60000 (58%)]	Loss: 0.502162
    Train Epoch: 11 [35200/60000 (59%)]	Loss: 0.255059
    Train Epoch: 11 [35840/60000 (60%)]	Loss: 0.411274
    Train Epoch: 11 [36480/60000 (61%)]	Loss: 0.523597
    Train Epoch: 11 [37120/60000 (62%)]	Loss: 0.413543
    Train Epoch: 11 [37760/60000 (63%)]	Loss: 0.416163
    Train Epoch: 11 [38400/60000 (64%)]	Loss: 0.369535
    Train Epoch: 11 [39040/60000 (65%)]	Loss: 0.611558
    Train Epoch: 11 [39680/60000 (66%)]	Loss: 0.304744
    Train Epoch: 11 [40320/60000 (67%)]	Loss: 0.430891
    Train Epoch: 11 [40960/60000 (68%)]	Loss: 0.405095
    Train Epoch: 11 [41600/60000 (69%)]	Loss: 0.459111
    Train Epoch: 11 [42240/60000 (70%)]	Loss: 0.305776
    Train Epoch: 11 [42880/60000 (71%)]	Loss: 0.383718
    Train Epoch: 11 [43520/60000 (72%)]	Loss: 0.357237
    Train Epoch: 11 [44160/60000 (74%)]	Loss: 0.882389
    Train Epoch: 11 [44800/60000 (75%)]	Loss: 0.515517
    Train Epoch: 11 [45440/60000 (76%)]	Loss: 0.431814
    Train Epoch: 11 [46080/60000 (77%)]	Loss: 0.502057
    Train Epoch: 11 [46720/60000 (78%)]	Loss: 0.363643
    Train Epoch: 11 [47360/60000 (79%)]	Loss: 0.300866
    Train Epoch: 11 [48000/60000 (80%)]	Loss: 0.379479
    Train Epoch: 11 [48640/60000 (81%)]	Loss: 0.409872
    Train Epoch: 11 [49280/60000 (82%)]	Loss: 0.459707
    Train Epoch: 11 [49920/60000 (83%)]	Loss: 0.407088
    Train Epoch: 11 [50560/60000 (84%)]	Loss: 0.442198
    Train Epoch: 11 [51200/60000 (85%)]	Loss: 0.360245
    Train Epoch: 11 [51840/60000 (86%)]	Loss: 0.391902
    Train Epoch: 11 [52480/60000 (87%)]	Loss: 0.690278
    Train Epoch: 11 [53120/60000 (88%)]	Loss: 0.578411
    Train Epoch: 11 [53760/60000 (90%)]	Loss: 0.317039
    Train Epoch: 11 [54400/60000 (91%)]	Loss: 0.361648
    Train Epoch: 11 [55040/60000 (92%)]	Loss: 0.256818
    Train Epoch: 11 [55680/60000 (93%)]	Loss: 0.305927
    Train Epoch: 11 [56320/60000 (94%)]	Loss: 0.334767
    Train Epoch: 11 [56960/60000 (95%)]	Loss: 0.393670
    Train Epoch: 11 [57600/60000 (96%)]	Loss: 0.357648
    Train Epoch: 11 [58240/60000 (97%)]	Loss: 0.281211
    Train Epoch: 11 [58880/60000 (98%)]	Loss: 0.324076
    Train Epoch: 11 [59520/60000 (99%)]	Loss: 0.372610
    
    Test set: Average loss: 0.2098, Accuracy: 9373/10000 (94%)
    
    Train Epoch: 12 [0/60000 (0%)]	Loss: 0.392381
    Train Epoch: 12 [640/60000 (1%)]	Loss: 0.296244
    Train Epoch: 12 [1280/60000 (2%)]	Loss: 0.375837
    Train Epoch: 12 [1920/60000 (3%)]	Loss: 0.511141
    Train Epoch: 12 [2560/60000 (4%)]	Loss: 0.328571
    Train Epoch: 12 [3200/60000 (5%)]	Loss: 0.407022
    Train Epoch: 12 [3840/60000 (6%)]	Loss: 0.298561
    Train Epoch: 12 [4480/60000 (7%)]	Loss: 0.294834
    Train Epoch: 12 [5120/60000 (9%)]	Loss: 0.459634
    Train Epoch: 12 [5760/60000 (10%)]	Loss: 0.427800
    Train Epoch: 12 [6400/60000 (11%)]	Loss: 0.315486
    Train Epoch: 12 [7040/60000 (12%)]	Loss: 0.369394
    Train Epoch: 12 [7680/60000 (13%)]	Loss: 0.383769
    Train Epoch: 12 [8320/60000 (14%)]	Loss: 0.360964
    Train Epoch: 12 [8960/60000 (15%)]	Loss: 0.565721
    Train Epoch: 12 [9600/60000 (16%)]	Loss: 0.339542
    Train Epoch: 12 [10240/60000 (17%)]	Loss: 0.318309
    Train Epoch: 12 [10880/60000 (18%)]	Loss: 0.354276
    Train Epoch: 12 [11520/60000 (19%)]	Loss: 0.729153
    Train Epoch: 12 [12160/60000 (20%)]	Loss: 0.637019
    Train Epoch: 12 [12800/60000 (21%)]	Loss: 0.311870
    Train Epoch: 12 [13440/60000 (22%)]	Loss: 0.475887
    Train Epoch: 12 [14080/60000 (23%)]	Loss: 0.593350
    Train Epoch: 12 [14720/60000 (25%)]	Loss: 0.401409
    Train Epoch: 12 [15360/60000 (26%)]	Loss: 0.340033
    Train Epoch: 12 [16000/60000 (27%)]	Loss: 0.268461
    Train Epoch: 12 [16640/60000 (28%)]	Loss: 0.246901
    Train Epoch: 12 [17280/60000 (29%)]	Loss: 0.220537
    Train Epoch: 12 [17920/60000 (30%)]	Loss: 0.343910
    Train Epoch: 12 [18560/60000 (31%)]	Loss: 0.404446
    Train Epoch: 12 [19200/60000 (32%)]	Loss: 0.390659
    Train Epoch: 12 [19840/60000 (33%)]	Loss: 0.428503
    Train Epoch: 12 [20480/60000 (34%)]	Loss: 0.349072
    Train Epoch: 12 [21120/60000 (35%)]	Loss: 0.486959
    Train Epoch: 12 [21760/60000 (36%)]	Loss: 0.328149
    Train Epoch: 12 [22400/60000 (37%)]	Loss: 0.516612
    Train Epoch: 12 [23040/60000 (38%)]	Loss: 0.457053
    Train Epoch: 12 [23680/60000 (39%)]	Loss: 0.608891
    Train Epoch: 12 [24320/60000 (41%)]	Loss: 0.689961
    Train Epoch: 12 [24960/60000 (42%)]	Loss: 0.294651
    Train Epoch: 12 [25600/60000 (43%)]	Loss: 0.393591
    Train Epoch: 12 [26240/60000 (44%)]	Loss: 0.338527
    Train Epoch: 12 [26880/60000 (45%)]	Loss: 0.577185
    Train Epoch: 12 [27520/60000 (46%)]	Loss: 0.353298
    Train Epoch: 12 [28160/60000 (47%)]	Loss: 0.622562
    Train Epoch: 12 [28800/60000 (48%)]	Loss: 0.282284
    Train Epoch: 12 [29440/60000 (49%)]	Loss: 0.313890
    Train Epoch: 12 [30080/60000 (50%)]	Loss: 0.351841
    Train Epoch: 12 [30720/60000 (51%)]	Loss: 0.396683
    Train Epoch: 12 [31360/60000 (52%)]	Loss: 0.525927
    Train Epoch: 12 [32000/60000 (53%)]	Loss: 0.234338
    Train Epoch: 12 [32640/60000 (54%)]	Loss: 0.462475
    Train Epoch: 12 [33280/60000 (55%)]	Loss: 0.566766
    Train Epoch: 12 [33920/60000 (57%)]	Loss: 0.384067
    Train Epoch: 12 [34560/60000 (58%)]	Loss: 0.281657
    Train Epoch: 12 [35200/60000 (59%)]	Loss: 0.392156
    Train Epoch: 12 [35840/60000 (60%)]	Loss: 0.567646
    Train Epoch: 12 [36480/60000 (61%)]	Loss: 0.294172
    Train Epoch: 12 [37120/60000 (62%)]	Loss: 0.395887
    Train Epoch: 12 [37760/60000 (63%)]	Loss: 0.241547
    Train Epoch: 12 [38400/60000 (64%)]	Loss: 0.475506
    Train Epoch: 12 [39040/60000 (65%)]	Loss: 0.444349
    Train Epoch: 12 [39680/60000 (66%)]	Loss: 0.590313
    Train Epoch: 12 [40320/60000 (67%)]	Loss: 0.380521
    Train Epoch: 12 [40960/60000 (68%)]	Loss: 0.319756
    Train Epoch: 12 [41600/60000 (69%)]	Loss: 0.419879
    Train Epoch: 12 [42240/60000 (70%)]	Loss: 0.384562
    Train Epoch: 12 [42880/60000 (71%)]	Loss: 0.234591
    Train Epoch: 12 [43520/60000 (72%)]	Loss: 0.330877
    Train Epoch: 12 [44160/60000 (74%)]	Loss: 0.697167
    Train Epoch: 12 [44800/60000 (75%)]	Loss: 0.272816
    Train Epoch: 12 [45440/60000 (76%)]	Loss: 0.415027
    Train Epoch: 12 [46080/60000 (77%)]	Loss: 0.403599
    Train Epoch: 12 [46720/60000 (78%)]	Loss: 0.350379
    Train Epoch: 12 [47360/60000 (79%)]	Loss: 0.210333
    Train Epoch: 12 [48000/60000 (80%)]	Loss: 0.350989
    Train Epoch: 12 [48640/60000 (81%)]	Loss: 0.421243
    Train Epoch: 12 [49280/60000 (82%)]	Loss: 0.257715
    Train Epoch: 12 [49920/60000 (83%)]	Loss: 0.430463
    Train Epoch: 12 [50560/60000 (84%)]	Loss: 0.436658
    Train Epoch: 12 [51200/60000 (85%)]	Loss: 0.385483
    Train Epoch: 12 [51840/60000 (86%)]	Loss: 0.449448
    Train Epoch: 12 [52480/60000 (87%)]	Loss: 0.369401
    Train Epoch: 12 [53120/60000 (88%)]	Loss: 0.380905
    Train Epoch: 12 [53760/60000 (90%)]	Loss: 0.391110
    Train Epoch: 12 [54400/60000 (91%)]	Loss: 0.381158
    Train Epoch: 12 [55040/60000 (92%)]	Loss: 0.317574
    Train Epoch: 12 [55680/60000 (93%)]	Loss: 0.616171
    Train Epoch: 12 [56320/60000 (94%)]	Loss: 0.333590
    Train Epoch: 12 [56960/60000 (95%)]	Loss: 0.460308
    Train Epoch: 12 [57600/60000 (96%)]	Loss: 0.586635
    Train Epoch: 12 [58240/60000 (97%)]	Loss: 0.323481
    Train Epoch: 12 [58880/60000 (98%)]	Loss: 0.410162
    Train Epoch: 12 [59520/60000 (99%)]	Loss: 0.475991
    
    Test set: Average loss: 0.2096, Accuracy: 9381/10000 (94%)
    
    Train Epoch: 13 [0/60000 (0%)]	Loss: 0.555876
    Train Epoch: 13 [640/60000 (1%)]	Loss: 0.298020
    Train Epoch: 13 [1280/60000 (2%)]	Loss: 0.341556
    Train Epoch: 13 [1920/60000 (3%)]	Loss: 0.387244
    Train Epoch: 13 [2560/60000 (4%)]	Loss: 0.299948
    Train Epoch: 13 [3200/60000 (5%)]	Loss: 0.352979
    Train Epoch: 13 [3840/60000 (6%)]	Loss: 0.445687
    Train Epoch: 13 [4480/60000 (7%)]	Loss: 0.223049
    Train Epoch: 13 [5120/60000 (9%)]	Loss: 0.494325
    Train Epoch: 13 [5760/60000 (10%)]	Loss: 0.749437
    Train Epoch: 13 [6400/60000 (11%)]	Loss: 0.404310
    Train Epoch: 13 [7040/60000 (12%)]	Loss: 0.337297
    Train Epoch: 13 [7680/60000 (13%)]	Loss: 0.434966
    Train Epoch: 13 [8320/60000 (14%)]	Loss: 0.401748
    Train Epoch: 13 [8960/60000 (15%)]	Loss: 0.340427
    Train Epoch: 13 [9600/60000 (16%)]	Loss: 0.614933
    Train Epoch: 13 [10240/60000 (17%)]	Loss: 0.428032
    Train Epoch: 13 [10880/60000 (18%)]	Loss: 0.520478
    Train Epoch: 13 [11520/60000 (19%)]	Loss: 0.343638
    Train Epoch: 13 [12160/60000 (20%)]	Loss: 0.282134
    Train Epoch: 13 [12800/60000 (21%)]	Loss: 0.236920
    Train Epoch: 13 [13440/60000 (22%)]	Loss: 0.331308
    Train Epoch: 13 [14080/60000 (23%)]	Loss: 0.342169
    Train Epoch: 13 [14720/60000 (25%)]	Loss: 0.494079
    Train Epoch: 13 [15360/60000 (26%)]	Loss: 0.566828
    Train Epoch: 13 [16000/60000 (27%)]	Loss: 0.515479
    Train Epoch: 13 [16640/60000 (28%)]	Loss: 0.546353
    Train Epoch: 13 [17280/60000 (29%)]	Loss: 0.462009
    Train Epoch: 13 [17920/60000 (30%)]	Loss: 0.547893
    Train Epoch: 13 [18560/60000 (31%)]	Loss: 0.519924
    Train Epoch: 13 [19200/60000 (32%)]	Loss: 0.445337
    Train Epoch: 13 [19840/60000 (33%)]	Loss: 0.254473
    Train Epoch: 13 [20480/60000 (34%)]	Loss: 0.351020
    Train Epoch: 13 [21120/60000 (35%)]	Loss: 0.388969
    Train Epoch: 13 [21760/60000 (36%)]	Loss: 0.285459
    Train Epoch: 13 [22400/60000 (37%)]	Loss: 0.308739
    Train Epoch: 13 [23040/60000 (38%)]	Loss: 0.501287
    Train Epoch: 13 [23680/60000 (39%)]	Loss: 0.392744
    Train Epoch: 13 [24320/60000 (41%)]	Loss: 0.490546
    Train Epoch: 13 [24960/60000 (42%)]	Loss: 0.407411
    Train Epoch: 13 [25600/60000 (43%)]	Loss: 0.557519
    Train Epoch: 13 [26240/60000 (44%)]	Loss: 0.407774
    Train Epoch: 13 [26880/60000 (45%)]	Loss: 0.313496
    Train Epoch: 13 [27520/60000 (46%)]	Loss: 0.470231
    Train Epoch: 13 [28160/60000 (47%)]	Loss: 0.457754
    Train Epoch: 13 [28800/60000 (48%)]	Loss: 0.314194
    Train Epoch: 13 [29440/60000 (49%)]	Loss: 0.395972
    Train Epoch: 13 [30080/60000 (50%)]	Loss: 0.575824
    Train Epoch: 13 [30720/60000 (51%)]	Loss: 0.275038
    Train Epoch: 13 [31360/60000 (52%)]	Loss: 0.376274
    Train Epoch: 13 [32000/60000 (53%)]	Loss: 0.517350
    Train Epoch: 13 [32640/60000 (54%)]	Loss: 0.386348
    Train Epoch: 13 [33280/60000 (55%)]	Loss: 0.315578
    Train Epoch: 13 [33920/60000 (57%)]	Loss: 0.385711
    Train Epoch: 13 [34560/60000 (58%)]	Loss: 0.308083
    Train Epoch: 13 [35200/60000 (59%)]	Loss: 0.412020
    Train Epoch: 13 [35840/60000 (60%)]	Loss: 0.630597
    Train Epoch: 13 [36480/60000 (61%)]	Loss: 0.530440
    Train Epoch: 13 [37120/60000 (62%)]	Loss: 0.324687
    Train Epoch: 13 [37760/60000 (63%)]	Loss: 0.334050
    Train Epoch: 13 [38400/60000 (64%)]	Loss: 0.539303
    Train Epoch: 13 [39040/60000 (65%)]	Loss: 0.168277
    Train Epoch: 13 [39680/60000 (66%)]	Loss: 0.218963
    Train Epoch: 13 [40320/60000 (67%)]	Loss: 0.526194
    Train Epoch: 13 [40960/60000 (68%)]	Loss: 0.554866
    Train Epoch: 13 [41600/60000 (69%)]	Loss: 0.519487
    Train Epoch: 13 [42240/60000 (70%)]	Loss: 0.659214
    Train Epoch: 13 [42880/60000 (71%)]	Loss: 0.347684
    Train Epoch: 13 [43520/60000 (72%)]	Loss: 0.218574
    Train Epoch: 13 [44160/60000 (74%)]	Loss: 0.498827
    Train Epoch: 13 [44800/60000 (75%)]	Loss: 0.428912
    Train Epoch: 13 [45440/60000 (76%)]	Loss: 0.554430
    Train Epoch: 13 [46080/60000 (77%)]	Loss: 0.334990
    Train Epoch: 13 [46720/60000 (78%)]	Loss: 0.312058
    Train Epoch: 13 [47360/60000 (79%)]	Loss: 0.393213
    Train Epoch: 13 [48000/60000 (80%)]	Loss: 0.328563
    Train Epoch: 13 [48640/60000 (81%)]	Loss: 0.441794
    Train Epoch: 13 [49280/60000 (82%)]	Loss: 0.487448
    Train Epoch: 13 [49920/60000 (83%)]	Loss: 0.393158
    Train Epoch: 13 [50560/60000 (84%)]	Loss: 0.413585
    Train Epoch: 13 [51200/60000 (85%)]	Loss: 0.331015
    Train Epoch: 13 [51840/60000 (86%)]	Loss: 0.293183
    Train Epoch: 13 [52480/60000 (87%)]	Loss: 0.448310
    Train Epoch: 13 [53120/60000 (88%)]	Loss: 0.275573
    Train Epoch: 13 [53760/60000 (90%)]	Loss: 0.361041
    Train Epoch: 13 [54400/60000 (91%)]	Loss: 0.270119
    Train Epoch: 13 [55040/60000 (92%)]	Loss: 0.339491
    Train Epoch: 13 [55680/60000 (93%)]	Loss: 0.460334
    Train Epoch: 13 [56320/60000 (94%)]	Loss: 0.355197
    Train Epoch: 13 [56960/60000 (95%)]	Loss: 0.324064
    Train Epoch: 13 [57600/60000 (96%)]	Loss: 0.461057
    Train Epoch: 13 [58240/60000 (97%)]	Loss: 0.520947
    Train Epoch: 13 [58880/60000 (98%)]	Loss: 0.555590
    Train Epoch: 13 [59520/60000 (99%)]	Loss: 0.347576
    
    Test set: Average loss: 0.2075, Accuracy: 9385/10000 (94%)
    
    Train Epoch: 14 [0/60000 (0%)]	Loss: 0.319042
    Train Epoch: 14 [640/60000 (1%)]	Loss: 0.286377
    Train Epoch: 14 [1280/60000 (2%)]	Loss: 0.475702
    Train Epoch: 14 [1920/60000 (3%)]	Loss: 0.460729
    Train Epoch: 14 [2560/60000 (4%)]	Loss: 0.227350
    Train Epoch: 14 [3200/60000 (5%)]	Loss: 0.430530
    Train Epoch: 14 [3840/60000 (6%)]	Loss: 0.370811
    Train Epoch: 14 [4480/60000 (7%)]	Loss: 0.292918
    Train Epoch: 14 [5120/60000 (9%)]	Loss: 0.462069
    Train Epoch: 14 [5760/60000 (10%)]	Loss: 0.240440
    Train Epoch: 14 [6400/60000 (11%)]	Loss: 0.330162
    Train Epoch: 14 [7040/60000 (12%)]	Loss: 0.385992
    Train Epoch: 14 [7680/60000 (13%)]	Loss: 0.260772
    Train Epoch: 14 [8320/60000 (14%)]	Loss: 0.431668
    Train Epoch: 14 [8960/60000 (15%)]	Loss: 0.391845
    Train Epoch: 14 [9600/60000 (16%)]	Loss: 0.607404
    Train Epoch: 14 [10240/60000 (17%)]	Loss: 0.517053
    Train Epoch: 14 [10880/60000 (18%)]	Loss: 0.460434
    Train Epoch: 14 [11520/60000 (19%)]	Loss: 0.294837
    Train Epoch: 14 [12160/60000 (20%)]	Loss: 0.376117
    Train Epoch: 14 [12800/60000 (21%)]	Loss: 0.302840
    Train Epoch: 14 [13440/60000 (22%)]	Loss: 0.423695
    Train Epoch: 14 [14080/60000 (23%)]	Loss: 0.396550
    Train Epoch: 14 [14720/60000 (25%)]	Loss: 0.315363
    Train Epoch: 14 [15360/60000 (26%)]	Loss: 0.452954
    Train Epoch: 14 [16000/60000 (27%)]	Loss: 0.492529
    Train Epoch: 14 [16640/60000 (28%)]	Loss: 0.209144
    Train Epoch: 14 [17280/60000 (29%)]	Loss: 0.361104
    Train Epoch: 14 [17920/60000 (30%)]	Loss: 0.337909
    Train Epoch: 14 [18560/60000 (31%)]	Loss: 0.235293
    Train Epoch: 14 [19200/60000 (32%)]	Loss: 0.378781
    Train Epoch: 14 [19840/60000 (33%)]	Loss: 0.698394
    Train Epoch: 14 [20480/60000 (34%)]	Loss: 0.654676
    Train Epoch: 14 [21120/60000 (35%)]	Loss: 0.261703
    Train Epoch: 14 [21760/60000 (36%)]	Loss: 0.491567
    Train Epoch: 14 [22400/60000 (37%)]	Loss: 0.460270
    Train Epoch: 14 [23040/60000 (38%)]	Loss: 0.663426
    Train Epoch: 14 [23680/60000 (39%)]	Loss: 0.488279
    Train Epoch: 14 [24320/60000 (41%)]	Loss: 0.412345
    Train Epoch: 14 [24960/60000 (42%)]	Loss: 0.330990
    Train Epoch: 14 [25600/60000 (43%)]	Loss: 0.319392
    Train Epoch: 14 [26240/60000 (44%)]	Loss: 0.364210
    Train Epoch: 14 [26880/60000 (45%)]	Loss: 0.279273
    Train Epoch: 14 [27520/60000 (46%)]	Loss: 0.176225
    Train Epoch: 14 [28160/60000 (47%)]	Loss: 0.297679
    Train Epoch: 14 [28800/60000 (48%)]	Loss: 0.378201
    Train Epoch: 14 [29440/60000 (49%)]	Loss: 0.232203
    Train Epoch: 14 [30080/60000 (50%)]	Loss: 0.525251
    Train Epoch: 14 [30720/60000 (51%)]	Loss: 0.368206
    Train Epoch: 14 [31360/60000 (52%)]	Loss: 0.304667
    Train Epoch: 14 [32000/60000 (53%)]	Loss: 0.358427
    Train Epoch: 14 [32640/60000 (54%)]	Loss: 0.427945
    Train Epoch: 14 [33280/60000 (55%)]	Loss: 0.488428
    Train Epoch: 14 [33920/60000 (57%)]	Loss: 0.526154
    Train Epoch: 14 [34560/60000 (58%)]	Loss: 0.725787
    Train Epoch: 14 [35200/60000 (59%)]	Loss: 0.599196
    Train Epoch: 14 [35840/60000 (60%)]	Loss: 0.327683
    Train Epoch: 14 [36480/60000 (61%)]	Loss: 0.611174
    Train Epoch: 14 [37120/60000 (62%)]	Loss: 0.429955
    Train Epoch: 14 [37760/60000 (63%)]	Loss: 0.384994
    Train Epoch: 14 [38400/60000 (64%)]	Loss: 0.302765
    Train Epoch: 14 [39040/60000 (65%)]	Loss: 0.637129
    Train Epoch: 14 [39680/60000 (66%)]	Loss: 0.300277
    Train Epoch: 14 [40320/60000 (67%)]	Loss: 0.605257
    Train Epoch: 14 [40960/60000 (68%)]	Loss: 0.563442
    Train Epoch: 14 [41600/60000 (69%)]	Loss: 0.315805
    Train Epoch: 14 [42240/60000 (70%)]	Loss: 0.498133
    Train Epoch: 14 [42880/60000 (71%)]	Loss: 0.304480
    Train Epoch: 14 [43520/60000 (72%)]	Loss: 0.358127
    Train Epoch: 14 [44160/60000 (74%)]	Loss: 0.354776
    Train Epoch: 14 [44800/60000 (75%)]	Loss: 0.349251
    Train Epoch: 14 [45440/60000 (76%)]	Loss: 0.363538
    Train Epoch: 14 [46080/60000 (77%)]	Loss: 0.397053
    Train Epoch: 14 [46720/60000 (78%)]	Loss: 0.569868
    Train Epoch: 14 [47360/60000 (79%)]	Loss: 0.387928
    Train Epoch: 14 [48000/60000 (80%)]	Loss: 0.348416
    Train Epoch: 14 [48640/60000 (81%)]	Loss: 0.377062
    Train Epoch: 14 [49280/60000 (82%)]	Loss: 0.260186
    Train Epoch: 14 [49920/60000 (83%)]	Loss: 0.297211
    Train Epoch: 14 [50560/60000 (84%)]	Loss: 0.702463
    Train Epoch: 14 [51200/60000 (85%)]	Loss: 0.302333
    Train Epoch: 14 [51840/60000 (86%)]	Loss: 0.526482
    Train Epoch: 14 [52480/60000 (87%)]	Loss: 0.400840
    Train Epoch: 14 [53120/60000 (88%)]	Loss: 0.501183
    Train Epoch: 14 [53760/60000 (90%)]	Loss: 0.302831
    Train Epoch: 14 [54400/60000 (91%)]	Loss: 0.351778
    Train Epoch: 14 [55040/60000 (92%)]	Loss: 0.406741
    Train Epoch: 14 [55680/60000 (93%)]	Loss: 0.455119
    Train Epoch: 14 [56320/60000 (94%)]	Loss: 0.324182
    Train Epoch: 14 [56960/60000 (95%)]	Loss: 0.380480
    Train Epoch: 14 [57600/60000 (96%)]	Loss: 0.729591
    Train Epoch: 14 [58240/60000 (97%)]	Loss: 0.435105
    Train Epoch: 14 [58880/60000 (98%)]	Loss: 0.378653
    Train Epoch: 14 [59520/60000 (99%)]	Loss: 0.280005
    
    Test set: Average loss: 0.2066, Accuracy: 9386/10000 (94%)
    


## Running on bacalhau

### Uploading the dataset to IPFS

Since Container running on bacalhau has no network we need to manually upload the dateset to IPFS

we can download the dataset using pytorch datasets in this case we need to download the MNIST dataset we create a folder data where we will download the dataset


```bash
%%bash
mkdir ./data
```


```python
from torchvision import datasets
from torchvision.transforms import ToTensor

training_data = datasets.MNIST(
    root="./data",
    train=True,
    download=True,
    transform=ToTensor()
)

test_data = datasets.MNIST(
    root="./data",
    train=False,
    download=True,
    transform=ToTensor()
)
```

    Downloading http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz
    Downloading http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz to ./data/MNIST/raw/train-images-idx3-ubyte.gz



      0%|          | 0/9912422 [00:00<?, ?it/s]


    Extracting ./data/MNIST/raw/train-images-idx3-ubyte.gz to ./data/MNIST/raw
    
    Downloading http://yann.lecun.com/exdb/mnist/train-labels-idx1-ubyte.gz
    Downloading http://yann.lecun.com/exdb/mnist/train-labels-idx1-ubyte.gz to ./data/MNIST/raw/train-labels-idx1-ubyte.gz



      0%|          | 0/28881 [00:00<?, ?it/s]


    Extracting ./data/MNIST/raw/train-labels-idx1-ubyte.gz to ./data/MNIST/raw
    
    Downloading http://yann.lecun.com/exdb/mnist/t10k-images-idx3-ubyte.gz
    Downloading http://yann.lecun.com/exdb/mnist/t10k-images-idx3-ubyte.gz to ./data/MNIST/raw/t10k-images-idx3-ubyte.gz



      0%|          | 0/1648877 [00:00<?, ?it/s]


    Extracting ./data/MNIST/raw/t10k-images-idx3-ubyte.gz to ./data/MNIST/raw
    
    Downloading http://yann.lecun.com/exdb/mnist/t10k-labels-idx1-ubyte.gz
    Downloading http://yann.lecun.com/exdb/mnist/t10k-labels-idx1-ubyte.gz to ./data/MNIST/raw/t10k-labels-idx1-ubyte.gz



      0%|          | 0/4542 [00:00<?, ?it/s]


    Extracting ./data/MNIST/raw/t10k-labels-idx1-ubyte.gz to ./data/MNIST/raw
    


### Uploading the dataset to IPFS

Using the IPFS cli
```
ipfs add -r data
```



Since the data Uploaded To IPFS using IPFS CLI isn’t pinned or will be garbage collected

The Data needs to be Pinned, Pinning is the mechanism that allows you to tell IPFS to always keep a given object somewhere, the default being your local node, though this can be different if you use a third-party remote pinning service.

There a different pinning services available you can you any one of them


## [Pinata](https://app.pinata.cloud/)

Click on the upload folder button

![](https://i.imgur.com/crnkrwy.png)

After the Upload has finished copy the CID

### [NFT.Storage](https://nft.storage/) (Recommneded Option)

[Upload files and directories with NFTUp](https://nft.storage/docs/how-to/nftup/) 

To upload your dataset using NFTup just drag and drop your directory it will upload it to IPFS

![](https://i.imgur.com/03NEonV.png)


Copy the CID in this case it is QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw
(If you used pinata) or bafybeif5m2md7bo2iua3kfate72kh54jgwr2spgvdtn33zdeqffh3d6qce
(if you used nft.storage)

You can view you uploaded dataset by clicking on the Gateway URL

[https://gateway.pinata.cloud/ipfs/QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw/?filename=data](https://gateway.pinata.cloud/ipfs/QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw/?filename=data)


```python
!curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.13 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.13/bacalhau_v0.3.13_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.13/bacalhau_v0.3.13_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.3.13
    Server Version: v0.3.13



```bash
%%bash --out job_id
bacalhau docker run \
--gpu 1 \
--timeout 3600 \
--wait-timeout-secs 3600 \
--wait \
--id-only \
pytorch/pytorch \
-w /outputs \
 -v QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw:/data \
-u https://raw.githubusercontent.com/pytorch/examples/main/mnist_rnn/main.py \
-- python ../inputs/main.py --save-model
```

Sturucture of the command

Request 1 GPU to train the model --gpu 1

Using the official pytorch docker Image pytorch/pytorch

Mounting the uploaded dataset to path /data -v QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw:/data

Mounting our training script we will use the [Training script](https://github.com/pytorch/examples/blob/main/mnist_rnn/main.py) from the pytorch examples and use the raw link of the script
-u https://raw.githubusercontent.com/pytorch/examples/main/mnist_rnn/main.py

Its the folder where we will to save the model as it will automatically gets uploaded to IPFS as outputs so we choose /outputs as our working directory
-w /outputs

Running the script
python ../inputs/main.py --save-model

since the URL script gets mounted to the /inputs folder in the container
we will execute that script but since our working directory is /outputs we provide the relave path to python to execute the script


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 14:43:37 [0m[97;40m 1658bb6b [0m[97;40m Docker pytorch/pytor... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmTZKuZJX3Zj9v... [0m


Where it says "Completed", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:



```bash
%%bash
bacalhau describe ${JOB_ID}
```


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job '1658bb6b-21d1-4d1a-a278-b0984c967e14'...
    Results for job '1658bb6b-21d1-4d1a-a278-b0984c967e14' have been written to...
    results


    2022/11/21 14:46:56 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.



```bash
%%bash
ls results/
```

    combined_results
    per_shard
    raw



```bash
%%bash
cat results/combined_results/stdout
```

    Train Epoch: 1 [0/60000 (0%)]	Loss: 2.257103
    Train Epoch: 1 [640/60000 (1%)]	Loss: 2.343541
    Train Epoch: 1 [1280/60000 (2%)]	Loss: 2.286971
    Train Epoch: 1 [1920/60000 (3%)]	Loss: 2.278690
    Train Epoch: 1 [2560/60000 (4%)]	Loss: 2.325279
    Train Epoch: 1 [3200/60000 (5%)]	Loss: 2.156002
    Train Epoch: 1 [3840/60000 (6%)]	Loss: 2.213600
    Train Epoch: 1 [4480/60000 (7%)]	Loss: 2.205997
    Train Epoch: 1 [5120/60000 (9%)]	Loss: 2.104978
    Train Epoch: 1 [5760/60000 (10%)]	Loss: 2.133132
    Train Epoch: 1 [6400/60000 (11%)]	Loss: 2.141112
    Train Epoch: 1 [7040/60000 (12%)]	Loss: 2.029041
    Train Epoch: 1 [7680/60000 (13%)]	Loss: 2.038754
    Train Epoch: 1 [8320/60000 (14%)]	Loss: 1.982695
    Train Epoch: 1 [8960/60000 (15%)]	Loss: 2.027745
    Train Epoch: 1 [9600/60000 (16%)]	Loss: 1.933618
    Train Epoch: 1 [10240/60000 (17%)]	Loss: 2.001938
    Train Epoch: 1 [10880/60000 (18%)]	Loss: 1.990632
    Train Epoch: 1 [11520/60000 (19%)]	Loss: 1.903336
    Train Epoch: 1 [12160/60000 (20%)]	Loss: 1.927148
    Train Epoch: 1 [12800/60000 (21%)]	Loss: 1.932347
    Train Epoch: 1 [13440/60000 (22%)]	Loss: 1.768175
    Train Epoch: 1 [14080/60000 (23%)]	Loss: 1.793583
    Train Epoch: 1 [14720/60000 (25%)]	Loss: 1.698625
    Train Epoch: 1 [15360/60000 (26%)]	Loss: 1.919402
    Train Epoch: 1 [16000/60000 (27%)]	Loss: 1.819005
    Train Epoch: 1 [16640/60000 (28%)]	Loss: 1.798551
    Train Epoch: 1 [17280/60000 (29%)]	Loss: 1.752450
    Train Epoch: 1 [17920/60000 (30%)]	Loss: 1.580650
    Train Epoch: 1 [18560/60000 (31%)]	Loss: 1.669491
    Train Epoch: 1 [19200/60000 (32%)]	Loss: 1.666683
    Train Epoch: 1 [19840/60000 (33%)]	Loss: 1.746461
    Train Epoch: 1 [20480/60000 (34%)]	Loss: 1.750646
    Train Epoch: 1 [21120/60000 (35%)]	Loss: 1.704663
    Train Epoch: 1 [21760/60000 (36%)]	Loss: 1.545694
    Train Epoch: 1 [22400/60000 (37%)]	Loss: 1.800772
    Train Epoch: 1 [23040/60000 (38%)]	Loss: 1.807309
    Train Epoch: 1 [23680/60000 (39%)]	Loss: 1.531072
    Train Epoch: 1 [24320/60000 (41%)]	Loss: 1.644449
    Train Epoch: 1 [24960/60000 (42%)]	Loss: 1.440658
    Train Epoch: 1 [25600/60000 (43%)]	Loss: 1.572379
    Train Epoch: 1 [26240/60000 (44%)]	Loss: 1.542955
    Train Epoch: 1 [26880/60000 (45%)]	Loss: 1.636800
    Train Epoch: 1 [27520/60000 (46%)]	Loss: 1.732645
    Train Epoch: 1 [28160/60000 (47%)]	Loss: 1.556232
    Train Epoch: 1 [28800/60000 (48%)]	Loss: 1.797165
    Train Epoch: 1 [29440/60000 (49%)]	Loss: 1.550113
    Train Epoch: 1 [30080/60000 (50%)]	Loss: 1.513264
    Train Epoch: 1 [30720/60000 (51%)]	Loss: 1.349926
    Train Epoch: 1 [31360/60000 (52%)]	Loss: 1.168647
    Train Epoch: 1 [32000/60000 (53%)]	Loss: 1.371591
    Train Epoch: 1 [32640/60000 (54%)]	Loss: 1.360642
    Train Epoch: 1 [33280/60000 (55%)]	Loss: 1.319583
    Train Epoch: 1 [33920/60000 (57%)]	Loss: 1.470899
    Train Epoch: 1 [34560/60000 (58%)]	Loss: 1.229612
    Train Epoch: 1 [35200/60000 (59%)]	Loss: 1.355430
    Train Epoch: 1 [35840/60000 (60%)]	Loss: 1.162910
    Train Epoch: 1 [36480/60000 (61%)]	Loss: 1.264161
    Train Epoch: 1 [37120/60000 (62%)]	Loss: 1.304694
    Train Epoch: 1 [37760/60000 (63%)]	Loss: 1.245098
    Train Epoch: 1 [38400/60000 (64%)]	Loss: 1.276992
    Train Epoch: 1 [39040/60000 (65%)]	Loss: 1.224096
    Train Epoch: 1 [39680/60000 (66%)]	Loss: 1.017790
    Train Epoch: 1 [40320/60000 (67%)]	Loss: 1.265200
    Train Epoch: 1 [40960/60000 (68%)]	Loss: 1.095893
    Train Epoch: 1 [41600/60000 (69%)]	Loss: 1.253011
    Train Epoch: 1 [42240/60000 (70%)]	Loss: 1.309954
    Train Epoch: 1 [42880/60000 (71%)]	Loss: 1.072964
    Train Epoch: 1 [43520/60000 (72%)]	Loss: 1.278133
    Train Epoch: 1 [44160/60000 (74%)]	Loss: 1.042409
    Train Epoch: 1 [44800/60000 (75%)]	Loss: 1.204304
    Train Epoch: 1 [45440/60000 (76%)]	Loss: 1.224481
    Train Epoch: 1 [46080/60000 (77%)]	Loss: 1.168465
    Train Epoch: 1 [46720/60000 (78%)]	Loss: 1.225616
    Train Epoch: 1 [47360/60000 (79%)]	Loss: 1.107115
    Train Epoch: 1 [48000/60000 (80%)]	Loss: 0.964020
    Train Epoch: 1 [48640/60000 (81%)]	Loss: 1.150630
    Train Epoch: 1 [49280/60000 (82%)]	Loss: 1.298064
    Train Epoch: 1 [49920/60000 (83%)]	Loss: 1.385768
    Train Epoch: 1 [50560/60000 (84%)]	Loss: 1.130490
    Train Epoch: 1 [51200/60000 (85%)]	Loss: 0.967750
    Train Epoch: 1 [51840/60000 (86%)]	Loss: 1.239161
    Train Epoch: 1 [52480/60000 (87%)]	Loss: 0.985015
    Train Epoch: 1 [53120/60000 (88%)]	Loss: 1.048505
    Train Epoch: 1 [53760/60000 (90%)]	Loss: 0.928014
    Train Epoch: 1 [54400/60000 (91%)]	Loss: 1.156546
    Train Epoch: 1 [55040/60000 (92%)]	Loss: 1.117476
    Train Epoch: 1 [55680/60000 (93%)]	Loss: 1.082589
    Train Epoch: 1 [56320/60000 (94%)]	Loss: 1.037969
    Train Epoch: 1 [56960/60000 (95%)]	Loss: 0.901225
    Train Epoch: 1 [57600/60000 (96%)]	Loss: 0.939105
    Train Epoch: 1 [58240/60000 (97%)]	Loss: 0.977517
    Train Epoch: 1 [58880/60000 (98%)]	Loss: 1.061300
    Train Epoch: 1 [59520/60000 (99%)]	Loss: 1.161198
    
    Test set: Average loss: 0.7476, Accuracy: 7615/10000 (76%)
    
    Train Epoch: 2 [0/60000 (0%)]	Loss: 1.074720
    Train Epoch: 2 [640/60000 (1%)]	Loss: 1.031572
    Train Epoch: 2 [1280/60000 (2%)]	Loss: 0.896288
    Train Epoch: 2 [1920/60000 (3%)]	Loss: 1.111214
    Train Epoch: 2 [2560/60000 (4%)]	Loss: 1.075807
    Train Epoch: 2 [3200/60000 (5%)]	Loss: 0.896091
    Train Epoch: 2 [3840/60000 (6%)]	Loss: 0.898205
    Train Epoch: 2 [4480/60000 (7%)]	Loss: 0.909036
    Train Epoch: 2 [5120/60000 (9%)]	Loss: 0.871764
    Train Epoch: 2 [5760/60000 (10%)]	Loss: 0.809469
    Train Epoch: 2 [6400/60000 (11%)]	Loss: 1.018834
    Train Epoch: 2 [7040/60000 (12%)]	Loss: 0.893395
    Train Epoch: 2 [7680/60000 (13%)]	Loss: 0.832215
    Train Epoch: 2 [8320/60000 (14%)]	Loss: 0.942631
    Train Epoch: 2 [8960/60000 (15%)]	Loss: 0.899457
    Train Epoch: 2 [9600/60000 (16%)]	Loss: 1.078218
    Train Epoch: 2 [10240/60000 (17%)]	Loss: 0.860738
    Train Epoch: 2 [10880/60000 (18%)]	Loss: 0.742847
    Train Epoch: 2 [11520/60000 (19%)]	Loss: 1.037842
    Train Epoch: 2 [12160/60000 (20%)]	Loss: 1.066162
    Train Epoch: 2 [12800/60000 (21%)]	Loss: 0.885088
    Train Epoch: 2 [13440/60000 (22%)]	Loss: 0.996853
    Train Epoch: 2 [14080/60000 (23%)]	Loss: 0.822172
    Train Epoch: 2 [14720/60000 (25%)]	Loss: 0.993543
    Train Epoch: 2 [15360/60000 (26%)]	Loss: 0.810572
    Train Epoch: 2 [16000/60000 (27%)]	Loss: 1.058692
    Train Epoch: 2 [16640/60000 (28%)]	Loss: 0.866647
    Train Epoch: 2 [17280/60000 (29%)]	Loss: 0.772441
    Train Epoch: 2 [17920/60000 (30%)]	Loss: 0.720768
    Train Epoch: 2 [18560/60000 (31%)]	Loss: 0.866728
    Train Epoch: 2 [19200/60000 (32%)]	Loss: 0.705710
    Train Epoch: 2 [19840/60000 (33%)]	Loss: 0.890331
    Train Epoch: 2 [20480/60000 (34%)]	Loss: 0.834183
    Train Epoch: 2 [21120/60000 (35%)]	Loss: 0.774839
    Train Epoch: 2 [21760/60000 (36%)]	Loss: 0.879249
    Train Epoch: 2 [22400/60000 (37%)]	Loss: 0.861507
    Train Epoch: 2 [23040/60000 (38%)]	Loss: 0.725027
    Train Epoch: 2 [23680/60000 (39%)]	Loss: 0.870410
    Train Epoch: 2 [24320/60000 (41%)]	Loss: 0.694554
    Train Epoch: 2 [24960/60000 (42%)]	Loss: 0.808239
    Train Epoch: 2 [25600/60000 (43%)]	Loss: 0.807047
    Train Epoch: 2 [26240/60000 (44%)]	Loss: 0.861262
    Train Epoch: 2 [26880/60000 (45%)]	Loss: 0.760611
    Train Epoch: 2 [27520/60000 (46%)]	Loss: 0.723064
    Train Epoch: 2 [28160/60000 (47%)]	Loss: 0.645913
    Train Epoch: 2 [28800/60000 (48%)]	Loss: 0.794883
    Train Epoch: 2 [29440/60000 (49%)]	Loss: 1.018256
    Train Epoch: 2 [30080/60000 (50%)]	Loss: 0.897736
    Train Epoch: 2 [30720/60000 (51%)]	Loss: 1.036487
    Train Epoch: 2 [31360/60000 (52%)]	Loss: 0.957585
    Train Epoch: 2 [32000/60000 (53%)]	Loss: 0.648525
    Train Epoch: 2 [32640/60000 (54%)]	Loss: 0.908357
    Train Epoch: 2 [33280/60000 (55%)]	Loss: 0.844382
    Train Epoch: 2 [33920/60000 (57%)]	Loss: 0.492543
    Train Epoch: 2 [34560/60000 (58%)]	Loss: 0.767534
    Train Epoch: 2 [35200/60000 (59%)]	Loss: 0.583981
    Train Epoch: 2 [35840/60000 (60%)]	Loss: 0.670485
    Train Epoch: 2 [36480/60000 (61%)]	Loss: 0.812931
    Train Epoch: 2 [37120/60000 (62%)]	Loss: 0.675361
    Train Epoch: 2 [37760/60000 (63%)]	Loss: 0.719999
    Train Epoch: 2 [38400/60000 (64%)]	Loss: 0.733327
    Train Epoch: 2 [39040/60000 (65%)]	Loss: 0.595985
    Train Epoch: 2 [39680/60000 (66%)]	Loss: 0.761033
    Train Epoch: 2 [40320/60000 (67%)]	Loss: 0.547535
    Train Epoch: 2 [40960/60000 (68%)]	Loss: 0.713410
    Train Epoch: 2 [41600/60000 (69%)]	Loss: 0.774444
    Train Epoch: 2 [42240/60000 (70%)]	Loss: 0.536494
    Train Epoch: 2 [42880/60000 (71%)]	Loss: 0.678178
    Train Epoch: 2 [43520/60000 (72%)]	Loss: 0.612846
    Train Epoch: 2 [44160/60000 (74%)]	Loss: 0.596894
    Train Epoch: 2 [44800/60000 (75%)]	Loss: 0.629905
    Train Epoch: 2 [45440/60000 (76%)]	Loss: 0.812533
    Train Epoch: 2 [46080/60000 (77%)]	Loss: 0.749563
    Train Epoch: 2 [46720/60000 (78%)]	Loss: 0.686619
    Train Epoch: 2 [47360/60000 (79%)]	Loss: 0.817192
    Train Epoch: 2 [48000/60000 (80%)]	Loss: 0.521638
    Train Epoch: 2 [48640/60000 (81%)]	Loss: 0.948533
    Train Epoch: 2 [49280/60000 (82%)]	Loss: 0.807676
    Train Epoch: 2 [49920/60000 (83%)]	Loss: 0.609730
    Train Epoch: 2 [50560/60000 (84%)]	Loss: 0.624522
    Train Epoch: 2 [51200/60000 (85%)]	Loss: 0.688772
    Train Epoch: 2 [51840/60000 (86%)]	Loss: 0.576914
    Train Epoch: 2 [52480/60000 (87%)]	Loss: 0.583184
    Train Epoch: 2 [53120/60000 (88%)]	Loss: 0.739166
    Train Epoch: 2 [53760/60000 (90%)]	Loss: 0.768429
    Train Epoch: 2 [54400/60000 (91%)]	Loss: 0.767365
    Train Epoch: 2 [55040/60000 (92%)]	Loss: 0.739564
    Train Epoch: 2 [55680/60000 (93%)]	Loss: 0.969297
    Train Epoch: 2 [56320/60000 (94%)]	Loss: 0.545870
    Train Epoch: 2 [56960/60000 (95%)]	Loss: 0.490728
    Train Epoch: 2 [57600/60000 (96%)]	Loss: 0.738210
    Train Epoch: 2 [58240/60000 (97%)]	Loss: 0.649950
    Train Epoch: 2 [58880/60000 (98%)]	Loss: 0.534231
    Train Epoch: 2 [59520/60000 (99%)]	Loss: 0.701677
    
    Test set: Average loss: 0.4355, Accuracy: 8636/10000 (86%)
    
    Train Epoch: 3 [0/60000 (0%)]	Loss: 0.436861
    Train Epoch: 3 [640/60000 (1%)]	Loss: 0.613573
    Train Epoch: 3 [1280/60000 (2%)]	Loss: 0.751559
    Train Epoch: 3 [1920/60000 (3%)]	Loss: 0.518953
    Train Epoch: 3 [2560/60000 (4%)]	Loss: 0.706350
    Train Epoch: 3 [3200/60000 (5%)]	Loss: 0.463392
    Train Epoch: 3 [3840/60000 (6%)]	Loss: 0.637765
    Train Epoch: 3 [4480/60000 (7%)]	Loss: 0.707880
    Train Epoch: 3 [5120/60000 (9%)]	Loss: 0.705076
    Train Epoch: 3 [5760/60000 (10%)]	Loss: 0.473644
    Train Epoch: 3 [6400/60000 (11%)]	Loss: 0.566551
    Train Epoch: 3 [7040/60000 (12%)]	Loss: 0.554120
    Train Epoch: 3 [7680/60000 (13%)]	Loss: 0.735059
    Train Epoch: 3 [8320/60000 (14%)]	Loss: 0.492775
    Train Epoch: 3 [8960/60000 (15%)]	Loss: 0.705045
    Train Epoch: 3 [9600/60000 (16%)]	Loss: 0.723935
    Train Epoch: 3 [10240/60000 (17%)]	Loss: 0.657871
    Train Epoch: 3 [10880/60000 (18%)]	Loss: 0.546103
    Train Epoch: 3 [11520/60000 (19%)]	Loss: 0.576001
    Train Epoch: 3 [12160/60000 (20%)]	Loss: 0.762758
    Train Epoch: 3 [12800/60000 (21%)]	Loss: 0.672853
    Train Epoch: 3 [13440/60000 (22%)]	Loss: 0.690244
    Train Epoch: 3 [14080/60000 (23%)]	Loss: 0.491185
    Train Epoch: 3 [14720/60000 (25%)]	Loss: 0.819045
    Train Epoch: 3 [15360/60000 (26%)]	Loss: 0.633367
    Train Epoch: 3 [16000/60000 (27%)]	Loss: 0.631507
    Train Epoch: 3 [16640/60000 (28%)]	Loss: 0.742323
    Train Epoch: 3 [17280/60000 (29%)]	Loss: 0.769272
    Train Epoch: 3 [17920/60000 (30%)]	Loss: 0.547987
    Train Epoch: 3 [18560/60000 (31%)]	Loss: 0.726344
    Train Epoch: 3 [19200/60000 (32%)]	Loss: 0.500911
    Train Epoch: 3 [19840/60000 (33%)]	Loss: 0.609957
    Train Epoch: 3 [20480/60000 (34%)]	Loss: 0.567650
    Train Epoch: 3 [21120/60000 (35%)]	Loss: 0.592656
    Train Epoch: 3 [21760/60000 (36%)]	Loss: 0.659012
    Train Epoch: 3 [22400/60000 (37%)]	Loss: 0.792519
    Train Epoch: 3 [23040/60000 (38%)]	Loss: 0.649515
    Train Epoch: 3 [23680/60000 (39%)]	Loss: 0.535163
    Train Epoch: 3 [24320/60000 (41%)]	Loss: 0.510494
    Train Epoch: 3 [24960/60000 (42%)]	Loss: 0.753703
    Train Epoch: 3 [25600/60000 (43%)]	Loss: 0.588570
    Train Epoch: 3 [26240/60000 (44%)]	Loss: 0.524773
    Train Epoch: 3 [26880/60000 (45%)]	Loss: 0.654643
    Train Epoch: 3 [27520/60000 (46%)]	Loss: 0.464091
    Train Epoch: 3 [28160/60000 (47%)]	Loss: 0.517499
    Train Epoch: 3 [28800/60000 (48%)]	Loss: 0.743199
    Train Epoch: 3 [29440/60000 (49%)]	Loss: 0.712906
    Train Epoch: 3 [30080/60000 (50%)]	Loss: 0.898138
    Train Epoch: 3 [30720/60000 (51%)]	Loss: 0.471215
    Train Epoch: 3 [31360/60000 (52%)]	Loss: 0.586351
    Train Epoch: 3 [32000/60000 (53%)]	Loss: 0.619581
    Train Epoch: 3 [32640/60000 (54%)]	Loss: 0.431174
    Train Epoch: 3 [33280/60000 (55%)]	Loss: 0.805528
    Train Epoch: 3 [33920/60000 (57%)]	Loss: 0.434236
    Train Epoch: 3 [34560/60000 (58%)]	Loss: 0.833718
    Train Epoch: 3 [35200/60000 (59%)]	Loss: 0.737563
    Train Epoch: 3 [35840/60000 (60%)]	Loss: 0.814904
    Train Epoch: 3 [36480/60000 (61%)]	Loss: 0.658191
    Train Epoch: 3 [37120/60000 (62%)]	Loss: 0.642526
    Train Epoch: 3 [37760/60000 (63%)]	Loss: 0.528398
    Train Epoch: 3 [38400/60000 (64%)]	Loss: 0.401048
    Train Epoch: 3 [39040/60000 (65%)]	Loss: 0.638032
    Train Epoch: 3 [39680/60000 (66%)]	Loss: 0.885019
    Train Epoch: 3 [40320/60000 (67%)]	Loss: 0.639517
    Train Epoch: 3 [40960/60000 (68%)]	Loss: 0.777474
    Train Epoch: 3 [41600/60000 (69%)]	Loss: 0.529243
    Train Epoch: 3 [42240/60000 (70%)]	Loss: 0.383692
    Train Epoch: 3 [42880/60000 (71%)]	Loss: 0.399004
    Train Epoch: 3 [43520/60000 (72%)]	Loss: 0.602192
    Train Epoch: 3 [44160/60000 (74%)]	Loss: 0.728852
    Train Epoch: 3 [44800/60000 (75%)]	Loss: 0.605767
    Train Epoch: 3 [45440/60000 (76%)]	Loss: 1.022341
    Train Epoch: 3 [46080/60000 (77%)]	Loss: 0.670445
    Train Epoch: 3 [46720/60000 (78%)]	Loss: 0.567436
    Train Epoch: 3 [47360/60000 (79%)]	Loss: 0.486619
    Train Epoch: 3 [48000/60000 (80%)]	Loss: 0.636935
    Train Epoch: 3 [48640/60000 (81%)]	Loss: 0.501475
    Train Epoch: 3 [49280/60000 (82%)]	Loss: 0.448360
    Train Epoch: 3 [49920/60000 (83%)]	Loss: 0.548112
    Train Epoch: 3 [50560/60000 (84%)]	Loss: 0.518546
    Train Epoch: 3 [51200/60000 (85%)]	Loss: 0.460728
    Train Epoch: 3 [51840/60000 (86%)]	Loss: 0.566899
    Train Epoch: 3 [52480/60000 (87%)]	Loss: 0.455567
    Train Epoch: 3 [53120/60000 (88%)]	Loss: 0.590804
    Train Epoch: 3 [53760/60000 (90%)]	Loss: 0.655986
    Train Epoch: 3 [54400/60000 (91%)]	Loss: 0.603358
    Train Epoch: 3 [55040/60000 (92%)]	Loss: 0.498250
    Train Epoch: 3 [55680/60000 (93%)]	Loss: 0.582818
    Train Epoch: 3 [56320/60000 (94%)]	Loss: 0.671843
    Train Epoch: 3 [56960/60000 (95%)]	Loss: 0.562645
    Train Epoch: 3 [57600/60000 (96%)]	Loss: 0.710898
    Train Epoch: 3 [58240/60000 (97%)]	Loss: 0.704995
    Train Epoch: 3 [58880/60000 (98%)]	Loss: 0.426514
    Train Epoch: 3 [59520/60000 (99%)]	Loss: 0.586657
    
    Test set: Average loss: 0.3266, Accuracy: 9035/10000 (90%)
    
    Train Epoch: 4 [0/60000 (0%)]	Loss: 0.555241
    Train Epoch: 4 [640/60000 (1%)]	Loss: 0.414488
    Train Epoch: 4 [1280/60000 (2%)]	Loss: 0.423981
    Train Epoch: 4 [1920/60000 (3%)]	Loss: 0.458799
    Train Epoch: 4 [2560/60000 (4%)]	Loss: 0.526234
    Train Epoch: 4 [3200/60000 (5%)]	Loss: 0.502130
    Train Epoch: 4 [3840/60000 (6%)]	Loss: 0.572710
    Train Epoch: 4 [4480/60000 (7%)]	Loss: 0.768068
    Train Epoch: 4 [5120/60000 (9%)]	Loss: 0.552236
    Train Epoch: 4 [5760/60000 (10%)]	Loss: 0.413747
    Train Epoch: 4 [6400/60000 (11%)]	Loss: 0.495317
    Train Epoch: 4 [7040/60000 (12%)]	Loss: 0.513442
    Train Epoch: 4 [7680/60000 (13%)]	Loss: 0.371071
    Train Epoch: 4 [8320/60000 (14%)]	Loss: 0.537922
    Train Epoch: 4 [8960/60000 (15%)]	Loss: 0.550542
    Train Epoch: 4 [9600/60000 (16%)]	Loss: 0.492354
    Train Epoch: 4 [10240/60000 (17%)]	Loss: 0.430003
    Train Epoch: 4 [10880/60000 (18%)]	Loss: 0.676727
    Train Epoch: 4 [11520/60000 (19%)]	Loss: 0.522242
    Train Epoch: 4 [12160/60000 (20%)]	Loss: 0.323046
    Train Epoch: 4 [12800/60000 (21%)]	Loss: 0.413817
    Train Epoch: 4 [13440/60000 (22%)]	Loss: 0.493616
    Train Epoch: 4 [14080/60000 (23%)]	Loss: 0.482043
    Train Epoch: 4 [14720/60000 (25%)]	Loss: 0.598020
    Train Epoch: 4 [15360/60000 (26%)]	Loss: 0.698045
    Train Epoch: 4 [16000/60000 (27%)]	Loss: 0.464925
    Train Epoch: 4 [16640/60000 (28%)]	Loss: 0.598145
    Train Epoch: 4 [17280/60000 (29%)]	Loss: 0.513251
    Train Epoch: 4 [17920/60000 (30%)]	Loss: 0.383759
    Train Epoch: 4 [18560/60000 (31%)]	Loss: 0.451445
    Train Epoch: 4 [19200/60000 (32%)]	Loss: 0.298578
    Train Epoch: 4 [19840/60000 (33%)]	Loss: 0.724677
    Train Epoch: 4 [20480/60000 (34%)]	Loss: 0.648704
    Train Epoch: 4 [21120/60000 (35%)]	Loss: 0.417878
    Train Epoch: 4 [21760/60000 (36%)]	Loss: 0.587597
    Train Epoch: 4 [22400/60000 (37%)]	Loss: 0.650825
    Train Epoch: 4 [23040/60000 (38%)]	Loss: 0.461850
    Train Epoch: 4 [23680/60000 (39%)]	Loss: 0.498996
    Train Epoch: 4 [24320/60000 (41%)]	Loss: 0.272354
    Train Epoch: 4 [24960/60000 (42%)]	Loss: 0.552614
    Train Epoch: 4 [25600/60000 (43%)]	Loss: 0.559007
    Train Epoch: 4 [26240/60000 (44%)]	Loss: 0.514660
    Train Epoch: 4 [26880/60000 (45%)]	Loss: 0.449900
    Train Epoch: 4 [27520/60000 (46%)]	Loss: 0.459001
    Train Epoch: 4 [28160/60000 (47%)]	Loss: 0.510848
    Train Epoch: 4 [28800/60000 (48%)]	Loss: 0.376767
    Train Epoch: 4 [29440/60000 (49%)]	Loss: 0.663157
    Train Epoch: 4 [30080/60000 (50%)]	Loss: 0.380203
    Train Epoch: 4 [30720/60000 (51%)]	Loss: 0.487593
    Train Epoch: 4 [31360/60000 (52%)]	Loss: 0.368222
    Train Epoch: 4 [32000/60000 (53%)]	Loss: 0.531884
    Train Epoch: 4 [32640/60000 (54%)]	Loss: 0.514744
    Train Epoch: 4 [33280/60000 (55%)]	Loss: 0.413709
    Train Epoch: 4 [33920/60000 (57%)]	Loss: 0.466324
    Train Epoch: 4 [34560/60000 (58%)]	Loss: 0.481780
    Train Epoch: 4 [35200/60000 (59%)]	Loss: 0.332192
    Train Epoch: 4 [35840/60000 (60%)]	Loss: 0.535553
    Train Epoch: 4 [36480/60000 (61%)]	Loss: 0.701526
    Train Epoch: 4 [37120/60000 (62%)]	Loss: 0.472824
    Train Epoch: 4 [37760/60000 (63%)]	Loss: 0.506160
    Train Epoch: 4 [38400/60000 (64%)]	Loss: 0.434093
    Train Epoch: 4 [39040/60000 (65%)]	Loss: 0.458589
    Train Epoch: 4 [39680/60000 (66%)]	Loss: 0.571873
    Train Epoch: 4 [40320/60000 (67%)]	Loss: 0.417425
    Train Epoch: 4 [40960/60000 (68%)]	Loss: 0.562600
    Train Epoch: 4 [41600/60000 (69%)]	Loss: 0.595764
    Train Epoch: 4 [42240/60000 (70%)]	Loss: 0.763260
    Train Epoch: 4 [42880/60000 (71%)]	Loss: 0.449961
    Train Epoch: 4 [43520/60000 (72%)]	Loss: 0.504708
    Train Epoch: 4 [44160/60000 (74%)]	Loss: 0.518068
    Train Epoch: 4 [44800/60000 (75%)]	Loss: 0.457749
    Train Epoch: 4 [45440/60000 (76%)]	Loss: 0.556885
    Train Epoch: 4 [46080/60000 (77%)]	Loss: 0.407525
    Train Epoch: 4 [46720/60000 (78%)]	Loss: 0.627191
    Train Epoch: 4 [47360/60000 (79%)]	Loss: 0.640686
    Train Epoch: 4 [48000/60000 (80%)]	Loss: 0.461735
    Train Epoch: 4 [48640/60000 (81%)]	Loss: 0.440985
    Train Epoch: 4 [49280/60000 (82%)]	Loss: 0.617622
    Train Epoch: 4 [49920/60000 (83%)]	Loss: 0.502659
    Train Epoch: 4 [50560/60000 (84%)]	Loss: 0.525112
    Train Epoch: 4 [51200/60000 (85%)]	Loss: 0.530758
    Train Epoch: 4 [51840/60000 (86%)]	Loss: 0.327249
    Train Epoch: 4 [52480/60000 (87%)]	Loss: 0.392865
    Train Epoch: 4 [53120/60000 (88%)]	Loss: 0.716493
    Train Epoch: 4 [53760/60000 (90%)]	Loss: 0.916052
    Train Epoch: 4 [54400/60000 (91%)]	Loss: 0.398535
    Train Epoch: 4 [55040/60000 (92%)]	Loss: 0.514751
    Train Epoch: 4 [55680/60000 (93%)]	Loss: 0.466898
    Train Epoch: 4 [56320/60000 (94%)]	Loss: 0.446998
    Train Epoch: 4 [56960/60000 (95%)]	Loss: 0.575153
    Train Epoch: 4 [57600/60000 (96%)]	Loss: 0.578760
    Train Epoch: 4 [58240/60000 (97%)]	Loss: 0.473565
    Train Epoch: 4 [58880/60000 (98%)]	Loss: 0.520567
    Train Epoch: 4 [59520/60000 (99%)]	Loss: 0.242124
    
    Test set: Average loss: 0.2797, Accuracy: 9146/10000 (91%)
    
    Train Epoch: 5 [0/60000 (0%)]	Loss: 0.509089
    Train Epoch: 5 [640/60000 (1%)]	Loss: 0.581981
    Train Epoch: 5 [1280/60000 (2%)]	Loss: 0.393444
    Train Epoch: 5 [1920/60000 (3%)]	Loss: 0.635975
    Train Epoch: 5 [2560/60000 (4%)]	Loss: 0.359194
    Train Epoch: 5 [3200/60000 (5%)]	Loss: 0.446414
    Train Epoch: 5 [3840/60000 (6%)]	Loss: 0.638959
    Train Epoch: 5 [4480/60000 (7%)]	Loss: 0.456178
    Train Epoch: 5 [5120/60000 (9%)]	Loss: 0.676888
    Train Epoch: 5 [5760/60000 (10%)]	Loss: 0.725724
    Train Epoch: 5 [6400/60000 (11%)]	Loss: 0.758731
    Train Epoch: 5 [7040/60000 (12%)]	Loss: 0.298135
    Train Epoch: 5 [7680/60000 (13%)]	Loss: 0.498484
    Train Epoch: 5 [8320/60000 (14%)]	Loss: 0.781466
    Train Epoch: 5 [8960/60000 (15%)]	Loss: 0.372765
    Train Epoch: 5 [9600/60000 (16%)]	Loss: 0.551780
    Train Epoch: 5 [10240/60000 (17%)]	Loss: 0.671177
    Train Epoch: 5 [10880/60000 (18%)]	Loss: 0.386135
    Train Epoch: 5 [11520/60000 (19%)]	Loss: 0.429770
    Train Epoch: 5 [12160/60000 (20%)]	Loss: 0.351372
    Train Epoch: 5 [12800/60000 (21%)]	Loss: 0.712960
    Train Epoch: 5 [13440/60000 (22%)]	Loss: 0.696321
    Train Epoch: 5 [14080/60000 (23%)]	Loss: 0.242317
    Train Epoch: 5 [14720/60000 (25%)]	Loss: 0.757245
    Train Epoch: 5 [15360/60000 (26%)]	Loss: 0.641723
    Train Epoch: 5 [16000/60000 (27%)]	Loss: 0.303924
    Train Epoch: 5 [16640/60000 (28%)]	Loss: 0.451921
    Train Epoch: 5 [17280/60000 (29%)]	Loss: 0.546511
    Train Epoch: 5 [17920/60000 (30%)]	Loss: 0.449047
    Train Epoch: 5 [18560/60000 (31%)]	Loss: 0.497756
    Train Epoch: 5 [19200/60000 (32%)]	Loss: 0.590394
    Train Epoch: 5 [19840/60000 (33%)]	Loss: 0.591735
    Train Epoch: 5 [20480/60000 (34%)]	Loss: 0.422177
    Train Epoch: 5 [21120/60000 (35%)]	Loss: 0.596936
    Train Epoch: 5 [21760/60000 (36%)]	Loss: 0.533217
    Train Epoch: 5 [22400/60000 (37%)]	Loss: 0.441299
    Train Epoch: 5 [23040/60000 (38%)]	Loss: 0.472163
    Train Epoch: 5 [23680/60000 (39%)]	Loss: 0.565845
    Train Epoch: 5 [24320/60000 (41%)]	Loss: 0.585979
    Train Epoch: 5 [24960/60000 (42%)]	Loss: 0.654992
    Train Epoch: 5 [25600/60000 (43%)]	Loss: 0.646539
    Train Epoch: 5 [26240/60000 (44%)]	Loss: 0.327595
    Train Epoch: 5 [26880/60000 (45%)]	Loss: 0.361459
    Train Epoch: 5 [27520/60000 (46%)]	Loss: 0.527023
    Train Epoch: 5 [28160/60000 (47%)]	Loss: 0.510979
    Train Epoch: 5 [28800/60000 (48%)]	Loss: 0.596272
    Train Epoch: 5 [29440/60000 (49%)]	Loss: 0.641762
    Train Epoch: 5 [30080/60000 (50%)]	Loss: 0.352163
    Train Epoch: 5 [30720/60000 (51%)]	Loss: 0.477677
    Train Epoch: 5 [31360/60000 (52%)]	Loss: 0.331182
    Train Epoch: 5 [32000/60000 (53%)]	Loss: 0.546108
    Train Epoch: 5 [32640/60000 (54%)]	Loss: 0.691825
    Train Epoch: 5 [33280/60000 (55%)]	Loss: 0.432296
    Train Epoch: 5 [33920/60000 (57%)]	Loss: 0.293409
    Train Epoch: 5 [34560/60000 (58%)]	Loss: 0.461842
    Train Epoch: 5 [35200/60000 (59%)]	Loss: 0.441172
    Train Epoch: 5 [35840/60000 (60%)]	Loss: 0.450768
    Train Epoch: 5 [36480/60000 (61%)]	Loss: 0.479811
    Train Epoch: 5 [37120/60000 (62%)]	Loss: 0.368303
    Train Epoch: 5 [37760/60000 (63%)]	Loss: 0.714117
    Train Epoch: 5 [38400/60000 (64%)]	Loss: 0.512306
    Train Epoch: 5 [39040/60000 (65%)]	Loss: 0.353668
    Train Epoch: 5 [39680/60000 (66%)]	Loss: 0.634520
    Train Epoch: 5 [40320/60000 (67%)]	Loss: 0.508756
    Train Epoch: 5 [40960/60000 (68%)]	Loss: 0.574379
    Train Epoch: 5 [41600/60000 (69%)]	Loss: 0.515620
    Train Epoch: 5 [42240/60000 (70%)]	Loss: 0.340576
    Train Epoch: 5 [42880/60000 (71%)]	Loss: 0.285465
    Train Epoch: 5 [43520/60000 (72%)]	Loss: 0.502436
    Train Epoch: 5 [44160/60000 (74%)]	Loss: 0.399609
    Train Epoch: 5 [44800/60000 (75%)]	Loss: 0.348736
    Train Epoch: 5 [45440/60000 (76%)]	Loss: 0.346850
    Train Epoch: 5 [46080/60000 (77%)]	Loss: 0.276397
    Train Epoch: 5 [46720/60000 (78%)]	Loss: 0.838089
    Train Epoch: 5 [47360/60000 (79%)]	Loss: 0.402148
    Train Epoch: 5 [48000/60000 (80%)]	Loss: 0.303684
    Train Epoch: 5 [48640/60000 (81%)]	Loss: 0.553139
    Train Epoch: 5 [49280/60000 (82%)]	Loss: 0.497245
    Train Epoch: 5 [49920/60000 (83%)]	Loss: 0.535974
    Train Epoch: 5 [50560/60000 (84%)]	Loss: 0.429837
    Train Epoch: 5 [51200/60000 (85%)]	Loss: 0.462402
    Train Epoch: 5 [51840/60000 (86%)]	Loss: 0.443050
    Train Epoch: 5 [52480/60000 (87%)]	Loss: 0.449189
    Train Epoch: 5 [53120/60000 (88%)]	Loss: 0.407580
    Train Epoch: 5 [53760/60000 (90%)]	Loss: 0.709943
    Train Epoch: 5 [54400/60000 (91%)]	Loss: 0.663003
    Train Epoch: 5 [55040/60000 (92%)]	Loss: 0.664517
    Train Epoch: 5 [55680/60000 (93%)]	Loss: 0.559337
    Train Epoch: 5 [56320/60000 (94%)]	Loss: 0.369790
    Train Epoch: 5 [56960/60000 (95%)]	Loss: 0.673157
    Train Epoch: 5 [57600/60000 (96%)]	Loss: 0.338669
    Train Epoch: 5 [58240/60000 (97%)]	Loss: 0.492030
    Train Epoch: 5 [58880/60000 (98%)]	Loss: 0.344073
    Train Epoch: 5 [59520/60000 (99%)]	Loss: 0.422336
    
    Test set: Average loss: 0.2519, Accuracy: 9238/10000 (92%)
    
    Train Epoch: 6 [0/60000 (0%)]	Loss: 0.386451
    Train Epoch: 6 [640/60000 (1%)]	Loss: 0.457663
    Train Epoch: 6 [1280/60000 (2%)]	Loss: 0.515762
    Train Epoch: 6 [1920/60000 (3%)]	Loss: 0.612986
    Train Epoch: 6 [2560/60000 (4%)]	Loss: 0.787486
    Train Epoch: 6 [3200/60000 (5%)]	Loss: 0.491760
    Train Epoch: 6 [3840/60000 (6%)]	Loss: 0.454228
    Train Epoch: 6 [4480/60000 (7%)]	Loss: 0.359811
    Train Epoch: 6 [5120/60000 (9%)]	Loss: 0.368993
    Train Epoch: 6 [5760/60000 (10%)]	Loss: 0.442591
    Train Epoch: 6 [6400/60000 (11%)]	Loss: 0.597940
    Train Epoch: 6 [7040/60000 (12%)]	Loss: 0.383114
    Train Epoch: 6 [7680/60000 (13%)]	Loss: 0.362789
    Train Epoch: 6 [8320/60000 (14%)]	Loss: 0.514896
    Train Epoch: 6 [8960/60000 (15%)]	Loss: 0.774907
    Train Epoch: 6 [9600/60000 (16%)]	Loss: 0.390480
    Train Epoch: 6 [10240/60000 (17%)]	Loss: 0.584314
    Train Epoch: 6 [10880/60000 (18%)]	Loss: 0.288985
    Train Epoch: 6 [11520/60000 (19%)]	Loss: 0.426987
    Train Epoch: 6 [12160/60000 (20%)]	Loss: 0.278613
    Train Epoch: 6 [12800/60000 (21%)]	Loss: 0.499849
    Train Epoch: 6 [13440/60000 (22%)]	Loss: 0.431185
    Train Epoch: 6 [14080/60000 (23%)]	Loss: 0.689421
    Train Epoch: 6 [14720/60000 (25%)]	Loss: 0.337867
    Train Epoch: 6 [15360/60000 (26%)]	Loss: 0.626686
    Train Epoch: 6 [16000/60000 (27%)]	Loss: 0.497805
    Train Epoch: 6 [16640/60000 (28%)]	Loss: 0.441193
    Train Epoch: 6 [17280/60000 (29%)]	Loss: 0.561231
    Train Epoch: 6 [17920/60000 (30%)]	Loss: 0.401973
    Train Epoch: 6 [18560/60000 (31%)]	Loss: 0.561977
    Train Epoch: 6 [19200/60000 (32%)]	Loss: 0.410718
    Train Epoch: 6 [19840/60000 (33%)]	Loss: 0.770684
    Train Epoch: 6 [20480/60000 (34%)]	Loss: 0.639804
    Train Epoch: 6 [21120/60000 (35%)]	Loss: 0.302792
    Train Epoch: 6 [21760/60000 (36%)]	Loss: 0.529687
    Train Epoch: 6 [22400/60000 (37%)]	Loss: 0.717905
    Train Epoch: 6 [23040/60000 (38%)]	Loss: 0.498946
    Train Epoch: 6 [23680/60000 (39%)]	Loss: 0.429929
    Train Epoch: 6 [24320/60000 (41%)]	Loss: 0.435225
    Train Epoch: 6 [24960/60000 (42%)]	Loss: 0.320319
    Train Epoch: 6 [25600/60000 (43%)]	Loss: 0.590387
    Train Epoch: 6 [26240/60000 (44%)]	Loss: 0.265355
    Train Epoch: 6 [26880/60000 (45%)]	Loss: 0.454372
    Train Epoch: 6 [27520/60000 (46%)]	Loss: 0.790875
    Train Epoch: 6 [28160/60000 (47%)]	Loss: 0.486921
    Train Epoch: 6 [28800/60000 (48%)]	Loss: 0.462752
    Train Epoch: 6 [29440/60000 (49%)]	Loss: 0.813336
    Train Epoch: 6 [30080/60000 (50%)]	Loss: 0.308711
    Train Epoch: 6 [30720/60000 (51%)]	Loss: 0.476948
    Train Epoch: 6 [31360/60000 (52%)]	Loss: 0.649331
    Train Epoch: 6 [32000/60000 (53%)]	Loss: 0.337971
    Train Epoch: 6 [32640/60000 (54%)]	Loss: 0.552407
    Train Epoch: 6 [33280/60000 (55%)]	Loss: 0.584258
    Train Epoch: 6 [33920/60000 (57%)]	Loss: 0.682540
    Train Epoch: 6 [34560/60000 (58%)]	Loss: 0.472494
    Train Epoch: 6 [35200/60000 (59%)]	Loss: 0.581826
    Train Epoch: 6 [35840/60000 (60%)]	Loss: 0.430555
    Train Epoch: 6 [36480/60000 (61%)]	Loss: 0.408300
    Train Epoch: 6 [37120/60000 (62%)]	Loss: 0.544223
    Train Epoch: 6 [37760/60000 (63%)]	Loss: 0.276038
    Train Epoch: 6 [38400/60000 (64%)]	Loss: 0.383865
    Train Epoch: 6 [39040/60000 (65%)]	Loss: 0.486723
    Train Epoch: 6 [39680/60000 (66%)]	Loss: 0.401155
    Train Epoch: 6 [40320/60000 (67%)]	Loss: 0.501816
    Train Epoch: 6 [40960/60000 (68%)]	Loss: 0.514987
    Train Epoch: 6 [41600/60000 (69%)]	Loss: 0.501831
    Train Epoch: 6 [42240/60000 (70%)]	Loss: 0.471296
    Train Epoch: 6 [42880/60000 (71%)]	Loss: 0.467298
    Train Epoch: 6 [43520/60000 (72%)]	Loss: 0.421591
    Train Epoch: 6 [44160/60000 (74%)]	Loss: 0.485595
    Train Epoch: 6 [44800/60000 (75%)]	Loss: 0.450340
    Train Epoch: 6 [45440/60000 (76%)]	Loss: 0.339639
    Train Epoch: 6 [46080/60000 (77%)]	Loss: 0.386936
    Train Epoch: 6 [46720/60000 (78%)]	Loss: 0.288080
    Train Epoch: 6 [47360/60000 (79%)]	Loss: 0.448823
    Train Epoch: 6 [48000/60000 (80%)]	Loss: 0.774343
    Train Epoch: 6 [48640/60000 (81%)]	Loss: 0.379256
    Train Epoch: 6 [49280/60000 (82%)]	Loss: 0.430137
    Train Epoch: 6 [49920/60000 (83%)]	Loss: 0.486229
    Train Epoch: 6 [50560/60000 (84%)]	Loss: 0.548015
    Train Epoch: 6 [51200/60000 (85%)]	Loss: 0.312752
    Train Epoch: 6 [51840/60000 (86%)]	Loss: 0.405820
    Train Epoch: 6 [52480/60000 (87%)]	Loss: 0.346440
    Train Epoch: 6 [53120/60000 (88%)]	Loss: 0.289083
    Train Epoch: 6 [53760/60000 (90%)]	Loss: 0.595599
    Train Epoch: 6 [54400/60000 (91%)]	Loss: 0.303218
    Train Epoch: 6 [55040/60000 (92%)]	Loss: 0.461978
    Train Epoch: 6 [55680/60000 (93%)]	Loss: 0.425981
    Train Epoch: 6 [56320/60000 (94%)]	Loss: 0.318439
    Train Epoch: 6 [56960/60000 (95%)]	Loss: 0.555306
    Train Epoch: 6 [57600/60000 (96%)]	Loss: 0.662118
    Train Epoch: 6 [58240/60000 (97%)]	Loss: 0.489320
    Train Epoch: 6 [58880/60000 (98%)]	Loss: 0.406899
    Train Epoch: 6 [59520/60000 (99%)]	Loss: 0.385348
    
    Test set: Average loss: 0.2355, Accuracy: 9277/10000 (93%)
    
    Train Epoch: 7 [0/60000 (0%)]	Loss: 0.717746
    Train Epoch: 7 [640/60000 (1%)]	Loss: 0.469850
    Train Epoch: 7 [1280/60000 (2%)]	Loss: 0.594132
    Train Epoch: 7 [1920/60000 (3%)]	Loss: 0.475335
    Train Epoch: 7 [2560/60000 (4%)]	Loss: 0.430496
    Train Epoch: 7 [3200/60000 (5%)]	Loss: 0.294112
    Train Epoch: 7 [3840/60000 (6%)]	Loss: 0.312968
    Train Epoch: 7 [4480/60000 (7%)]	Loss: 0.362220
    Train Epoch: 7 [5120/60000 (9%)]	Loss: 0.429730
    Train Epoch: 7 [5760/60000 (10%)]	Loss: 0.357846
    Train Epoch: 7 [6400/60000 (11%)]	Loss: 0.336342
    Train Epoch: 7 [7040/60000 (12%)]	Loss: 0.553370
    Train Epoch: 7 [7680/60000 (13%)]	Loss: 0.517778
    Train Epoch: 7 [8320/60000 (14%)]	Loss: 0.441374
    Train Epoch: 7 [8960/60000 (15%)]	Loss: 0.242141
    Train Epoch: 7 [9600/60000 (16%)]	Loss: 0.288597
    Train Epoch: 7 [10240/60000 (17%)]	Loss: 0.355947
    Train Epoch: 7 [10880/60000 (18%)]	Loss: 0.225561
    Train Epoch: 7 [11520/60000 (19%)]	Loss: 0.556642
    Train Epoch: 7 [12160/60000 (20%)]	Loss: 0.426134
    Train Epoch: 7 [12800/60000 (21%)]	Loss: 0.408436
    Train Epoch: 7 [13440/60000 (22%)]	Loss: 0.452092
    Train Epoch: 7 [14080/60000 (23%)]	Loss: 0.417876
    Train Epoch: 7 [14720/60000 (25%)]	Loss: 0.312885
    Train Epoch: 7 [15360/60000 (26%)]	Loss: 0.513127
    Train Epoch: 7 [16000/60000 (27%)]	Loss: 0.371684
    Train Epoch: 7 [16640/60000 (28%)]	Loss: 0.347489
    Train Epoch: 7 [17280/60000 (29%)]	Loss: 0.463195
    Train Epoch: 7 [17920/60000 (30%)]	Loss: 0.391325
    Train Epoch: 7 [18560/60000 (31%)]	Loss: 0.483348
    Train Epoch: 7 [19200/60000 (32%)]	Loss: 0.341747
    Train Epoch: 7 [19840/60000 (33%)]	Loss: 0.484753
    Train Epoch: 7 [20480/60000 (34%)]	Loss: 0.342775
    Train Epoch: 7 [21120/60000 (35%)]	Loss: 0.680684
    Train Epoch: 7 [21760/60000 (36%)]	Loss: 0.297526
    Train Epoch: 7 [22400/60000 (37%)]	Loss: 0.473823
    Train Epoch: 7 [23040/60000 (38%)]	Loss: 0.535453
    Train Epoch: 7 [23680/60000 (39%)]	Loss: 0.457003
    Train Epoch: 7 [24320/60000 (41%)]	Loss: 0.428764
    Train Epoch: 7 [24960/60000 (42%)]	Loss: 0.437032
    Train Epoch: 7 [25600/60000 (43%)]	Loss: 0.626991
    Train Epoch: 7 [26240/60000 (44%)]	Loss: 0.401498
    Train Epoch: 7 [26880/60000 (45%)]	Loss: 0.341815
    Train Epoch: 7 [27520/60000 (46%)]	Loss: 0.347058
    Train Epoch: 7 [28160/60000 (47%)]	Loss: 0.592645
    Train Epoch: 7 [28800/60000 (48%)]	Loss: 0.486121
    Train Epoch: 7 [29440/60000 (49%)]	Loss: 0.521025
    Train Epoch: 7 [30080/60000 (50%)]	Loss: 0.396133
    Train Epoch: 7 [30720/60000 (51%)]	Loss: 0.568312
    Train Epoch: 7 [31360/60000 (52%)]	Loss: 0.475080
    Train Epoch: 7 [32000/60000 (53%)]	Loss: 0.496030
    Train Epoch: 7 [32640/60000 (54%)]	Loss: 0.321438
    Train Epoch: 7 [33280/60000 (55%)]	Loss: 0.361846
    Train Epoch: 7 [33920/60000 (57%)]	Loss: 0.436478
    Train Epoch: 7 [34560/60000 (58%)]	Loss: 0.532364
    Train Epoch: 7 [35200/60000 (59%)]	Loss: 0.510952
    Train Epoch: 7 [35840/60000 (60%)]	Loss: 0.645716
    Train Epoch: 7 [36480/60000 (61%)]	Loss: 0.459233
    Train Epoch: 7 [37120/60000 (62%)]	Loss: 0.372445
    Train Epoch: 7 [37760/60000 (63%)]	Loss: 0.232452
    Train Epoch: 7 [38400/60000 (64%)]	Loss: 0.349685
    Train Epoch: 7 [39040/60000 (65%)]	Loss: 0.594317
    Train Epoch: 7 [39680/60000 (66%)]	Loss: 0.716788
    Train Epoch: 7 [40320/60000 (67%)]	Loss: 0.736326
    Train Epoch: 7 [40960/60000 (68%)]	Loss: 0.434928
    Train Epoch: 7 [41600/60000 (69%)]	Loss: 0.504802
    Train Epoch: 7 [42240/60000 (70%)]	Loss: 0.458648
    Train Epoch: 7 [42880/60000 (71%)]	Loss: 0.433149
    Train Epoch: 7 [43520/60000 (72%)]	Loss: 0.291753
    Train Epoch: 7 [44160/60000 (74%)]	Loss: 0.414158
    Train Epoch: 7 [44800/60000 (75%)]	Loss: 0.387175
    Train Epoch: 7 [45440/60000 (76%)]	Loss: 0.412587
    Train Epoch: 7 [46080/60000 (77%)]	Loss: 0.396877
    Train Epoch: 7 [46720/60000 (78%)]	Loss: 0.497912
    Train Epoch: 7 [47360/60000 (79%)]	Loss: 0.428157
    Train Epoch: 7 [48000/60000 (80%)]	Loss: 0.457888
    Train Epoch: 7 [48640/60000 (81%)]	Loss: 0.519679
    Train Epoch: 7 [49280/60000 (82%)]	Loss: 0.357949
    Train Epoch: 7 [49920/60000 (83%)]	Loss: 0.349139
    Train Epoch: 7 [50560/60000 (84%)]	Loss: 0.389948
    Train Epoch: 7 [51200/60000 (85%)]	Loss: 0.426888
    Train Epoch: 7 [51840/60000 (86%)]	Loss: 0.348460
    Train Epoch: 7 [52480/60000 (87%)]	Loss: 0.596196
    Train Epoch: 7 [53120/60000 (88%)]	Loss: 0.567125
    Train Epoch: 7 [53760/60000 (90%)]	Loss: 0.301156
    Train Epoch: 7 [54400/60000 (91%)]	Loss: 0.650556
    Train Epoch: 7 [55040/60000 (92%)]	Loss: 0.716238
    Train Epoch: 7 [55680/60000 (93%)]	Loss: 0.478881
    Train Epoch: 7 [56320/60000 (94%)]	Loss: 0.421738
    Train Epoch: 7 [56960/60000 (95%)]	Loss: 0.435452
    Train Epoch: 7 [57600/60000 (96%)]	Loss: 0.639111
    Train Epoch: 7 [58240/60000 (97%)]	Loss: 0.387537
    Train Epoch: 7 [58880/60000 (98%)]	Loss: 0.839673
    Train Epoch: 7 [59520/60000 (99%)]	Loss: 0.409900
    
    Test set: Average loss: 0.2244, Accuracy: 9333/10000 (93%)
    
    Train Epoch: 8 [0/60000 (0%)]	Loss: 0.469117
    Train Epoch: 8 [640/60000 (1%)]	Loss: 0.369546
    Train Epoch: 8 [1280/60000 (2%)]	Loss: 0.205326
    Train Epoch: 8 [1920/60000 (3%)]	Loss: 0.377605
    Train Epoch: 8 [2560/60000 (4%)]	Loss: 0.759715
    Train Epoch: 8 [3200/60000 (5%)]	Loss: 0.435699
    Train Epoch: 8 [3840/60000 (6%)]	Loss: 0.496597
    Train Epoch: 8 [4480/60000 (7%)]	Loss: 0.382842
    Train Epoch: 8 [5120/60000 (9%)]	Loss: 0.572179
    Train Epoch: 8 [5760/60000 (10%)]	Loss: 0.510330
    Train Epoch: 8 [6400/60000 (11%)]	Loss: 0.479856
    Train Epoch: 8 [7040/60000 (12%)]	Loss: 0.630408
    Train Epoch: 8 [7680/60000 (13%)]	Loss: 0.418155
    Train Epoch: 8 [8320/60000 (14%)]	Loss: 0.401250
    Train Epoch: 8 [8960/60000 (15%)]	Loss: 0.618374
    Train Epoch: 8 [9600/60000 (16%)]	Loss: 0.614909
    Train Epoch: 8 [10240/60000 (17%)]	Loss: 0.318959
    Train Epoch: 8 [10880/60000 (18%)]	Loss: 0.337133
    Train Epoch: 8 [11520/60000 (19%)]	Loss: 0.797270
    Train Epoch: 8 [12160/60000 (20%)]	Loss: 0.405077
    Train Epoch: 8 [12800/60000 (21%)]	Loss: 0.660093
    Train Epoch: 8 [13440/60000 (22%)]	Loss: 0.607703
    Train Epoch: 8 [14080/60000 (23%)]	Loss: 0.496708
    Train Epoch: 8 [14720/60000 (25%)]	Loss: 0.288580
    Train Epoch: 8 [15360/60000 (26%)]	Loss: 0.542241
    Train Epoch: 8 [16000/60000 (27%)]	Loss: 0.460526
    Train Epoch: 8 [16640/60000 (28%)]	Loss: 0.513786
    Train Epoch: 8 [17280/60000 (29%)]	Loss: 0.357061
    Train Epoch: 8 [17920/60000 (30%)]	Loss: 0.301968
    Train Epoch: 8 [18560/60000 (31%)]	Loss: 0.418004
    Train Epoch: 8 [19200/60000 (32%)]	Loss: 0.445466
    Train Epoch: 8 [19840/60000 (33%)]	Loss: 0.381778
    Train Epoch: 8 [20480/60000 (34%)]	Loss: 0.454850
    Train Epoch: 8 [21120/60000 (35%)]	Loss: 0.311810
    Train Epoch: 8 [21760/60000 (36%)]	Loss: 0.547685
    Train Epoch: 8 [22400/60000 (37%)]	Loss: 0.196215
    Train Epoch: 8 [23040/60000 (38%)]	Loss: 0.286037
    Train Epoch: 8 [23680/60000 (39%)]	Loss: 0.477281
    Train Epoch: 8 [24320/60000 (41%)]	Loss: 0.818387
    Train Epoch: 8 [24960/60000 (42%)]	Loss: 0.514256
    Train Epoch: 8 [25600/60000 (43%)]	Loss: 0.455588
    Train Epoch: 8 [26240/60000 (44%)]	Loss: 0.365949
    Train Epoch: 8 [26880/60000 (45%)]	Loss: 0.358121
    Train Epoch: 8 [27520/60000 (46%)]	Loss: 0.453270
    Train Epoch: 8 [28160/60000 (47%)]	Loss: 0.543010
    Train Epoch: 8 [28800/60000 (48%)]	Loss: 0.643081
    Train Epoch: 8 [29440/60000 (49%)]	Loss: 0.510997
    Train Epoch: 8 [30080/60000 (50%)]	Loss: 0.316055
    Train Epoch: 8 [30720/60000 (51%)]	Loss: 0.675489
    Train Epoch: 8 [31360/60000 (52%)]	Loss: 0.303624
    Train Epoch: 8 [32000/60000 (53%)]	Loss: 0.449534
    Train Epoch: 8 [32640/60000 (54%)]	Loss: 0.451441
    Train Epoch: 8 [33280/60000 (55%)]	Loss: 0.478364
    Train Epoch: 8 [33920/60000 (57%)]	Loss: 0.425091
    Train Epoch: 8 [34560/60000 (58%)]	Loss: 0.211938
    Train Epoch: 8 [35200/60000 (59%)]	Loss: 0.356066
    Train Epoch: 8 [35840/60000 (60%)]	Loss: 0.646257
    Train Epoch: 8 [36480/60000 (61%)]	Loss: 0.643567
    Train Epoch: 8 [37120/60000 (62%)]	Loss: 0.322013
    Train Epoch: 8 [37760/60000 (63%)]	Loss: 0.407144
    Train Epoch: 8 [38400/60000 (64%)]	Loss: 0.543189
    Train Epoch: 8 [39040/60000 (65%)]	Loss: 0.287052
    Train Epoch: 8 [39680/60000 (66%)]	Loss: 0.351675
    Train Epoch: 8 [40320/60000 (67%)]	Loss: 0.288525
    Train Epoch: 8 [40960/60000 (68%)]	Loss: 0.453517
    Train Epoch: 8 [41600/60000 (69%)]	Loss: 0.253906
    Train Epoch: 8 [42240/60000 (70%)]	Loss: 0.512110
    Train Epoch: 8 [42880/60000 (71%)]	Loss: 0.590715
    Train Epoch: 8 [43520/60000 (72%)]	Loss: 0.325584
    Train Epoch: 8 [44160/60000 (74%)]	Loss: 0.482525
    Train Epoch: 8 [44800/60000 (75%)]	Loss: 0.337738
    Train Epoch: 8 [45440/60000 (76%)]	Loss: 0.318561
    Train Epoch: 8 [46080/60000 (77%)]	Loss: 0.341067
    Train Epoch: 8 [46720/60000 (78%)]	Loss: 0.545489
    Train Epoch: 8 [47360/60000 (79%)]	Loss: 0.402002
    Train Epoch: 8 [48000/60000 (80%)]	Loss: 0.231705
    Train Epoch: 8 [48640/60000 (81%)]	Loss: 0.242956
    Train Epoch: 8 [49280/60000 (82%)]	Loss: 0.426706
    Train Epoch: 8 [49920/60000 (83%)]	Loss: 0.341219
    Train Epoch: 8 [50560/60000 (84%)]	Loss: 0.422939
    Train Epoch: 8 [51200/60000 (85%)]	Loss: 0.410270
    Train Epoch: 8 [51840/60000 (86%)]	Loss: 0.443087
    Train Epoch: 8 [52480/60000 (87%)]	Loss: 0.273087
    Train Epoch: 8 [53120/60000 (88%)]	Loss: 0.300433
    Train Epoch: 8 [53760/60000 (90%)]	Loss: 0.408494
    Train Epoch: 8 [54400/60000 (91%)]	Loss: 0.410628
    Train Epoch: 8 [55040/60000 (92%)]	Loss: 0.481743
    Train Epoch: 8 [55680/60000 (93%)]	Loss: 0.532843
    Train Epoch: 8 [56320/60000 (94%)]	Loss: 0.255752
    Train Epoch: 8 [56960/60000 (95%)]	Loss: 0.287013
    Train Epoch: 8 [57600/60000 (96%)]	Loss: 0.429710
    Train Epoch: 8 [58240/60000 (97%)]	Loss: 0.377912
    Train Epoch: 8 [58880/60000 (98%)]	Loss: 0.560696
    Train Epoch: 8 [59520/60000 (99%)]	Loss: 0.380459
    
    Test set: Average loss: 0.2163, Accuracy: 9362/10000 (94%)
    
    Train Epoch: 9 [0/60000 (0%)]	Loss: 0.585350
    Train Epoch: 9 [640/60000 (1%)]	Loss: 0.493246
    Train Epoch: 9 [1280/60000 (2%)]	Loss: 0.391806
    Train Epoch: 9 [1920/60000 (3%)]	Loss: 0.493008
    Train Epoch: 9 [2560/60000 (4%)]	Loss: 0.448494
    Train Epoch: 9 [3200/60000 (5%)]	Loss: 0.325095
    Train Epoch: 9 [3840/60000 (6%)]	Loss: 0.695937
    Train Epoch: 9 [4480/60000 (7%)]	Loss: 0.266650
    Train Epoch: 9 [5120/60000 (9%)]	Loss: 0.420216
    Train Epoch: 9 [5760/60000 (10%)]	Loss: 0.353440
    Train Epoch: 9 [6400/60000 (11%)]	Loss: 0.341078
    Train Epoch: 9 [7040/60000 (12%)]	Loss: 0.439247
    Train Epoch: 9 [7680/60000 (13%)]	Loss: 0.214539
    Train Epoch: 9 [8320/60000 (14%)]	Loss: 0.469013
    Train Epoch: 9 [8960/60000 (15%)]	Loss: 0.341292
    Train Epoch: 9 [9600/60000 (16%)]	Loss: 0.785741
    Train Epoch: 9 [10240/60000 (17%)]	Loss: 0.466753
    Train Epoch: 9 [10880/60000 (18%)]	Loss: 0.418933
    Train Epoch: 9 [11520/60000 (19%)]	Loss: 0.352860
    Train Epoch: 9 [12160/60000 (20%)]	Loss: 0.330622
    Train Epoch: 9 [12800/60000 (21%)]	Loss: 0.394191
    Train Epoch: 9 [13440/60000 (22%)]	Loss: 0.304991
    Train Epoch: 9 [14080/60000 (23%)]	Loss: 0.291812
    Train Epoch: 9 [14720/60000 (25%)]	Loss: 0.460314
    Train Epoch: 9 [15360/60000 (26%)]	Loss: 0.462962
    Train Epoch: 9 [16000/60000 (27%)]	Loss: 0.573508
    Train Epoch: 9 [16640/60000 (28%)]	Loss: 0.424545
    Train Epoch: 9 [17280/60000 (29%)]	Loss: 0.314215
    Train Epoch: 9 [17920/60000 (30%)]	Loss: 0.399477
    Train Epoch: 9 [18560/60000 (31%)]	Loss: 0.281409
    Train Epoch: 9 [19200/60000 (32%)]	Loss: 0.491287
    Train Epoch: 9 [19840/60000 (33%)]	Loss: 0.478374
    Train Epoch: 9 [20480/60000 (34%)]	Loss: 0.580464
    Train Epoch: 9 [21120/60000 (35%)]	Loss: 0.456699
    Train Epoch: 9 [21760/60000 (36%)]	Loss: 0.328621
    Train Epoch: 9 [22400/60000 (37%)]	Loss: 0.444202
    Train Epoch: 9 [23040/60000 (38%)]	Loss: 0.337673
    Train Epoch: 9 [23680/60000 (39%)]	Loss: 0.385429
    Train Epoch: 9 [24320/60000 (41%)]	Loss: 0.408061
    Train Epoch: 9 [24960/60000 (42%)]	Loss: 0.261543
    Train Epoch: 9 [25600/60000 (43%)]	Loss: 0.307577
    Train Epoch: 9 [26240/60000 (44%)]	Loss: 0.340200
    Train Epoch: 9 [26880/60000 (45%)]	Loss: 0.251914
    Train Epoch: 9 [27520/60000 (46%)]	Loss: 0.269231
    Train Epoch: 9 [28160/60000 (47%)]	Loss: 0.456552
    Train Epoch: 9 [28800/60000 (48%)]	Loss: 0.598232
    Train Epoch: 9 [29440/60000 (49%)]	Loss: 0.418177
    Train Epoch: 9 [30080/60000 (50%)]	Loss: 0.356407
    Train Epoch: 9 [30720/60000 (51%)]	Loss: 0.392345
    Train Epoch: 9 [31360/60000 (52%)]	Loss: 0.379441
    Train Epoch: 9 [32000/60000 (53%)]	Loss: 0.465713
    Train Epoch: 9 [32640/60000 (54%)]	Loss: 0.367991
    Train Epoch: 9 [33280/60000 (55%)]	Loss: 0.285676
    Train Epoch: 9 [33920/60000 (57%)]	Loss: 0.243431
    Train Epoch: 9 [34560/60000 (58%)]	Loss: 0.355942
    Train Epoch: 9 [35200/60000 (59%)]	Loss: 0.374828
    Train Epoch: 9 [35840/60000 (60%)]	Loss: 0.277245
    Train Epoch: 9 [36480/60000 (61%)]	Loss: 0.273998
    Train Epoch: 9 [37120/60000 (62%)]	Loss: 0.406776
    Train Epoch: 9 [37760/60000 (63%)]	Loss: 0.651791
    Train Epoch: 9 [38400/60000 (64%)]	Loss: 0.417006
    Train Epoch: 9 [39040/60000 (65%)]	Loss: 0.287786
    Train Epoch: 9 [39680/60000 (66%)]	Loss: 0.592248
    Train Epoch: 9 [40320/60000 (67%)]	Loss: 0.317200
    Train Epoch: 9 [40960/60000 (68%)]	Loss: 0.324063
    Train Epoch: 9 [41600/60000 (69%)]	Loss: 0.393426
    Train Epoch: 9 [42240/60000 (70%)]	Loss: 0.413506
    Train Epoch: 9 [42880/60000 (71%)]	Loss: 0.633301
    Train Epoch: 9 [43520/60000 (72%)]	Loss: 0.276478
    Train Epoch: 9 [44160/60000 (74%)]	Loss: 0.473216
    Train Epoch: 9 [44800/60000 (75%)]	Loss: 0.327980
    Train Epoch: 9 [45440/60000 (76%)]	Loss: 0.727830
    Train Epoch: 9 [46080/60000 (77%)]	Loss: 0.416605
    Train Epoch: 9 [46720/60000 (78%)]	Loss: 0.407099
    Train Epoch: 9 [47360/60000 (79%)]	Loss: 0.375051
    Train Epoch: 9 [48000/60000 (80%)]	Loss: 0.488992
    Train Epoch: 9 [48640/60000 (81%)]	Loss: 0.413114
    Train Epoch: 9 [49280/60000 (82%)]	Loss: 0.520725
    Train Epoch: 9 [49920/60000 (83%)]	Loss: 0.420221
    Train Epoch: 9 [50560/60000 (84%)]	Loss: 0.599522
    Train Epoch: 9 [51200/60000 (85%)]	Loss: 0.490780
    Train Epoch: 9 [51840/60000 (86%)]	Loss: 0.228232
    Train Epoch: 9 [52480/60000 (87%)]	Loss: 0.347773
    Train Epoch: 9 [53120/60000 (88%)]	Loss: 0.476633
    Train Epoch: 9 [53760/60000 (90%)]	Loss: 0.256656
    Train Epoch: 9 [54400/60000 (91%)]	Loss: 0.396474
    Train Epoch: 9 [55040/60000 (92%)]	Loss: 0.328017
    Train Epoch: 9 [55680/60000 (93%)]	Loss: 0.355086
    Train Epoch: 9 [56320/60000 (94%)]	Loss: 0.354232
    Train Epoch: 9 [56960/60000 (95%)]	Loss: 0.360218
    Train Epoch: 9 [57600/60000 (96%)]	Loss: 0.332373
    Train Epoch: 9 [58240/60000 (97%)]	Loss: 0.364290
    Train Epoch: 9 [58880/60000 (98%)]	Loss: 0.261339
    Train Epoch: 9 [59520/60000 (99%)]	Loss: 0.250586
    
    Test set: Average loss: 0.2151, Accuracy: 9366/10000 (94%)
    
    Train Epoch: 10 [0/60000 (0%)]	Loss: 0.438674
    Train Epoch: 10 [640/60000 (1%)]	Loss: 0.447094
    Train Epoch: 10 [1280/60000 (2%)]	Loss: 0.303145
    Train Epoch: 10 [1920/60000 (3%)]	Loss: 0.327251
    Train Epoch: 10 [2560/60000 (4%)]	Loss: 0.238297
    Train Epoch: 10 [3200/60000 (5%)]	Loss: 0.383331
    Train Epoch: 10 [3840/60000 (6%)]	Loss: 0.382009
    Train Epoch: 10 [4480/60000 (7%)]	Loss: 0.389430
    Train Epoch: 10 [5120/60000 (9%)]	Loss: 0.295570
    Train Epoch: 10 [5760/60000 (10%)]	Loss: 0.259864
    Train Epoch: 10 [6400/60000 (11%)]	Loss: 0.495970
    Train Epoch: 10 [7040/60000 (12%)]	Loss: 0.361643
    Train Epoch: 10 [7680/60000 (13%)]	Loss: 0.765771
    Train Epoch: 10 [8320/60000 (14%)]	Loss: 0.403898
    Train Epoch: 10 [8960/60000 (15%)]	Loss: 0.209247
    Train Epoch: 10 [9600/60000 (16%)]	Loss: 0.482393
    Train Epoch: 10 [10240/60000 (17%)]	Loss: 0.459047
    Train Epoch: 10 [10880/60000 (18%)]	Loss: 0.505761
    Train Epoch: 10 [11520/60000 (19%)]	Loss: 0.433308
    Train Epoch: 10 [12160/60000 (20%)]	Loss: 0.354521
    Train Epoch: 10 [12800/60000 (21%)]	Loss: 0.233018
    Train Epoch: 10 [13440/60000 (22%)]	Loss: 0.390475
    Train Epoch: 10 [14080/60000 (23%)]	Loss: 0.245935
    Train Epoch: 10 [14720/60000 (25%)]	Loss: 0.398528
    Train Epoch: 10 [15360/60000 (26%)]	Loss: 0.393017
    Train Epoch: 10 [16000/60000 (27%)]	Loss: 0.364166
    Train Epoch: 10 [16640/60000 (28%)]	Loss: 0.657179
    Train Epoch: 10 [17280/60000 (29%)]	Loss: 0.199565
    Train Epoch: 10 [17920/60000 (30%)]	Loss: 0.373811
    Train Epoch: 10 [18560/60000 (31%)]	Loss: 0.395341
    Train Epoch: 10 [19200/60000 (32%)]	Loss: 0.367141
    Train Epoch: 10 [19840/60000 (33%)]	Loss: 0.420444
    Train Epoch: 10 [20480/60000 (34%)]	Loss: 0.411721
    Train Epoch: 10 [21120/60000 (35%)]	Loss: 0.406184
    Train Epoch: 10 [21760/60000 (36%)]	Loss: 0.309357
    Train Epoch: 10 [22400/60000 (37%)]	Loss: 0.397585
    Train Epoch: 10 [23040/60000 (38%)]	Loss: 0.699485
    Train Epoch: 10 [23680/60000 (39%)]	Loss: 0.672690
    Train Epoch: 10 [24320/60000 (41%)]	Loss: 0.383667
    Train Epoch: 10 [24960/60000 (42%)]	Loss: 0.443057
    Train Epoch: 10 [25600/60000 (43%)]	Loss: 0.409219
    Train Epoch: 10 [26240/60000 (44%)]	Loss: 0.311079
    Train Epoch: 10 [26880/60000 (45%)]	Loss: 0.367074
    Train Epoch: 10 [27520/60000 (46%)]	Loss: 0.279823
    Train Epoch: 10 [28160/60000 (47%)]	Loss: 0.337272
    Train Epoch: 10 [28800/60000 (48%)]	Loss: 0.485713
    Train Epoch: 10 [29440/60000 (49%)]	Loss: 0.345926
    Train Epoch: 10 [30080/60000 (50%)]	Loss: 0.424248
    Train Epoch: 10 [30720/60000 (51%)]	Loss: 0.322441
    Train Epoch: 10 [31360/60000 (52%)]	Loss: 0.283901
    Train Epoch: 10 [32000/60000 (53%)]	Loss: 0.640329
    Train Epoch: 10 [32640/60000 (54%)]	Loss: 0.342490
    Train Epoch: 10 [33280/60000 (55%)]	Loss: 0.343811
    Train Epoch: 10 [33920/60000 (57%)]	Loss: 0.392110
    Train Epoch: 10 [34560/60000 (58%)]	Loss: 0.433465
    Train Epoch: 10 [35200/60000 (59%)]	Loss: 0.341572
    Train Epoch: 10 [35840/60000 (60%)]	Loss: 0.394995
    Train Epoch: 10 [36480/60000 (61%)]	Loss: 0.332045
    Train Epoch: 10 [37120/60000 (62%)]	Loss: 0.276502
    Train Epoch: 10 [37760/60000 (63%)]	Loss: 0.292657
    Train Epoch: 10 [38400/60000 (64%)]	Loss: 0.455167
    Train Epoch: 10 [39040/60000 (65%)]	Loss: 0.297509
    Train Epoch: 10 [39680/60000 (66%)]	Loss: 0.640905
    Train Epoch: 10 [40320/60000 (67%)]	Loss: 0.422916
    Train Epoch: 10 [40960/60000 (68%)]	Loss: 0.473346
    Train Epoch: 10 [41600/60000 (69%)]	Loss: 0.491302
    Train Epoch: 10 [42240/60000 (70%)]	Loss: 0.346931
    Train Epoch: 10 [42880/60000 (71%)]	Loss: 0.572828
    Train Epoch: 10 [43520/60000 (72%)]	Loss: 0.365607
    Train Epoch: 10 [44160/60000 (74%)]	Loss: 0.317555
    Train Epoch: 10 [44800/60000 (75%)]	Loss: 0.468910
    Train Epoch: 10 [45440/60000 (76%)]	Loss: 0.496312
    Train Epoch: 10 [46080/60000 (77%)]	Loss: 0.696475
    Train Epoch: 10 [46720/60000 (78%)]	Loss: 0.359580
    Train Epoch: 10 [47360/60000 (79%)]	Loss: 0.419243
    Train Epoch: 10 [48000/60000 (80%)]	Loss: 0.303316
    Train Epoch: 10 [48640/60000 (81%)]	Loss: 0.383328
    Train Epoch: 10 [49280/60000 (82%)]	Loss: 0.268373
    Train Epoch: 10 [49920/60000 (83%)]	Loss: 0.413617
    Train Epoch: 10 [50560/60000 (84%)]	Loss: 0.454594
    Train Epoch: 10 [51200/60000 (85%)]	Loss: 0.359163
    Train Epoch: 10 [51840/60000 (86%)]	Loss: 0.630097
    Train Epoch: 10 [52480/60000 (87%)]	Loss: 0.521165
    Train Epoch: 10 [53120/60000 (88%)]	Loss: 0.247819
    Train Epoch: 10 [53760/60000 (90%)]	Loss: 0.330510
    Train Epoch: 10 [54400/60000 (91%)]	Loss: 0.343167
    Train Epoch: 10 [55040/60000 (92%)]	Loss: 0.380156
    Train Epoch: 10 [55680/60000 (93%)]	Loss: 0.395422
    Train Epoch: 10 [56320/60000 (94%)]	Loss: 0.687743
    Train Epoch: 10 [56960/60000 (95%)]	Loss: 0.470193
    Train Epoch: 10 [57600/60000 (96%)]	Loss: 0.473724
    Train Epoch: 10 [58240/60000 (97%)]	Loss: 0.361689
    Train Epoch: 10 [58880/60000 (98%)]	Loss: 0.349370
    Train Epoch: 10 [59520/60000 (99%)]	Loss: 0.385800
    
    Test set: Average loss: 0.2124, Accuracy: 9367/10000 (94%)
    
    Train Epoch: 11 [0/60000 (0%)]	Loss: 0.426175
    Train Epoch: 11 [640/60000 (1%)]	Loss: 0.170051
    Train Epoch: 11 [1280/60000 (2%)]	Loss: 0.250144
    Train Epoch: 11 [1920/60000 (3%)]	Loss: 0.172225
    Train Epoch: 11 [2560/60000 (4%)]	Loss: 0.421107
    Train Epoch: 11 [3200/60000 (5%)]	Loss: 0.380877
    Train Epoch: 11 [3840/60000 (6%)]	Loss: 0.230397
    Train Epoch: 11 [4480/60000 (7%)]	Loss: 0.477565
    Train Epoch: 11 [5120/60000 (9%)]	Loss: 0.395525
    Train Epoch: 11 [5760/60000 (10%)]	Loss: 0.270285
    Train Epoch: 11 [6400/60000 (11%)]	Loss: 0.310442
    Train Epoch: 11 [7040/60000 (12%)]	Loss: 0.285871
    Train Epoch: 11 [7680/60000 (13%)]	Loss: 0.333100
    Train Epoch: 11 [8320/60000 (14%)]	Loss: 0.269914
    Train Epoch: 11 [8960/60000 (15%)]	Loss: 0.340485
    Train Epoch: 11 [9600/60000 (16%)]	Loss: 0.433936
    Train Epoch: 11 [10240/60000 (17%)]	Loss: 0.552323
    Train Epoch: 11 [10880/60000 (18%)]	Loss: 0.532913
    Train Epoch: 11 [11520/60000 (19%)]	Loss: 0.495746
    Train Epoch: 11 [12160/60000 (20%)]	Loss: 0.303815
    Train Epoch: 11 [12800/60000 (21%)]	Loss: 0.264451
    Train Epoch: 11 [13440/60000 (22%)]	Loss: 0.436694
    Train Epoch: 11 [14080/60000 (23%)]	Loss: 0.440698
    Train Epoch: 11 [14720/60000 (25%)]	Loss: 0.422329
    Train Epoch: 11 [15360/60000 (26%)]	Loss: 0.415076
    Train Epoch: 11 [16000/60000 (27%)]	Loss: 0.595345
    Train Epoch: 11 [16640/60000 (28%)]	Loss: 0.246912
    Train Epoch: 11 [17280/60000 (29%)]	Loss: 0.261347
    Train Epoch: 11 [17920/60000 (30%)]	Loss: 0.420687
    Train Epoch: 11 [18560/60000 (31%)]	Loss: 0.309478
    Train Epoch: 11 [19200/60000 (32%)]	Loss: 0.351695
    Train Epoch: 11 [19840/60000 (33%)]	Loss: 0.521406
    Train Epoch: 11 [20480/60000 (34%)]	Loss: 0.290906
    Train Epoch: 11 [21120/60000 (35%)]	Loss: 0.364633
    Train Epoch: 11 [21760/60000 (36%)]	Loss: 0.324597
    Train Epoch: 11 [22400/60000 (37%)]	Loss: 0.504305
    Train Epoch: 11 [23040/60000 (38%)]	Loss: 0.565828
    Train Epoch: 11 [23680/60000 (39%)]	Loss: 0.530418
    Train Epoch: 11 [24320/60000 (41%)]	Loss: 0.394785
    Train Epoch: 11 [24960/60000 (42%)]	Loss: 0.360259
    Train Epoch: 11 [25600/60000 (43%)]	Loss: 0.332049
    Train Epoch: 11 [26240/60000 (44%)]	Loss: 0.277467
    Train Epoch: 11 [26880/60000 (45%)]	Loss: 0.392917
    Train Epoch: 11 [27520/60000 (46%)]	Loss: 0.343030
    Train Epoch: 11 [28160/60000 (47%)]	Loss: 0.575351
    Train Epoch: 11 [28800/60000 (48%)]	Loss: 0.234557
    Train Epoch: 11 [29440/60000 (49%)]	Loss: 0.345107
    Train Epoch: 11 [30080/60000 (50%)]	Loss: 0.250498
    Train Epoch: 11 [30720/60000 (51%)]	Loss: 0.252943
    Train Epoch: 11 [31360/60000 (52%)]	Loss: 0.339441
    Train Epoch: 11 [32000/60000 (53%)]	Loss: 0.419630
    Train Epoch: 11 [32640/60000 (54%)]	Loss: 0.299459
    Train Epoch: 11 [33280/60000 (55%)]	Loss: 0.496848
    Train Epoch: 11 [33920/60000 (57%)]	Loss: 0.298093
    Train Epoch: 11 [34560/60000 (58%)]	Loss: 0.502162
    Train Epoch: 11 [35200/60000 (59%)]	Loss: 0.255059
    Train Epoch: 11 [35840/60000 (60%)]	Loss: 0.411274
    Train Epoch: 11 [36480/60000 (61%)]	Loss: 0.523598
    Train Epoch: 11 [37120/60000 (62%)]	Loss: 0.413543
    Train Epoch: 11 [37760/60000 (63%)]	Loss: 0.416163
    Train Epoch: 11 [38400/60000 (64%)]	Loss: 0.369535
    Train Epoch: 11 [39040/60000 (65%)]	Loss: 0.611558
    Train Epoch: 11 [39680/60000 (66%)]	Loss: 0.304744
    Train Epoch: 11 [40320/60000 (67%)]	Loss: 0.430891
    Train Epoch: 11 [40960/60000 (68%)]	Loss: 0.405095
    Train Epoch: 11 [41600/60000 (69%)]	Loss: 0.459111
    Train Epoch: 11 [42240/60000 (70%)]	Loss: 0.305776
    Train Epoch: 11 [42880/60000 (71%)]	Loss: 0.383718
    Train Epoch: 11 [43520/60000 (72%)]	Loss: 0.357237
    Train Epoch: 11 [44160/60000 (74%)]	Loss: 0.882389
    Train Epoch: 11 [44800/60000 (75%)]	Loss: 0.515517
    Train Epoch: 11 [45440/60000 (76%)]	Loss: 0.431814
    Train Epoch: 11 [46080/60000 (77%)]	Loss: 0.502057
    Train Epoch: 11 [46720/60000 (78%)]	Loss: 0.363643
    Train Epoch: 11 [47360/60000 (79%)]	Loss: 0.300866
    Train Epoch: 11 [48000/60000 (80%)]	Loss: 0.379479
    Train Epoch: 11 [48640/60000 (81%)]	Loss: 0.409872
    Train Epoch: 11 [49280/60000 (82%)]	Loss: 0.459707
    Train Epoch: 11 [49920/60000 (83%)]	Loss: 0.407087
    Train Epoch: 11 [50560/60000 (84%)]	Loss: 0.442198
    Train Epoch: 11 [51200/60000 (85%)]	Loss: 0.360245
    Train Epoch: 11 [51840/60000 (86%)]	Loss: 0.391902
    Train Epoch: 11 [52480/60000 (87%)]	Loss: 0.690279
    Train Epoch: 11 [53120/60000 (88%)]	Loss: 0.578411
    Train Epoch: 11 [53760/60000 (90%)]	Loss: 0.317039
    Train Epoch: 11 [54400/60000 (91%)]	Loss: 0.361648
    Train Epoch: 11 [55040/60000 (92%)]	Loss: 0.256818
    Train Epoch: 11 [55680/60000 (93%)]	Loss: 0.305927
    Train Epoch: 11 [56320/60000 (94%)]	Loss: 0.334766
    Train Epoch: 11 [56960/60000 (95%)]	Loss: 0.393670
    Train Epoch: 11 [57600/60000 (96%)]	Loss: 0.357648
    Train Epoch: 11 [58240/60000 (97%)]	Loss: 0.281212
    Train Epoch: 11 [58880/60000 (98%)]	Loss: 0.324076
    Train Epoch: 11 [59520/60000 (99%)]	Loss: 0.372610
    
    Test set: Average loss: 0.2098, Accuracy: 9373/10000 (94%)
    
    Train Epoch: 12 [0/60000 (0%)]	Loss: 0.392381
    Train Epoch: 12 [640/60000 (1%)]	Loss: 0.296244
    Train Epoch: 12 [1280/60000 (2%)]	Loss: 0.375838
    Train Epoch: 12 [1920/60000 (3%)]	Loss: 0.511141
    Train Epoch: 12 [2560/60000 (4%)]	Loss: 0.328571
    Train Epoch: 12 [3200/60000 (5%)]	Loss: 0.407022
    Train Epoch: 12 [3840/60000 (6%)]	Loss: 0.298561
    Train Epoch: 12 [4480/60000 (7%)]	Loss: 0.294833
    Train Epoch: 12 [5120/60000 (9%)]	Loss: 0.459635
    Train Epoch: 12 [5760/60000 (10%)]	Loss: 0.427801
    Train Epoch: 12 [6400/60000 (11%)]	Loss: 0.315486
    Train Epoch: 12 [7040/60000 (12%)]	Loss: 0.369394
    Train Epoch: 12 [7680/60000 (13%)]	Loss: 0.383768
    Train Epoch: 12 [8320/60000 (14%)]	Loss: 0.360965
    Train Epoch: 12 [8960/60000 (15%)]	Loss: 0.565722
    Train Epoch: 12 [9600/60000 (16%)]	Loss: 0.339543
    Train Epoch: 12 [10240/60000 (17%)]	Loss: 0.318308
    Train Epoch: 12 [10880/60000 (18%)]	Loss: 0.354275
    Train Epoch: 12 [11520/60000 (19%)]	Loss: 0.729154
    Train Epoch: 12 [12160/60000 (20%)]	Loss: 0.637020
    Train Epoch: 12 [12800/60000 (21%)]	Loss: 0.311871
    Train Epoch: 12 [13440/60000 (22%)]	Loss: 0.475887
    Train Epoch: 12 [14080/60000 (23%)]	Loss: 0.593350
    Train Epoch: 12 [14720/60000 (25%)]	Loss: 0.401409
    Train Epoch: 12 [15360/60000 (26%)]	Loss: 0.340033
    Train Epoch: 12 [16000/60000 (27%)]	Loss: 0.268460
    Train Epoch: 12 [16640/60000 (28%)]	Loss: 0.246902
    Train Epoch: 12 [17280/60000 (29%)]	Loss: 0.220537
    Train Epoch: 12 [17920/60000 (30%)]	Loss: 0.343910
    Train Epoch: 12 [18560/60000 (31%)]	Loss: 0.404446
    Train Epoch: 12 [19200/60000 (32%)]	Loss: 0.390659
    Train Epoch: 12 [19840/60000 (33%)]	Loss: 0.428503
    Train Epoch: 12 [20480/60000 (34%)]	Loss: 0.349071
    Train Epoch: 12 [21120/60000 (35%)]	Loss: 0.486959
    Train Epoch: 12 [21760/60000 (36%)]	Loss: 0.328149
    Train Epoch: 12 [22400/60000 (37%)]	Loss: 0.516612
    Train Epoch: 12 [23040/60000 (38%)]	Loss: 0.457053
    Train Epoch: 12 [23680/60000 (39%)]	Loss: 0.608891
    Train Epoch: 12 [24320/60000 (41%)]	Loss: 0.689961
    Train Epoch: 12 [24960/60000 (42%)]	Loss: 0.294651
    Train Epoch: 12 [25600/60000 (43%)]	Loss: 0.393591
    Train Epoch: 12 [26240/60000 (44%)]	Loss: 0.338528
    Train Epoch: 12 [26880/60000 (45%)]	Loss: 0.577185
    Train Epoch: 12 [27520/60000 (46%)]	Loss: 0.353298
    Train Epoch: 12 [28160/60000 (47%)]	Loss: 0.622561
    Train Epoch: 12 [28800/60000 (48%)]	Loss: 0.282284
    Train Epoch: 12 [29440/60000 (49%)]	Loss: 0.313890
    Train Epoch: 12 [30080/60000 (50%)]	Loss: 0.351842
    Train Epoch: 12 [30720/60000 (51%)]	Loss: 0.396683
    Train Epoch: 12 [31360/60000 (52%)]	Loss: 0.525928
    Train Epoch: 12 [32000/60000 (53%)]	Loss: 0.234339
    Train Epoch: 12 [32640/60000 (54%)]	Loss: 0.462475
    Train Epoch: 12 [33280/60000 (55%)]	Loss: 0.566767
    Train Epoch: 12 [33920/60000 (57%)]	Loss: 0.384068
    Train Epoch: 12 [34560/60000 (58%)]	Loss: 0.281656
    Train Epoch: 12 [35200/60000 (59%)]	Loss: 0.392156
    Train Epoch: 12 [35840/60000 (60%)]	Loss: 0.567646
    Train Epoch: 12 [36480/60000 (61%)]	Loss: 0.294172
    Train Epoch: 12 [37120/60000 (62%)]	Loss: 0.395887
    Train Epoch: 12 [37760/60000 (63%)]	Loss: 0.241547
    Train Epoch: 12 [38400/60000 (64%)]	Loss: 0.475505
    Train Epoch: 12 [39040/60000 (65%)]	Loss: 0.444348
    Train Epoch: 12 [39680/60000 (66%)]	Loss: 0.590313
    Train Epoch: 12 [40320/60000 (67%)]	Loss: 0.380521
    Train Epoch: 12 [40960/60000 (68%)]	Loss: 0.319756
    Train Epoch: 12 [41600/60000 (69%)]	Loss: 0.419879
    Train Epoch: 12 [42240/60000 (70%)]	Loss: 0.384562
    Train Epoch: 12 [42880/60000 (71%)]	Loss: 0.234591
    Train Epoch: 12 [43520/60000 (72%)]	Loss: 0.330877
    Train Epoch: 12 [44160/60000 (74%)]	Loss: 0.697167
    Train Epoch: 12 [44800/60000 (75%)]	Loss: 0.272816
    Train Epoch: 12 [45440/60000 (76%)]	Loss: 0.415027
    Train Epoch: 12 [46080/60000 (77%)]	Loss: 0.403599
    Train Epoch: 12 [46720/60000 (78%)]	Loss: 0.350379
    Train Epoch: 12 [47360/60000 (79%)]	Loss: 0.210332
    Train Epoch: 12 [48000/60000 (80%)]	Loss: 0.350990
    Train Epoch: 12 [48640/60000 (81%)]	Loss: 0.421243
    Train Epoch: 12 [49280/60000 (82%)]	Loss: 0.257715
    Train Epoch: 12 [49920/60000 (83%)]	Loss: 0.430463
    Train Epoch: 12 [50560/60000 (84%)]	Loss: 0.436658
    Train Epoch: 12 [51200/60000 (85%)]	Loss: 0.385483
    Train Epoch: 12 [51840/60000 (86%)]	Loss: 0.449448
    Train Epoch: 12 [52480/60000 (87%)]	Loss: 0.369401
    Train Epoch: 12 [53120/60000 (88%)]	Loss: 0.380906
    Train Epoch: 12 [53760/60000 (90%)]	Loss: 0.391110
    Train Epoch: 12 [54400/60000 (91%)]	Loss: 0.381157
    Train Epoch: 12 [55040/60000 (92%)]	Loss: 0.317574
    Train Epoch: 12 [55680/60000 (93%)]	Loss: 0.616172
    Train Epoch: 12 [56320/60000 (94%)]	Loss: 0.333590
    Train Epoch: 12 [56960/60000 (95%)]	Loss: 0.460308
    Train Epoch: 12 [57600/60000 (96%)]	Loss: 0.586635
    Train Epoch: 12 [58240/60000 (97%)]	Loss: 0.323481
    Train Epoch: 12 [58880/60000 (98%)]	Loss: 0.410162
    Train Epoch: 12 [59520/60000 (99%)]	Loss: 0.475990
    
    Test set: Average loss: 0.2096, Accuracy: 9381/10000 (94%)
    
    Train Epoch: 13 [0/60000 (0%)]	Loss: 0.555876
    Train Epoch: 13 [640/60000 (1%)]	Loss: 0.298020
    Train Epoch: 13 [1280/60000 (2%)]	Loss: 0.341556
    Train Epoch: 13 [1920/60000 (3%)]	Loss: 0.387244
    Train Epoch: 13 [2560/60000 (4%)]	Loss: 0.299948
    Train Epoch: 13 [3200/60000 (5%)]	Loss: 0.352978
    Train Epoch: 13 [3840/60000 (6%)]	Loss: 0.445687
    Train Epoch: 13 [4480/60000 (7%)]	Loss: 0.223049
    Train Epoch: 13 [5120/60000 (9%)]	Loss: 0.494324
    Train Epoch: 13 [5760/60000 (10%)]	Loss: 0.749437
    Train Epoch: 13 [6400/60000 (11%)]	Loss: 0.404310
    Train Epoch: 13 [7040/60000 (12%)]	Loss: 0.337297
    Train Epoch: 13 [7680/60000 (13%)]	Loss: 0.434967
    Train Epoch: 13 [8320/60000 (14%)]	Loss: 0.401748
    Train Epoch: 13 [8960/60000 (15%)]	Loss: 0.340427
    Train Epoch: 13 [9600/60000 (16%)]	Loss: 0.614933
    Train Epoch: 13 [10240/60000 (17%)]	Loss: 0.428032
    Train Epoch: 13 [10880/60000 (18%)]	Loss: 0.520478
    Train Epoch: 13 [11520/60000 (19%)]	Loss: 0.343639
    Train Epoch: 13 [12160/60000 (20%)]	Loss: 0.282134
    Train Epoch: 13 [12800/60000 (21%)]	Loss: 0.236920
    Train Epoch: 13 [13440/60000 (22%)]	Loss: 0.331308
    Train Epoch: 13 [14080/60000 (23%)]	Loss: 0.342169
    Train Epoch: 13 [14720/60000 (25%)]	Loss: 0.494080
    Train Epoch: 13 [15360/60000 (26%)]	Loss: 0.566829
    Train Epoch: 13 [16000/60000 (27%)]	Loss: 0.515479
    Train Epoch: 13 [16640/60000 (28%)]	Loss: 0.546352
    Train Epoch: 13 [17280/60000 (29%)]	Loss: 0.462010
    Train Epoch: 13 [17920/60000 (30%)]	Loss: 0.547893
    Train Epoch: 13 [18560/60000 (31%)]	Loss: 0.519924
    Train Epoch: 13 [19200/60000 (32%)]	Loss: 0.445337
    Train Epoch: 13 [19840/60000 (33%)]	Loss: 0.254473
    Train Epoch: 13 [20480/60000 (34%)]	Loss: 0.351019
    Train Epoch: 13 [21120/60000 (35%)]	Loss: 0.388970
    Train Epoch: 13 [21760/60000 (36%)]	Loss: 0.285459
    Train Epoch: 13 [22400/60000 (37%)]	Loss: 0.308739
    Train Epoch: 13 [23040/60000 (38%)]	Loss: 0.501287
    Train Epoch: 13 [23680/60000 (39%)]	Loss: 0.392744
    Train Epoch: 13 [24320/60000 (41%)]	Loss: 0.490547
    Train Epoch: 13 [24960/60000 (42%)]	Loss: 0.407411
    Train Epoch: 13 [25600/60000 (43%)]	Loss: 0.557519
    Train Epoch: 13 [26240/60000 (44%)]	Loss: 0.407774
    Train Epoch: 13 [26880/60000 (45%)]	Loss: 0.313497
    Train Epoch: 13 [27520/60000 (46%)]	Loss: 0.470231
    Train Epoch: 13 [28160/60000 (47%)]	Loss: 0.457753
    Train Epoch: 13 [28800/60000 (48%)]	Loss: 0.314194
    Train Epoch: 13 [29440/60000 (49%)]	Loss: 0.395972
    Train Epoch: 13 [30080/60000 (50%)]	Loss: 0.575824
    Train Epoch: 13 [30720/60000 (51%)]	Loss: 0.275038
    Train Epoch: 13 [31360/60000 (52%)]	Loss: 0.376275
    Train Epoch: 13 [32000/60000 (53%)]	Loss: 0.517350
    Train Epoch: 13 [32640/60000 (54%)]	Loss: 0.386347
    Train Epoch: 13 [33280/60000 (55%)]	Loss: 0.315577
    Train Epoch: 13 [33920/60000 (57%)]	Loss: 0.385711
    Train Epoch: 13 [34560/60000 (58%)]	Loss: 0.308082
    Train Epoch: 13 [35200/60000 (59%)]	Loss: 0.412021
    Train Epoch: 13 [35840/60000 (60%)]	Loss: 0.630597
    Train Epoch: 13 [36480/60000 (61%)]	Loss: 0.530441
    Train Epoch: 13 [37120/60000 (62%)]	Loss: 0.324686
    Train Epoch: 13 [37760/60000 (63%)]	Loss: 0.334050
    Train Epoch: 13 [38400/60000 (64%)]	Loss: 0.539302
    Train Epoch: 13 [39040/60000 (65%)]	Loss: 0.168276
    Train Epoch: 13 [39680/60000 (66%)]	Loss: 0.218964
    Train Epoch: 13 [40320/60000 (67%)]	Loss: 0.526193
    Train Epoch: 13 [40960/60000 (68%)]	Loss: 0.554866
    Train Epoch: 13 [41600/60000 (69%)]	Loss: 0.519486
    Train Epoch: 13 [42240/60000 (70%)]	Loss: 0.659215
    Train Epoch: 13 [42880/60000 (71%)]	Loss: 0.347684
    Train Epoch: 13 [43520/60000 (72%)]	Loss: 0.218575
    Train Epoch: 13 [44160/60000 (74%)]	Loss: 0.498827
    Train Epoch: 13 [44800/60000 (75%)]	Loss: 0.428912
    Train Epoch: 13 [45440/60000 (76%)]	Loss: 0.554431
    Train Epoch: 13 [46080/60000 (77%)]	Loss: 0.334991
    Train Epoch: 13 [46720/60000 (78%)]	Loss: 0.312058
    Train Epoch: 13 [47360/60000 (79%)]	Loss: 0.393212
    Train Epoch: 13 [48000/60000 (80%)]	Loss: 0.328563
    Train Epoch: 13 [48640/60000 (81%)]	Loss: 0.441795
    Train Epoch: 13 [49280/60000 (82%)]	Loss: 0.487448
    Train Epoch: 13 [49920/60000 (83%)]	Loss: 0.393158
    Train Epoch: 13 [50560/60000 (84%)]	Loss: 0.413586
    Train Epoch: 13 [51200/60000 (85%)]	Loss: 0.331015
    Train Epoch: 13 [51840/60000 (86%)]	Loss: 0.293184
    Train Epoch: 13 [52480/60000 (87%)]	Loss: 0.448311
    Train Epoch: 13 [53120/60000 (88%)]	Loss: 0.275574
    Train Epoch: 13 [53760/60000 (90%)]	Loss: 0.361041
    Train Epoch: 13 [54400/60000 (91%)]	Loss: 0.270119
    Train Epoch: 13 [55040/60000 (92%)]	Loss: 0.339491
    Train Epoch: 13 [55680/60000 (93%)]	Loss: 0.460334
    Train Epoch: 13 [56320/60000 (94%)]	Loss: 0.355198
    Train Epoch: 13 [56960/60000 (95%)]	Loss: 0.324064
    Train Epoch: 13 [57600/60000 (96%)]	Loss: 0.461057
    Train Epoch: 13 [58240/60000 (97%)]	Loss: 0.520947
    Train Epoch: 13 [58880/60000 (98%)]	Loss: 0.555590
    Train Epoch: 13 [59520/60000 (99%)]	Loss: 0.347576
    
    Test set: Average loss: 0.2075, Accuracy: 9385/10000 (94%)
    
    Train Epoch: 14 [0/60000 (0%)]	Loss: 0.319042
    Train Epoch: 14 [640/60000 (1%)]	Loss: 0.286378
    Train Epoch: 14 [1280/60000 (2%)]	Loss: 0.475702
    Train Epoch: 14 [1920/60000 (3%)]	Loss: 0.460729
    Train Epoch: 14 [2560/60000 (4%)]	Loss: 0.227350
    Train Epoch: 14 [3200/60000 (5%)]	Loss: 0.430530
    Train Epoch: 14 [3840/60000 (6%)]	Loss: 0.370811
    Train Epoch: 14 [4480/60000 (7%)]	Loss: 0.292919
    Train Epoch: 14 [5120/60000 (9%)]	Loss: 0.462068
    Train Epoch: 14 [5760/60000 (10%)]	Loss: 0.240440
    Train Epoch: 14 [6400/60000 (11%)]	Loss: 0.330162
    Train Epoch: 14 [7040/60000 (12%)]	Loss: 0.385991
    Train Epoch: 14 [7680/60000 (13%)]	Loss: 0.260772
    Train Epoch: 14 [8320/60000 (14%)]	Loss: 0.431668
    Train Epoch: 14 [8960/60000 (15%)]	Loss: 0.391844
    Train Epoch: 14 [9600/60000 (16%)]	Loss: 0.607404
    Train Epoch: 14 [10240/60000 (17%)]	Loss: 0.517053
    Train Epoch: 14 [10880/60000 (18%)]	Loss: 0.460433
    Train Epoch: 14 [11520/60000 (19%)]	Loss: 0.294837
    Train Epoch: 14 [12160/60000 (20%)]	Loss: 0.376116
    Train Epoch: 14 [12800/60000 (21%)]	Loss: 0.302840
    Train Epoch: 14 [13440/60000 (22%)]	Loss: 0.423696
    Train Epoch: 14 [14080/60000 (23%)]	Loss: 0.396551
    Train Epoch: 14 [14720/60000 (25%)]	Loss: 0.315363
    Train Epoch: 14 [15360/60000 (26%)]	Loss: 0.452954
    Train Epoch: 14 [16000/60000 (27%)]	Loss: 0.492528
    Train Epoch: 14 [16640/60000 (28%)]	Loss: 0.209144
    Train Epoch: 14 [17280/60000 (29%)]	Loss: 0.361104
    Train Epoch: 14 [17920/60000 (30%)]	Loss: 0.337909
    Train Epoch: 14 [18560/60000 (31%)]	Loss: 0.235292
    Train Epoch: 14 [19200/60000 (32%)]	Loss: 0.378781
    Train Epoch: 14 [19840/60000 (33%)]	Loss: 0.698395
    Train Epoch: 14 [20480/60000 (34%)]	Loss: 0.654676
    Train Epoch: 14 [21120/60000 (35%)]	Loss: 0.261703
    Train Epoch: 14 [21760/60000 (36%)]	Loss: 0.491567
    Train Epoch: 14 [22400/60000 (37%)]	Loss: 0.460270
    Train Epoch: 14 [23040/60000 (38%)]	Loss: 0.663427
    Train Epoch: 14 [23680/60000 (39%)]	Loss: 0.488279
    Train Epoch: 14 [24320/60000 (41%)]	Loss: 0.412345
    Train Epoch: 14 [24960/60000 (42%)]	Loss: 0.330990
    Train Epoch: 14 [25600/60000 (43%)]	Loss: 0.319391
    Train Epoch: 14 [26240/60000 (44%)]	Loss: 0.364210
    Train Epoch: 14 [26880/60000 (45%)]	Loss: 0.279273
    Train Epoch: 14 [27520/60000 (46%)]	Loss: 0.176225
    Train Epoch: 14 [28160/60000 (47%)]	Loss: 0.297678
    Train Epoch: 14 [28800/60000 (48%)]	Loss: 0.378201
    Train Epoch: 14 [29440/60000 (49%)]	Loss: 0.232202
    Train Epoch: 14 [30080/60000 (50%)]	Loss: 0.525252
    Train Epoch: 14 [30720/60000 (51%)]	Loss: 0.368206
    Train Epoch: 14 [31360/60000 (52%)]	Loss: 0.304667
    Train Epoch: 14 [32000/60000 (53%)]	Loss: 0.358428
    Train Epoch: 14 [32640/60000 (54%)]	Loss: 0.427945
    Train Epoch: 14 [33280/60000 (55%)]	Loss: 0.488429
    Train Epoch: 14 [33920/60000 (57%)]	Loss: 0.526154
    Train Epoch: 14 [34560/60000 (58%)]	Loss: 0.725787
    Train Epoch: 14 [35200/60000 (59%)]	Loss: 0.599196
    Train Epoch: 14 [35840/60000 (60%)]	Loss: 0.327683
    Train Epoch: 14 [36480/60000 (61%)]	Loss: 0.611174
    Train Epoch: 14 [37120/60000 (62%)]	Loss: 0.429956
    Train Epoch: 14 [37760/60000 (63%)]	Loss: 0.384994
    Train Epoch: 14 [38400/60000 (64%)]	Loss: 0.302766
    Train Epoch: 14 [39040/60000 (65%)]	Loss: 0.637129
    Train Epoch: 14 [39680/60000 (66%)]	Loss: 0.300277
    Train Epoch: 14 [40320/60000 (67%)]	Loss: 0.605256
    Train Epoch: 14 [40960/60000 (68%)]	Loss: 0.563442
    Train Epoch: 14 [41600/60000 (69%)]	Loss: 0.315805
    Train Epoch: 14 [42240/60000 (70%)]	Loss: 0.498134
    Train Epoch: 14 [42880/60000 (71%)]	Loss: 0.304480
    Train Epoch: 14 [43520/60000 (72%)]	Loss: 0.358127
    Train Epoch: 14 [44160/60000 (74%)]	Loss: 0.354775
    Train Epoch: 14 [44800/60000 (75%)]	Loss: 0.349251
    Train Epoch: 14 [45440/60000 (76%)]	Loss: 0.363537
    Train Epoch: 14 [46080/60000 (77%)]	Loss: 0.397053
    Train Epoch: 14 [46720/60000 (78%)]	Loss: 0.569868
    Train Epoch: 14 [47360/60000 (79%)]	Loss: 0.387928
    Train Epoch: 14 [48000/60000 (80%)]	Loss: 0.348417
    Train Epoch: 14 [48640/60000 (81%)]	Loss: 0.377063
    Train Epoch: 14 [49280/60000 (82%)]	Loss: 0.260186
    Train Epoch: 14 [49920/60000 (83%)]	Loss: 0.297211
    Train Epoch: 14 [50560/60000 (84%)]	Loss: 0.702463
    Train Epoch: 14 [51200/60000 (85%)]	Loss: 0.302332
    Train Epoch: 14 [51840/60000 (86%)]	Loss: 0.526482
    Train Epoch: 14 [52480/60000 (87%)]	Loss: 0.400840
    Train Epoch: 14 [53120/60000 (88%)]	Loss: 0.501183
    Train Epoch: 14 [53760/60000 (90%)]	Loss: 0.302832
    Train Epoch: 14 [54400/60000 (91%)]	Loss: 0.351779
    Train Epoch: 14 [55040/60000 (92%)]	Loss: 0.406741
    Train Epoch: 14 [55680/60000 (93%)]	Loss: 0.455118
    Train Epoch: 14 [56320/60000 (94%)]	Loss: 0.324182
    Train Epoch: 14 [56960/60000 (95%)]	Loss: 0.380480
    Train Epoch: 14 [57600/60000 (96%)]	Loss: 0.729591
    Train Epoch: 14 [58240/60000 (97%)]	Loss: 0.435104
    Train Epoch: 14 [58880/60000 (98%)]	Loss: 0.378653
    Train Epoch: 14 [59520/60000 (99%)]	Loss: 0.280005
    
    Test set: Average loss: 0.2066, Accuracy: 9386/10000 (94%)
    


The model has successfully trained and downloaded


```bash
%%bash
ls results/combined_results/outputs/
```

    mnist_rnn.pt

