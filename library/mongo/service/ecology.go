package service

import (
	"context"
	"github.com/AsimovNetwork/asimov/rpcs/rpcjson"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"gopkg.in/mgo.v2/bson"
)

type EcologyService struct{}

var contractService = ContractService{}
var transactionStatisticsService = TransactionStatisticsService{}

func (ecologyService EcologyService) Analyze(block rpcjson.GetBlockVerboseResult) error {
	err := recordAddress(block.Height, block.RawTx)
	if err != nil {
		return err
	}

	err = recordContract(block.Height, block.Receipts, block.RawTx)
	if err != nil {
		return err
	}

	err = recordAsiTrading(block.Height, block.RawTx)
	if err != nil {
		return err
	}

	return nil
}

func recordAddress(height int64, rawTx []rpcjson.TxResult) error {
	addresses := make([]string, 0)
	addressTransactionSlice := make([]interface{}, 0)
	transactionTxCountSlice := make([]model.TransactionCount, 0)
	addressTransactionMap := make(map[string]model.TransactionList)

	for _, tx := range rawTx {
		for _, vin := range tx.Vin {
			if vin.PrevOut != nil {
				for _, address := range vin.PrevOut.Addresses {
					if address[:4] == common.CitizenPrefix {
						if _, ok := addressTransactionMap[address]; !ok {
							feeSlice := make([]model.Fee, 0)
							for _, v := range tx.Fee {
								tmp := model.Fee{
									Value: v.Value,
									Asset: v.Asset,
								}
								feeSlice = append(feeSlice, tmp)
							}
							addressTransactionMap[address] = model.TransactionList{
								Height: height,
								Key:    address,
								TxHash: tx.Hash,
								Time:   tx.Time,
								Fee:    feeSlice,
							}

							transactionTxCount := model.TransactionCount{
								Key:      address,
								Category: model.CountAddress,
							}
							transactionTxCountSlice = append(transactionTxCountSlice, transactionTxCount)
						}
					}
				}
			}
		}

		for _, vout := range tx.Vout {
			for _, address := range vout.ScriptPubKey.Addresses {
				if address[:4] == common.CitizenPrefix {
					if _, ok := addressTransactionMap[address]; !ok {
						feeSlice := make([]model.Fee, 0)
						for _, v := range tx.Fee {
							tmp := model.Fee{
								Value: v.Value,
								Asset: v.Asset,
							}
							feeSlice = append(feeSlice, tmp)
						}
						addressTransactionMap[address] = model.TransactionList{
							Height: height,
							Key:    address,
							TxHash: tx.Hash,
							Time:   tx.Time,
							Fee:    feeSlice,
						}

						transactionTxCount := model.TransactionCount{
							Key:      address,
							Category: model.CountAddress,
						}
						transactionTxCountSlice = append(transactionTxCountSlice, transactionTxCount)
					}
				}
			}
		}
	}

	for k, v := range addressTransactionMap {
		addresses = append(addresses, k)
		addressTransactionSlice = append(addressTransactionSlice, v)
	}

	err := transactionStatisticsService.InsertOrUpdate(model.CountAddress, transactionTxCountSlice)
	if err != nil {
		return err
	}

	err = transactionStatisticsService.Record(mongo.CollectionAddressTransaction, addressTransactionSlice)
	if err != nil {
		return err
	}

	return nil
}

func recordContract(height int64, receipts []*rpcjson.ReceiptResult, txs []rpcjson.TxResult) error {
	addresses := make([]string, 0)
	contractTransactionSlice := make([]interface{}, 0)
	transactionTxCountSlice := make([]model.TransactionCount, 0)
	contractTransactionMap := make(map[string]model.TransactionList)
	for _, receipt := range receipts {
		if len(receipt.ContractAddress) > 0 && receipt.ContractAddress != common.EmptyContract {
			var creator string
			time := int64(0)
			for _, tx := range txs {
				if tx.Hash == receipt.TxHash {
					time = tx.Time
					feeSlice := make([]model.Fee, 0)
					for _, v := range tx.Fee {
						tmp := model.Fee{
							Value: v.Value,
							Asset: v.Asset,
						}
						feeSlice = append(feeSlice, tmp)
					}
					contractTransactionMap[receipt.ContractAddress] = model.TransactionList{
						Height: height,
						Key:    receipt.ContractAddress,
						TxHash: tx.Hash,
						Time:   tx.Time,
						Fee:    feeSlice,
					}

					if tx.Vin[0].PrevOut != nil {
						creator = tx.Vin[0].PrevOut.Addresses[0]
						break
					}
				}
			}

			contractTemplate, err := contractService.GetTemplate(receipt.ContractAddress)
			if err != nil {
				return err
			}

			transactionTxCount := model.TransactionCount{
				Key:           receipt.ContractAddress,
				Category:      model.CountContract,
				Time:          time,
				TxHash:        receipt.TxHash,
				Creator:       creator,
				TemplateType:  contractTemplate.TemplateType,
				TemplateTName: contractTemplate.TemplateTName,
			}
			transactionTxCountSlice = append(transactionTxCountSlice, transactionTxCount)
		}
	}

	for _, tx := range txs {
		for _, vout := range tx.Vout {
			for _, address := range vout.ScriptPubKey.Addresses {
				if address[:4] == common.ContractPrefix {
					if _, ok := common.SystemContractAddressMap[address]; !ok {
						if _, ok := contractTransactionMap[address]; !ok {
							feeSlice := make([]model.Fee, 0)
							for _, v := range tx.Fee {
								tmp := model.Fee{
									Value: v.Value,
									Asset: v.Asset,
								}
								feeSlice = append(feeSlice, tmp)
							}
							contractTransactionMap[address] = model.TransactionList{
								Height: height,
								Key:    address,
								TxHash: tx.Hash,
								Time:   tx.Time,
								Fee:    feeSlice,
							}

							transactionTxCount := model.TransactionCount{
								Key:      address,
								Category: model.CountContract,
							}
							transactionTxCountSlice = append(transactionTxCountSlice, transactionTxCount)
						}
					}
				}
			}
		}
	}
	for k, v := range contractTransactionMap {
		addresses = append(addresses, k)
		contractTransactionSlice = append(contractTransactionSlice, v)
	}

	err := transactionStatisticsService.InsertOrUpdate(model.CountContract, transactionTxCountSlice)
	if err != nil {
		return err
	}

	err = transactionStatisticsService.Record(mongo.CollectionContractTransaction, contractTransactionSlice)
	if err != nil {
		return err
	}
	return nil
}

func recordAsiTrading(height int64, txs []rpcjson.TxResult) error {
	tradings := make([]interface{}, 0)
	for _, tx := range txs {
		for _, vout := range tx.Vout {
			if vout.Value > 0 {
				tmp := model.Trading{
					Height: height,
					Value:  vout.Value,
					Time:   tx.Time,
					Asset:  vout.Asset,
				}
				tradings = append(tradings, tmp)
			}
		}
	}

	if len(tradings) > 0 {
		_, err := mongo.MongoDB.Collection(mongo.CollectionTrading).InsertMany(context.TODO(), tradings)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ecologyService EcologyService) DropOneDayBeforeData() error {
	oneDayBeforeTime := common.NowSecond() - common.SecondsOfDay
	deleteFilter := bson.M{
		"time": bson.M{
			"$lt": oneDayBeforeTime,
		},
	}

	_, err := mongo.MongoDB.Collection(mongo.CollectionTrading).DeleteMany(context.TODO(), deleteFilter)
	if err != nil {
		common.Logger.Errorf("delete from %s error. err: %s", mongo.CollectionTrading, err)
		return err
	}

	return nil
}
