package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BlockHash     string             `bson:"block_hash" json:"block_hash"`
	Height        int64              `bson:"height" json:"height"`
	// Hex           string             `bson:"hex" json:"hex"`
	Hash          string             `bson:"hash" json:"hash"`
	VtxHash       string             `bson:"vtx_hash,omitempty" json:"vtx_hash"`
	Size          int32              `bson:"size" json:"size"`
	Version       uint32             `bson:"version" json:"version"`
	LockTime      uint32             `bson:"lock_time" json:"lock_time"`
	Confirmations int64              `bson:"confirmations" json:"confirmations"`
	Time          int64              `bson:"time" json:"time"`
	Fee           []Fee              `bson:"fee" json:"fee"`
	GasLimit      int64              `bson:"gas_limit" json:"gas_limit"`
	// Assets        []string           `bson:"assets" json:"assets"`
	Vin  []Vin  `bson:"vin" json:"vin"`
	Vout []Vout `bson:"vout" json:"vout"`
}

type Fee struct {
	Value int64  `bson:"value" json:"value"`
	Asset string `bson:"asset" json:"asset"`
}

type Vin struct {
	// ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// TxHash string             `bson:"tx_hash" json:"tx_hash"`
	// Height int64              `bson:"height" json:"height"`
	// Time   int64              `bson:"time" json:"time"`
	// CoinBase Script
	CoinBase  string     `bson:"coin_base,omitempty" json:"coin_base,omitempty"`
	Sequence  uint32     `bson:"sequence,omitempty" json:"sequence,omitempty"`
	OutTxHash string     `bson:"out_tx_hash,omitempty" json:"out_tx_hash,omitempty"`
	VOut      *uint32    `bson:"v_out,omitempty" json:"v_out,omitempty"`
	ScriptSig *ScriptSig `bson:"script_sig,omitempty" json:"script_sig,omitempty"`
	PrevOut   *PrevOut   `bson:"prev_out,omitempty" json:"prev_out,omitempty"`
}

type ScriptSig struct {
	// Asm string `bson:"asm" json:"asm"`
	Hex string `bson:"hex" json:"hex"`
}

type PrevOut struct {
	Addresses []string `bson:"addresses" json:"addresses"`
	Value     int64    `bson:"value" json:"value"`
	Asset     string   `bson:"asset" json:"asset"`
	// Data      string   `bson:"data,omitempty" json:"data,omitempty"`
}

type Vout struct {
	// ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// TxHash       string             `bson:"tx_hash" json:"tx_hash"`
	// Height       int64              `bson:"height" json:"height"`
	// Time         int64              `bson:"time" json:"time"`
	Value        int64              `bson:"value" json:"value"`
	N            uint32             `bson:"n" json:"n"`
	ScriptPubKey ScriptPubKey       `bson:"script_pub_key" json:"script_pub_key"`
	// Data         string             `bson:"data,omitempty" json:"data,omitempty"`
	Asset string `bson:"asset" json:"asset"`
}

type ScriptPubKey struct {
	Asm string `bson:"asm" json:"asm"`
	// Hex       string   `bson:"hex" json:"hex"`
	ReqSigs   int32    `bson:"req_sigs,omitempty" json:"req_sigs,omitempty"`
	Type      string   `bson:"type" json:"type"`
	Addresses []string `bson:"addresses,omitempty" json:"addresses,omitempty"`
}

type AddressAssetBalance struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Height  int64              `bson:"height" json:"height"`
	Time    int64              `bson:"time" json:"time"`
	Address string             `bson:"address" json:"address"`
	Asset   string             `bson:"asset" json:"asset"`
	Balance int64              `bson:"balance" json:"balance"`
}
