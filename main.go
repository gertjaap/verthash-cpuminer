package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/gertjaap/verthash-cpuminer/composer"
	"github.com/gertjaap/verthash-cpuminer/config"
	"github.com/gertjaap/verthash-cpuminer/verthash"
)

func main() {

	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Printf("Could not find config. Make sure it exists in the executable path (verthash-cpuminer-config.json)")
		os.Exit(-1)
	}

	rpc, err := rpcclient.New(&rpcclient.ConnConfig{
		HTTPPostMode: true,
		Host:         cfg.RpcHost,
		User:         cfg.RpcUser,
		Pass:         cfg.RpcPassword,
		DisableTLS:   true,
	}, nil)

	if err != nil {
		panic(err)
	}

	vh, err := verthash.NewVerthasher(cfg.VerthashDatFile)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	var hashes uint64
	go func() {
		for {
			time.Sleep(time.Second)
			elapsed := time.Now().Sub(start).Seconds()
			h := atomic.SwapUint64(&hashes, 0)
			start = time.Now()
			hps := float64(h) / elapsed
			fmt.Printf("\rHashrate: %0.2f kH/s", hps/1000)
		}
	}()

	for {
		terminate := make(chan bool, runtime.NumCPU())

		blk, height, err := composer.ComposeBlock(rpc, cfg)
		if err != nil {
			panic(err)
		}

		// Monitor for changed height if a block is found. Terminate
		// workers and restart the loop.
		go func(height int) {
			for {
				time.Sleep(time.Second * 5)
				_, newHeight, err := composer.ComposeBlock(rpc, cfg)
				if err == nil && newHeight != height {
					for i := 0; i < runtime.NumCPU(); i++ {
						terminate <- true
					}
				}
			}
		}(height)

		fmt.Printf("\n\n-- Starting work on block %d --\n\n", height)

		var buf bytes.Buffer
		err = blk.Header.Serialize(&buf)
		if err != nil {
			panic(err)
		}

		headerBytes := buf.Bytes()
		target := blockchain.CompactToBig(blk.Header.Bits)
		var wg sync.WaitGroup

		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func(worker int) {
				myCopy := make([]byte, 80)
				copy(myCopy, headerBytes)
				t := time.Now().Unix()
				mining := true
				for mining {

					if t == time.Now().Unix() {
						t++
					} else {
						t = time.Now().Unix()
					}
					binary.LittleEndian.PutUint32(myCopy[68:], uint32(t))
					nonce := uint32(worker) * 100000000
					nonceLimit := uint32(worker+1) * 100000000
					for mining && int64(t+5) > time.Now().Unix() && nonce < nonceLimit {

						select {
						case <-terminate:
							mining = false
						default:
						}

						if !mining {
							break
						}

						binary.LittleEndian.PutUint32(myCopy[76:], uint32(nonce))
						h := vh.Hash(myCopy)
						ch, _ := chainhash.NewHash(h[:])
						bnHash := blockchain.HashToBig(ch)
						if bnHash.Cmp(target) <= 0 {
							// Found block!
							blk.Header.Timestamp = time.Unix(t, 0)
							blk.Header.Nonce = nonce

							var headerBuf bytes.Buffer
							blk.Header.Serialize(&headerBuf)

							var blockBuf bytes.Buffer
							blk.Serialize(&blockBuf)

							_, err := rpc.RawRequest("submitblock", []json.RawMessage{[]byte(fmt.Sprintf("\"%s\"", hex.EncodeToString(blockBuf.Bytes())))})
							if err != nil {
								fmt.Printf("\n\nBlock rejected: %s\n\n", err.Error())
							} else {
								fmt.Printf("\nBlock %d accepted!\n", height)
								for i := 0; i < runtime.NumCPU(); i++ {
									terminate <- true
								}
								mining = false
								break
							}
						}
						atomic.AddUint64(&hashes, 1)
						nonce++
					}

				}
				wg.Done()
			}(i)

		}

		wg.Wait()
		fmt.Printf("\n\n-- Work on block %d concluded. --\n\n", height)

	}

}
