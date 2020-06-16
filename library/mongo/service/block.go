package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/AsimovNetwork/asimov/rpcs/rpcjson"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
)

type BlockService struct{}

var validatorService = ValidatorService{}
var earningService = EarningService{}

func (blockService BlockService) GetBestBlockHeight() (int32, error) {
	// make([]interface{}, 0) --> []interface{}{}
	param := common.NewChainRequest("getBestBlock", []interface{}{})
	result, ok := common.Post(common.Cfg.BlockChainRpc, param)
	if !ok {
		return 0, errors.New("call block chain failed")
	}
	resultMap := (result).(map[string]interface{})

	bestBlock := &rpcjson.GetBestBlockResult{}
	err := common.ToStruct(resultMap, bestBlock)
	return bestBlock.Height, err
}

func (blockService BlockService) FetchBlocks(offset int32, count int32) ([]rpcjson.GetBlockVerboseResult, error) {
	param := common.NewChainRequest("getBlockListByHeight", []interface{}{offset, count})
	result, ok := common.Post(common.Cfg.BlockChainRpc, param)
	if !ok {
		return nil, response.NewCallBlockChainError()
	}
	mapSlice := (result).([]interface{})
	blockSlice := make([]rpcjson.GetBlockVerboseResult, 0)
	for _, v := range mapSlice {
		block := &rpcjson.GetBlockVerboseResult{}
		err := common.ToStruct(v, block)
		if err != nil {
			return nil, err
		}
		blockSlice = append(blockSlice, *block)
	}

	return blockSlice, nil
}

func (blockService BlockService) Insert(rpcBlock rpcjson.GetBlockVerboseResult) (primitive.ObjectID, error) {
	block := model.Block{
		Hash:              rpcBlock.Hash,
		Confirmations:     rpcBlock.Confirmations,
		Size:              rpcBlock.Size,
		Height:            rpcBlock.Height,
		Version:           rpcBlock.Version,
		MerkleRoot:        rpcBlock.MerkleRoot,
		Time:              rpcBlock.Time,
		TxCount:           rpcBlock.TxCount,
		PreviousBlockHash: rpcBlock.PreviousHash,
		StateRoot:         rpcBlock.StateRoot,
		Slot:              int32(rpcBlock.Slot),
		Round:             int32(rpcBlock.Round),
		Reward:            rpcBlock.Reward,
	}

	// 交易费
	feeSlice := make([]model.Fee, 0)
	for _, v := range rpcBlock.FeeList {
		tmp := model.Fee{
			Value: v.Value,
			Asset: v.Asset,
		}
		feeSlice = append(feeSlice, tmp)
	}
	block.Fee = feeSlice

	for _, tx := range rpcBlock.RawTx {
		if tx.Vin[0].Coinbase != "" {
			//block.Produced = tx.Vout[len(tx.Vout)-1].ScriptPubKey.Addresses[0]
			block.Produced = tx.Vout[0].ScriptPubKey.Addresses[0]
			// block.Reward = tx.Vout[0].Value
			if rpcBlock.Height > 0 {
				var earnings = make(map[string]map[string]int64)
				for _, vout := range tx.Vout {
					address := vout.ScriptPubKey.Addresses[0]
					if address != common.GenesisOrganizationAddress && address != common.ConsensusAddress {
						earning, ok := earnings[address]
						if !ok {
							earnings[address] = map[string]int64{vout.Asset: vout.Value}
						} else {
							amount, ok := earning[vout.Asset]
							if !ok {
								earning[vout.Asset] = vout.Value
							} else {
								earning[vout.Asset] = amount + vout.Value
							}
						}
					}
				}
				err := earningService.Insert(rpcBlock.Height, rpcBlock.Time, tx.Hash, earnings)
				if err != nil {
					return primitive.NilObjectID, err
				}
			}
			break
		}
	}

	insertResult, err := mongo.MongoDB.Collection(mongo.CollectionBlock).InsertOne(context.TODO(), block)
	if err != nil {
		common.Logger.Errorf("insert block error. err: %s", err)
		return primitive.NilObjectID, err
	}

	return insertResult.InsertedID.(primitive.ObjectID), err
}

func (blockService BlockService) GetHandledBlockHeight() (int64, error) {
	var result model.Block
	filter := bson.M{}
	option := options.FindOne().SetSort(bson.M{"height": -1})
	err := mongo.FindOne(mongo.CollectionBlock, filter, &result, option)
	return result.Height, err
}

func (blockService BlockService) GetBlockByHeight(height int64) (*model.Block, error) {
	var result model.Block
	filter := bson.M{"height": height}
	err := mongo.FindOne(mongo.CollectionBlock, filter, &result)
	return &result, err
}

func (blockService BlockService) CountTransaction(height int64) (int64, error) {
	command := `[
    {
      "$match": {"height": {"$gt" : %d}}
    },
    {
      "$group": {"_id": null, "total": {"$sum": "$tx_count"}}
    }
  ]`
	command = fmt.Sprintf(command, height)
	txCount, err := mongo.Sum(mongo.CollectionBlock, []byte(command), reflect.TypeOf(*new(int64)))
	return txCount.(int64), err
}
