# NOTE: You need to escape $ in the string
go run . submit --cids=$cid --commands="awk -F',' '{x=38.7077507-\$3; y=-9.1365919-\$4; if(x^2+y^2<0.3^2) print}' /ipfs/$cid" --jsonrpc-port=43595


go run . --jsonrpc-port=39359 submit --cids=$cid --commands="awk -F, '{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.1^2) print}'"
