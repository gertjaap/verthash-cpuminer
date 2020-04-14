# Simple Verthash CPU miner

For the time being, this miner connects to Vertcoin Core over RPC and uses GetBlockTemplate to compose a new block. Currently it does not mine any transactions, though.

Needed to test block hash verification in new Verthash prototypes.

## How to build

```
go get github.com/gertjaap/verthash-cpuminer
cd $GOPATH/src/github.com/gertjaap/verthash-cpuminer
go get ./...
go build
```

## How to run

Ensure all parameters are properly set in `verthash-cpuminer-config.json`, like the RPC host, credentials, and the address to pay out to.

Ensure Vertcoin core is running and has the verthash datafile created, and that you're pointing to the correct file from the config

Then start the miner with 

```
cd $GOPATH/src/github.com/gertjaap/verthash-cpuminer
./verthash-cpuminer
```
