package event

import (
	"encoding/json"
	"errors"
	"fmt"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/asimov/protos"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	"math/big"
)

type TransferSuccessEvent struct{}

func (transferSuccessEvent TransferSuccessEvent) handle(b blockInfo, args map[string]interface{}) error {
	contractAddress, ok := args["contractAddress"]
	if !ok {
		return transferSuccessEvent.checkArg("contractAddress")
	}

	receiver, ok := args["receiver"]
	if !ok {
		return transferSuccessEvent.checkArg("receiver")
	}

	asset, ok := args["asset"]
	if !ok {
		return transferSuccessEvent.checkArg("asset")
	}
	assetObj := protos.AssetFromInt(*asset.(**big.Int))
	assetStr := asimovCommon.Bytes2Hex(assetObj.Bytes())

	amount, ok := args["amount"]
	if !ok {
		return transferSuccessEvent.checkArg("amount")
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["asset"] = assetStr
	additionalInfo["amount"] = (*amount.(**big.Int)).Int64()
	additionalInfo["target_address"] = receiver.(*asimovCommon.Address).Hex()
	jsonStr, _ := json.Marshal(additionalInfo)
	err := daoMessageService.SaveMessage(b.height, constant.MessageCategoryTransferAsset, constant.MessageTypeReadOnly, constant.MessagePositionBoth, contractAddress.(*asimovCommon.Address).Hex(), "", string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

func (transferSuccessEvent TransferSuccessEvent) checkArg(arg string) error {
	errMsg := fmt.Sprintf("dao transfer success event miss arg %s", arg)
	common.Logger.Error(errMsg)
	return errors.New(errMsg)
}
