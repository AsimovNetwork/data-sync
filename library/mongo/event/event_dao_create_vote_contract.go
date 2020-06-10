package event

import (
	"encoding/json"
	"errors"
	"fmt"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	"github.com/AsimovNetwork/data-sync/library/mysql/service"
)

type CreateVoteContract struct{}

var mysqlDaoOrganizationService = service.MysqlDaoOrganizationService{}

func (createVoteContract CreateVoteContract) handle(b blockInfo, args map[string]interface{}) error {
	contractAddress, ok := args["contractAddress"]
	if !ok {
		return createVoteContract.checkArg("contractAddress")
	}

	voteContractAddress, ok := args["voteContractAddress"]
	if !ok {
		return createVoteContract.checkArg("voteContractAddress")
	}

	voteTemplateName, ok := args["voteTemplateName"]
	if !ok {
		return createVoteContract.checkArg("voteTemplateName")
	}

	president, ok := args["president"]
	if !ok {
		return createVoteContract.checkArg("president")
	}

	orgName, ok := args["orgName"]
	if !ok {
		return createVoteContract.checkArg("orgName")
	}

	orgId, ok := args["orgId"]
	if !ok {
		return createVoteContract.checkArg("orgId")
	}

	err := daoOrganizationService.Insert(b.height, b.time, b.txHash, contractAddress.(*asimovCommon.Address).Hex(), voteContractAddress.(*asimovCommon.Address).Hex(), *voteTemplateName.(*string), president.(*asimovCommon.Address).Hex(), *orgName.(*string), *orgId.(*uint32))
	if err != nil {
		return err
	}

	err = mysqlDaoOrganizationService.CreateOrg(b.txHash, contractAddress.(*asimovCommon.Address).Hex(), voteContractAddress.(*asimovCommon.Address).Hex())
	if err != nil {
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["creator_address"] = president.(*asimovCommon.Address).Hex()
	jsonStr, _ := json.Marshal(additionalInfo)
	err = daoMessageService.SaveMessage(b.height, constant.MessageCategoryCreateOrg, constant.MessageTypeReadOnly, constant.MessagePositionBoth, contractAddress.(*asimovCommon.Address).Hex(), "", string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

func (createVoteContract CreateVoteContract) checkArg(arg string) error {
	errMsg := fmt.Sprintf("dao create vote contract event miss arg %s", arg)
	common.Logger.Error(errMsg)
	return errors.New(errMsg)
}
