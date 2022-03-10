sed -n '/38.7[2-4]..,-9.1[3-7]../p' temperature_sensor_data.csv  | wc -l
3871

Total Samples: 3000000
Total from Lisbon: 1000000
Total within 25 km of LISBON city center: 400771

wc -l temperature_sensor_data.csv 
3000001 temperature_sensor_data.csv

├─ 28.395 main  generate_csv.py:17
│  ├─ 13.607 savetxt  <__array_function__ internals>:177
│  │     [7 frames hidden]  <__array_function__ internals>, numpy...
│  │        13.607 savetxt  numpy/lib/npyio.py:1217
│  │        ├─ 12.669 [self]  
│  ├─ 5.686 ndarray.astype  <built-in>:0
│  │     [2 frames hidden]  <built-in>
│  ├─ 4.600 values  pandas/core/frame.py:10802
│  │     [15 frames hidden]  pandas, <built-in>
│  ├─ 1.506 __init__  pandas/core/frame.py:587
│  │     [62 frames hidden]  pandas, <__array_function__ internals...
│  ├─ 1.106 [self]  
│  ├─ 0.434 new_method  pandas/core/ops/common.py:55
│  │     [19 frames hidden]  pandas, <built-in>
│  ├─ 0.398 concatenate  <__array_function__ internals>:177
│  │     [3 frames hidden]  <__array_function__ internals>, <buil...
│  └─ 0.397 calc_temperature  generate_csv.py:70
└─ 0.289 [self]

# Instructions
../../bin/bacalhau devstack --dev
export file_path="./temperature_sensor_data.csv"
cid=$( IPFS_PATH=/tmp/bacalhau-ipfs1883232639 ipfs add -q $file_path )

./bin/bacalhau submit --cids=$cid --commands="sed -n '/38.7[2-4]..,-9.1[3-7]../p' /ipfs/$cid"