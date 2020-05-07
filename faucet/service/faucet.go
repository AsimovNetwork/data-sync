package service

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/AsimovNetwork/asimov/asiutil"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/asimov/crypto"
	"github.com/AsimovNetwork/asimov/protos"
	"github.com/AsimovNetwork/asimov/txscript"
	"github.com/AsimovNetwork/data-sync/library/common"
)

type FaucetService struct {
}

type UTXO struct {
	Account       string `json:"account"`
	Address       string `json:"address"`
	Amount        int64  `json:"amount"`
	Assets        string `json:"assets"`
	Confirmations int64  `json:"confirmations"`
	ScriptPubKey  string `json:"scriptPubKey"`
	TxID          string `json:"txid"`
	Vout          uint32 `json:"vout"`
	Spendable     bool   `json:"spendable"`
}

/**
通过私钥生成地址
*/
func (faucetService FaucetService) GenerateAddressViaPrivate() (*asimovCommon.Address, *crypto.PrivateKey, error) {
	privateKey := common.Cfg.FaucetPrivateKey[2:]
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		common.Logger.Errorf("decode private key error. privateKey: %s, err: %s", privateKey, err)
		return nil, nil, err
	}
	curvePrivate, curvePublic := crypto.PrivKeyFromBytes(crypto.S256(), privateKeyBytes)
	pubKeyHash := asimovCommon.Hash160(curvePublic.SerializeCompressed())
	address, err := asimovCommon.NewAddressWithId(asimovCommon.PubKeyHashAddrID, pubKeyHash)
	if err != nil {
		common.Logger.Errorf("generate address error, err: %s", privateKey, err)
		return nil, nil, err
	}

	return address, curvePrivate, nil
}

func (faucetService FaucetService) GetUTXO(address string) ([]UTXO, error) {
	param := common.NewChainRequest("getUtxoByAddress", []interface{}{[]string{address}, "000000000000000000000000"})
	result, ok := common.Post(common.Cfg.BlockChainRpc, param)
	if !ok {
		return nil, errors.New("call block chain failed")
	}

	mapSlice := (result).([]interface{})
	utxoSlice := make([]UTXO, 0)
	for _, v := range mapSlice {
		utxo := &UTXO{}
		err := common.ToStruct(v, utxo)
		if err != nil {
			return nil, err
		}
		utxoSlice = append(utxoSlice, *utxo)
	}

	return utxoSlice, nil
}

func (faucetService FaucetService) GenerateTransaction(receiver string, amount int64, utxoSlice []UTXO, miner *asimovCommon.Address) (*protos.MsgTx, error) {
	redeemTx := protos.NewMsgTx(protos.TxVersion)
	redeemTx.TxContract.GasLimit = 21000
	fee := int64(210)
	var spendAmount int64 = 0
	for _, v := range utxoSlice {
		spendAmount += v.Amount

		// 构建输入
		utxoIDHash := asimovCommon.HexToHash(v.TxID)
		prevOut := protos.NewOutPoint(&utxoIDHash, v.Vout)
		txIn := protos.NewTxIn(prevOut, []byte{})
		redeemTx.AddTxIn(txIn)

		if spendAmount == amount {
			receiver := asimovCommon.HexToAddress(receiver[2:])
			receiverPkScript, err := txscript.PayToAddrScript(&receiver)
			if err != nil {
				common.Logger.Errorf("pay to addr script failed. address: %s, err: %s", receiver.String(), err)
				return nil, err
			}
			receiverTxOut := protos.NewTxOut(amount, receiverPkScript, asiutil.FlowCoinAsset)
			redeemTx.AddTxOut(receiverTxOut)
			break
		} else if spendAmount > amount {
			receiver := asimovCommon.HexToAddress(receiver[2:])
			receiverPkScript, err := txscript.PayToAddrScript(&receiver)
			if err != nil {
				common.Logger.Errorf("pay to addr script failed. address: %s, err: %s", receiver.String(), err)
				return nil, err
			}
			receiverTxOut := protos.NewTxOut(amount, receiverPkScript, asiutil.FlowCoinAsset)
			redeemTx.AddTxOut(receiverTxOut)

			// utxo多的钱返回给款工地址
			minerPkScript, err := txscript.PayToAddrScript(miner)
			if err != nil {
				common.Logger.Errorf("pay to addr script failed. address: %s，err: %s", miner.String(), err)
				return nil, err
			}
			minneTxOut := protos.NewTxOut(spendAmount-amount-fee, minerPkScript, asiutil.FlowCoinAsset)
			redeemTx.AddTxOut(minneTxOut)
			break
		}
	}
	return redeemTx, nil
}

func (faucetService FaucetService) SignTransaction(miner *asimovCommon.Address, redeemTx *protos.MsgTx, curvePrivate *crypto.PrivateKey) error {
	lookupKey := func(a asimovCommon.IAddress) (*crypto.PrivateKey, bool, error) {
		return curvePrivate, true, nil
	}
	minerPkScript, err := txscript.PayToAddrScript(miner)
	if err != nil {
		common.Logger.Errorf("pay to addr script failed. address: %s, err: %s", miner.String(), err)
		return err
	}
	signSlice := make([][]byte, 0)
	for i, _ := range redeemTx.TxIn {
		sigScript, err := txscript.SignTxOutput(
			redeemTx, i, minerPkScript, txscript.SigHashAll,
			txscript.KeyClosure(lookupKey), nil, nil)
		if err != nil {
			common.Logger.Errorf("sign tx in failed. err: %s", err)
			return err
		}
		signSlice = append(signSlice, sigScript)
	}
	for i, v := range redeemTx.TxIn {
		v.SignatureScript = signSlice[i]
	}
	return nil
}

func (faucetService FaucetService) SendTransaction(redeemTx *protos.MsgTx) (string, error) {
	maxProtocolVersion := uint32(70002)
	var buf bytes.Buffer
	if err := redeemTx.VVSEncode(&buf, maxProtocolVersion, protos.BaseEncoding); err != nil {
		common.Logger.Errorf("encode redeem tx failed. err: %s", err)
		return "", err
	}

	txHex := hex.EncodeToString(buf.Bytes())
	param := common.NewChainRequest("sendRawTransaction", []interface{}{txHex})
	result, ok := common.Post(common.Cfg.BlockChainRpc, param)
	if !ok {
		return "", errors.New("call block chain failed")
	}

	return result.(string), nil
}
