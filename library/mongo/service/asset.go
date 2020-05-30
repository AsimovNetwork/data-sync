package service

import (
	"context"
	"errors"
	"fmt"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/asimov/protos"
	"github.com/AsimovNetwork/asimov/rpcs/rpcjson"
	"github.com/AsimovNetwork/asimov/vm/fvm"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/response"
	"go.mongodb.org/mongo-driver/bson"
)

type AssetService struct {
	AssetInfo func(asset string) (map[string]string, error)
}

var systemContractService = SystemContractService{}
var blockService = BlockService{}

func (assetService AssetService) HeightInit() error {
	block, err := blockService.FetchBlocks(0, 1)
	if err != nil {
		common.Logger.ErrorPanic("insert assetInfo error:", err)
	}
	assetInfoMap, err := assetService.AssetInfo(common.ASIM)
	if err != nil {
		return err
	}
	if assetInfoMap == nil {
		return errors.New("ASIM is not exist")
	}
	assetInfo := model.Asset{
		Height:       0,
		Time:         block[0].Time,
		IssueAddress: "",
		Asset:        assetInfoMap["asset"],
		Name:         assetInfoMap["name"],
		Symbol:       assetInfoMap["symbol"],
		Description:  assetInfoMap["description"],
		Logo:         assetInfoMap["logo"],
	}
	filter := bson.M{
		"asset": common.ASIM,
	}
	exist, err := mongo.Exist(mongo.CollectionAsset, filter)
	if err != nil {
		return err
	}
	if !exist {
		_, err = mongo.MongoDB.Collection(mongo.CollectionAsset).InsertOne(context.TODO(), assetInfo)
		if err != nil {
			return err
		}
	}

	common.Logger.Info("init asset success.")
	return nil
}

func (assetService AssetService) GetAssetInfo(asset string) *model.Asset {
	assetInfo, err := getAssetInfoFromDB(asset)
	if err != nil {
		common.Logger.Errorf("get asset info from mongo failed, asset: %s", asset)
		return nil
	}
	return assetInfo
}

// TODO 考虑资产信息是否可以不用从合约获取了，直接用assetService.AssetInfo(asset)的信息
func (assetService AssetService) getAssetInfo(asset string) (*model.Asset, error) {
	assetInfo := &model.Asset{
		Asset:       asset,
		Name:        "",
		Symbol:      "",
		Description: "",
		Logo:        "",
	}

	currentHeight, err := blockService.GetHandledBlockHeight()
	if err != nil {
		return assetInfo, err
	}
	exist, abi, err := systemContractService.GetSystemContractAbi(currentHeight, common.GenesisRegistryAddress)
	if err != nil || !exist {
		return assetInfo, err
	}
	funcName := "getAssetInfoByAssetId"
	_, orgId, coinId := protos.AssetFromBytes(asimovCommon.Hex2Bytes(asset)).AssetFields()
	data, err := fvm.PackFunctionArgs(abi, funcName, orgId, coinId)
	if err != nil {
		return assetInfo, err
	}

	param := common.NewChainRequest("callReadOnlyFunction", []interface{}{common.OfficialCaller, common.GenesisRegistryAddress, asimovCommon.Bytes2Hex(data), funcName, abi})
	result, ok := common.Post(common.Cfg.BlockChainRpc, param)
	if !ok {
		return assetInfo, errors.New("call block chain failed")
	}
	slice := (result).([]interface{})
	// The 0th parameter indicates whether the currency information was queried
	if slice[0].(bool) == false {
		return assetInfo, errors.New(fmt.Sprintf("asset %v do not exists", asset))
	}

	assetLogoMap, _ := assetService.AssetInfo(asset)

	assetInfo = &model.Asset{
		Asset:       asset,
		Name:        slice[1].(string),
		Symbol:      slice[2].(string),
		Description: slice[3].(string),
		Logo:        assetLogoMap[asset],
	}

	return assetInfo, nil
}

func getAssetInfoFromDB(asset string) (*model.Asset, error) {
	var result model.Asset
	filter := bson.M{"asset": asset}
	err := mongo.FindOne(mongo.CollectionAsset, filter, &result)
	return &result, err
}

func (assetService AssetService) Insert(blockHash string, height int64, rawTx []rpcjson.TxResult, vTx []rpcjson.TxResult) error {
	if len(vTx) == 0 {
		return nil
	}

	var assetInfoObj model.Asset
	var assetIssueObj model.AssetIssue
	// virtual transaction
	for _, tx := range vTx {
		// coin base is not empty, indicating that is the first create or mint
		if len(tx.Vin) > 0 && len(tx.Vout) > 0 {
			vin := tx.Vin[0]
			vout := tx.Vout[0]
			if vin.Coinbase != "" {
				assetInfo, err := assetService.getAssetInfo(vout.Asset)
				if err != nil {
					common.Logger.Errorf("get asset info from rpc failed, asset: %s, error: %s", vout.Asset, err)
				}
				assetInfoObj = model.Asset{
					Height:       height,
					Time:         tx.Time,
					IssueAddress: vout.ScriptPubKey.Addresses[0],
					Asset:        vout.Asset,
					Name:         assetInfo.Name,
					Symbol:       assetInfo.Symbol,
					Description:  assetInfo.Description,
					Logo:         assetInfo.Logo,
				}
				assetIssueObj = model.AssetIssue{
					Asset:     vout.Asset,
					Height:    height,
					Time:      tx.Time,
					IssueType: 0,
					Value:     vout.Value,
				}
				filter := bson.M{"asset": assetInfoObj.Asset}
				err = mongo.FindOne(mongo.CollectionAsset, filter, &assetInfoObj)
				if err == nil {
					// if err is nil, record exist, mint
					assetIssueObj.IssueType = model.IssueAsset
				} else if response.IsDataNotExistError(err) {
					assetIssueObj.IssueType = model.CreateAsset
					// save assetInfo
					_, err = mongo.MongoDB.Collection(mongo.CollectionAsset).InsertOne(context.TODO(), assetInfoObj)
					if err != nil {
						return err
					}
				} else {
					return err
				}
				// save assetIssueInfo
				_, err = mongo.MongoDB.Collection(mongo.CollectionAssetIssue).InsertOne(context.TODO(), assetIssueObj)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
