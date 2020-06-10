package event

import (
	"encoding/json"
	"errors"
	"fmt"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	"math/big"
)

type MintAssetEvent struct{}

func (mintAssetEvent MintAssetEvent) handle(b blockInfo, args map[string]interface{}) error {
	contractAddress, ok := args["contractAddress"]
	if !ok {
		return mintAssetEvent.checkArg("contractAddress")
	}

	assetIndex, ok := args["assetIndex"]
	if !ok {
		return mintAssetEvent.checkArg("assetIndex")
	}

	amountOrVoucherId, ok := args["amountOrVoucherId"]
	if !ok {
		return mintAssetEvent.checkArg("amountOrVoucherId")
	}
	asset, err := daoOrganizationAssetService.GetAsset(contractAddress.(*asimovCommon.Address).Hex(), *assetIndex.(*uint32))
	if err != nil {
		return err
	}
	if asset.AssetType == model.AssetTypeIndivisible {
		err := mysqlDaoIndivisibleAssetService.UpdateByAssetAndVoucherId(asset.Asset, (*amountOrVoucherId.(**big.Int)).Int64())
		if err != nil {
			return err
		}
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["asset"] = asset.Asset
	additionalInfo["asset_type"] = asset.AssetType
	additionalInfo["amount_or_voucher_id"] = (*amountOrVoucherId.(**big.Int)).Int64()

	jsonStr, _ := json.Marshal(additionalInfo)
	err = daoMessageService.SaveMessage(b.height, constant.MessageCategoryMintAsset, constant.MessageTypeReadOnly, constant.MessagePositionBoth, contractAddress.(*asimovCommon.Address).Hex(), "", string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

func (mintAssetEvent MintAssetEvent) checkArg(arg string) error {
	errMsg := fmt.Sprintf("dao min asset event miss arg %s", arg)
	common.Logger.Error(errMsg)
	return errors.New(errMsg)
}
