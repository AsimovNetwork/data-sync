package event

import (
	"encoding/json"
	"errors"
	"fmt"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/asimov/protos"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo/service"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	mysqlService "github.com/AsimovNetwork/data-sync/library/mysql/service"
	"math/big"
)

type CreateAssetEvent struct{}

var daoOrganizationAssetService = service.DaoOrganizationAssetService{}
var mysqlDaoAssetService = mysqlService.MysqlDaoAssetService{}
var mysqlDaoIndivisibleAssetService = mysqlService.MysqlDaoIndivisibleAssetService{}

func (createAssetEvent CreateAssetEvent) handle(b blockInfo, args map[string]interface{}) error {
	contractAddress, ok := args["contractAddress"]
	if !ok {
		return createAssetEvent.checkArg("contractAddress")
	}

	name, ok := args["name"]
	if !ok {
		return createAssetEvent.checkArg("name")
	}

	symbol, ok := args["symbol"]
	if !ok {
		return createAssetEvent.checkArg("symbol")
	}

	description, ok := args["description"]
	if !ok {
		return createAssetEvent.checkArg("description")
	}

	assetType, ok := args["assetType"]
	if !ok {
		return createAssetEvent.checkArg("assetType")
	}

	organizationId, ok := args["organizationId"]
	if !ok {
		return createAssetEvent.checkArg("organizationId")
	}

	assetIndex, ok := args["assetIndex"]
	if !ok {
		return createAssetEvent.checkArg("assetIndex")
	}

	amount, ok := args["amount"]
	if !ok {
		return createAssetEvent.checkArg("amount")
	}

	asset := protos.NewAsset(*assetType.(*uint32), *organizationId.(*uint32), *assetIndex.(*uint32))

	err := daoOrganizationAssetService.Insert(b.height, b.time, contractAddress.(*asimovCommon.Address).Hex(), asimovCommon.Bytes2Hex(asset.Bytes()), (*amount.(**big.Int)).Int64(), *assetType.(*uint32), *assetIndex.(*uint32))
	if err != nil {
		return err
	}

	err = mysqlDaoAssetService.UpdateByAsset(asimovCommon.Bytes2Hex(asset.Bytes()))
	if err != nil {
		return err
	}

	err = mysqlDaoIndivisibleAssetService.UpdateByAsset(asimovCommon.Bytes2Hex(asset.Bytes()))
	if err != nil {
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["asset"] = asimovCommon.Bytes2Hex(asset.Bytes())
	additionalInfo["amount"] = (*amount.(**big.Int)).Int64()
	additionalInfo["name"] = *name.(*string)
	additionalInfo["symbol"] = *symbol.(*string)
	additionalInfo["description"] = *description.(*string)
	jsonStr, _ := json.Marshal(additionalInfo)
	err = daoMessageService.SaveMessage(constant.MessageCategoryIssueAsset, constant.MessageTypeReadOnly, constant.MessagePositionBoth, contractAddress.(*asimovCommon.Address).Hex(), "", string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

func (createAssetEvent CreateAssetEvent) checkArg(arg string) error {
	errMsg := fmt.Sprintf("dao create asset event miss arg %s", arg)
	common.Logger.Error(errMsg)
	return errors.New(errMsg)
}
