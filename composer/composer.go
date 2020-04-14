package composer

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/base58"
	"github.com/gertjaap/verthash-cpuminer/config"
)

var CoinbaseFlags = "/P2SH/Verthash/"

func ComposeBlock(rpc *rpcclient.Client, cfg config.MinerConfig) (*wire.MsgBlock, int, error) {

	var params = map[string]interface{}{}
	params["rules"] = []string{"segwit"}
	j, err := json.Marshal(params)
	if err != nil {
		return nil, -1, err
	}
	res, err := rpc.RawRequest("getblocktemplate", []json.RawMessage{j})
	if err != nil {
		return nil, -1, err
	}
	err = json.Unmarshal(res, &params)
	if err != nil {
		return nil, -1, err
	}

	prevHash, _ := chainhash.NewHashFromStr(params["previousblockhash"].(string))
	hexBits, _ := hex.DecodeString(params["bits"].(string))
	bits := binary.BigEndian.Uint32(hexBits)

	height := int64(params["height"].(float64))
	coinbaseScript, err := txscript.NewScriptBuilder().AddInt64(height).AddInt64(int64(0)).AddData([]byte(CoinbaseFlags)).Script()
	if err != nil {
		return nil, -1, err
	}

	scriptHash, _, err := base58.CheckDecode(cfg.PayRewardsTo)
	if err != nil {
		return nil, -1, fmt.Errorf("invalid_address")
	}
	if len(scriptHash) != 20 {
		return nil, -1, fmt.Errorf("invalid_address_length")
	}
	pkScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_HASH160).AddData(scriptHash).AddOp(txscript.OP_EQUAL).Script()
	if err != nil {
		return nil, -1, fmt.Errorf("script_failure")
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{}, wire.MaxPrevOutIndex),
		SignatureScript:  coinbaseScript,
		Sequence:         wire.MaxTxInSequenceNum,
	})
	tx.AddTxOut(&wire.TxOut{
		Value:    int64(params["coinbasevalue"].(float64)),
		PkScript: pkScript,
	})

	coinbaseHash := tx.TxHash()
	hdr := wire.NewBlockHeader(int32(params["version"].(float64)), prevHash, &coinbaseHash, bits, 0)

	blk := wire.NewMsgBlock(hdr)
	blk.Transactions = []*wire.MsgTx{tx}

	return blk, int(height), nil
}
