package service

import (
	"context"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/asimov/protos"
	"github.com/AsimovNetwork/asimov/rpcs/rpcjson"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
)

var addressAssetBalanceService = AddressAssetBalanceService{}

type TransactionService struct{}

func (transactionService TransactionService) Insert(blockHash string, height int64, time int64, rawTx []rpcjson.TxResult, vTx []rpcjson.TxResult) error {
	transactions := make([]interface{}, 0)
	addressAssetBalances := make([]model.AddressAssetBalance, 0)
	for i, tx := range rawTx {
		transaction, addressAssetBalanceSlice := assemble(blockHash, height, tx, i)
		transactions = append(transactions, &transaction)
		addressAssetBalances = append(addressAssetBalances, addressAssetBalanceSlice...)
	}
	
	virtualTransactions := make([]interface{}, 0)
	for i, tx := range vTx {
		transaction, addressAssetBalanceSlice := assemble(blockHash, height, tx, i)
		virtualTransactions = append(virtualTransactions, &transaction)
		transactions[transaction.Version].(*model.Transaction).VtxHash = transaction.Hash
		addressAssetBalances = append(addressAssetBalances, addressAssetBalanceSlice...)
	}
	_, err := mongo.MongoDB.Collection(mongo.CollectionTransaction).InsertMany(context.TODO(), transactions)
	if err != nil {
		return err
	}
	
	if len(virtualTransactions) > 0 {
		_, err = mongo.MongoDB.Collection(mongo.CollectionVirtualTransaction).InsertMany(context.TODO(), virtualTransactions)
		if err != nil {
			return err
		}
	}
	
	err = addressAssetBalanceService.InsertOrUpdate(height, addressAssetBalances)
	if err != nil {
		return err
	}
	
	assetTransactionSlice := make([]interface{}, 0)
	transactionTxCountSlice := make([]model.TransactionCount, 0)
	for _, tx := range rawTx {
		vinAssets := make([]string, 0)
		for _, vin := range tx.Vin {
			if vin.PrevOut != nil {
				vinAssets = append(vinAssets, vin.PrevOut.Asset)
			}
		}
		vinAssets = common.RemoveRepeatByLoop(vinAssets)
		for _, asset := range vinAssets {
			feeSlice := make([]model.Fee, 0)
			for _, v := range tx.Fee {
				tmp := model.Fee{
					Value: v.Value,
					Asset: v.Asset,
				}
				feeSlice = append(feeSlice, tmp)
			}
			assetTransactionSlice = append(assetTransactionSlice, model.TransactionList{
				Height: height,
				Key:    asset,
				TxHash: tx.Hash,
				Time:   tx.Time,
				Fee:    feeSlice,
			})
			
			transactionTxCount := model.TransactionCount{
				Key:      asset,
				Category: model.CountAsset,
			}
			transactionTxCountSlice = append(transactionTxCountSlice, transactionTxCount)
		}
	}
	
	err = transactionStatisticsService.InsertOrUpdate(model.CountAsset, transactionTxCountSlice)
	if err != nil {
		return err
	}
	
	err = transactionStatisticsService.Record(mongo.CollectionAssetTransaction, assetTransactionSlice)
	if err != nil {
		return err
	}
	return nil
}

func assemble(blockHash string, height int64, tx rpcjson.TxResult, txIndex int) (model.Transaction, []model.AddressAssetBalance) {
	addressAssetBalanceSlice := make([]model.AddressAssetBalance, 0)
	transaction := model.Transaction{
		BlockHash:     blockHash,
		Height:        height,
		// Hex:           tx.Hex,
		Hash:          tx.Hash,
		Size:          tx.Size,
		Version:       tx.Version,
		LockTime:      tx.LockTime,
		Confirmations: tx.Confirmations,
		Time:          tx.Time,
		GasLimit:      tx.GasLimit,
	}
	
	feeSlice := make([]model.Fee, 0)
	for _, v := range tx.Fee {
		tmp := model.Fee{
			Value: v.Value,
			Asset: v.Asset,
		}
		feeSlice = append(feeSlice, tmp)
	}
	transaction.Fee = feeSlice
	
	// Vin
	vins := make([]model.Vin, 0)
	for i := 0; i < len(tx.Vin); i++ {
		v := tx.Vin[i]
		// record address balance
		if v.PrevOut != nil {
			asset := protos.AssetFromBytes(asimovCommon.Hex2Bytes(v.PrevOut.Asset))
			value := -v.PrevOut.Value
			if asset.IsIndivisible() && v.PrevOut.Value > 0 {
				value = -1
			}
			for _, x := range v.PrevOut.Addresses {
				addressAssetBalance := model.AddressAssetBalance{
					Height:  height,
					Time:    tx.Time,
					Address: x,
					Asset:   v.PrevOut.Asset,
					Balance: value,
				}
				addressAssetBalanceSlice = append(addressAssetBalanceSlice, addressAssetBalance)
			}
		}
		
		vin := model.Vin{
			// TxHash:   tx.Hash,
			Sequence: v.Sequence,
			// Height:   height,
			// Time:     tx.Time,
		}
		
		if v.Coinbase != "" {
			// coin base transaction
			vin.CoinBase = v.Coinbase
		} else {
			vin.OutTxHash = v.Txid
			vin.VOut = &v.Vout
			scriptSig := &model.ScriptSig{
				// Asm: v.ScriptSig.Asm,
				Hex: v.ScriptSig.Hex,
			}
			vin.ScriptSig = scriptSig
			preOut := &model.PrevOut{
				Addresses: v.PrevOut.Addresses,
				Value:     v.PrevOut.Value,
				Asset:     v.PrevOut.Asset,
				// Data:      v.PrevOut.Data,
			}
			vin.PrevOut = preOut
		}
		vins = append(vins, vin)
	}
	
	// VOut
	vouts := make([]model.Vout, 0)
	for i := 0; i < len(tx.Vout); i++ {
		v := tx.Vout[i]
		asset := protos.AssetFromBytes(asimovCommon.Hex2Bytes(v.Asset))
		value := v.Value
		if asset.IsIndivisible() && v.Value > 0 {
			value = 1
		}
		// record address balance
		for _, x := range v.ScriptPubKey.Addresses {
			addressAssetBalance := model.AddressAssetBalance{
				Height:  height,
				Time:    tx.Time,
				Address: x,
				Asset:   v.Asset,
				Balance: value,
			}
			addressAssetBalanceSlice = append(addressAssetBalanceSlice, addressAssetBalance)
		}
		
		scriptPubKey := model.ScriptPubKey{
			Asm: v.ScriptPubKey.Asm,
			// Hex:       v.ScriptPubKey.Hex,
			ReqSigs:   v.ScriptPubKey.ReqSigs,
			Type:      v.ScriptPubKey.Type,
			Addresses: v.ScriptPubKey.Addresses,
		}
		vout := model.Vout{
			// TxHash:       tx.Hash,
			// Height:       height,
			// Time:         tx.Time,
			Value:        v.Value,
			N:            v.N,
			ScriptPubKey: scriptPubKey,
			// Data:         v.Data,
			Asset: v.Asset,
		}
		vouts = append(vouts, vout)
	}
	
	transaction.Vin = vins
	transaction.Vout = vouts
	return transaction, addressAssetBalanceSlice
}
