package event

import (
	"errors"
	"fmt"
	asimovCommon "github.com/AsimovNetwork/asimov/common"
	"github.com/AsimovNetwork/data-sync/library/common"
	mongodb "github.com/AsimovNetwork/data-sync/library/mongo/service"
	"github.com/AsimovNetwork/data-sync/library/mysql/service"
)

type CloseOrganizationEvent struct{}

var contractService = mongodb.ContractService{}

var daoMessageService = service.DaoMessageService{}

func (closeOrganizationEvent CloseOrganizationEvent) handle(b blockInfo, args map[string]interface{}) error {
	contractAddress, ok := args["contractAddress"]
	if !ok {
		return closeOrganizationEvent.checkArg("contractAddress")
	}
	// update dao_organization of mongodb
	err := daoOrganizationService.CloseOrg(b.height, contractAddress.(*asimovCommon.Address).Hex())
	if err != nil {
		return err
	}

	// update t_dao_organization of mysql
	err = mysqlDaoOrganizationService.CloseOrg(contractAddress.(*asimovCommon.Address).Hex(), b.height)
	if err != nil {
		return err
	}
	return nil
}

func (closeOrganizationEvent CloseOrganizationEvent) checkArg(arg string) error {
	errMsg := fmt.Sprintf("dao close organization event miss arg %s", arg)
	common.Logger.Error(errMsg)
	return errors.New(errMsg)
}
