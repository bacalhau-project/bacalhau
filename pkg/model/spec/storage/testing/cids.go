package storagetesting

import "github.com/ipfs/go-cid"

var TestCID1 cid.Cid
var TestCID2 cid.Cid

func init() {
	// A real CID that can be resolved: https://docs.ipfs.tech/how-to/command-line-quick-start/#initialize-the-repository
	c, err := cid.Decode("QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG")
	if err != nil {
		panic(err)
	}
	TestCID1 = c

	// a real CID that can be resolved: https://docs.ipfs.tech/how-to/command-line-quick-start/#take-your-node-online (spaceship)
	c, err = cid.Decode("QmSgvgwxZGaBLqkGyWemEDqikCqU52XxsYLKtdy3vGZ8uq")
	if err != nil {
		panic(err)
	}
	TestCID2 = c
}
