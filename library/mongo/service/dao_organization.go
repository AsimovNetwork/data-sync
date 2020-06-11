package service

import (
	"context"
	"encoding/json"
	"github.com/AsimovNetwork/asimov/rpcs/rpcjson"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	"github.com/AsimovNetwork/data-sync/library/mysql/service"
	"github.com/AsimovNetwork/data-sync/library/response"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"strings"
)

type DaoOrganizationService struct{}

type DaoReceiveAsset struct {
	ContractAddress string
	SenderAddress   string
	Asset           string
	Amount          int64
}

var daoMessageService = service.DaoMessageService{}

var ContractAddressCache = map[string]string{}

func init() {
	filter := bson.M{
		"status": model.OrgStatusNormal,
	}
	orgList, err := mongo.Find(mongo.CollectionDaoOrganization, filter, reflect.TypeOf(model.DaoOrganization{}), reflect.TypeOf(&model.DaoOrganization{}))
	if err != nil {
		common.Logger.Errorf("cache contract address error. err: %s", err)
	}
	for _, v := range orgList.([]*model.DaoOrganization) {
		ContractAddressCache[v.ContractAddress] = v.ContractAddress
	}
}

func (daoOrganizationService DaoOrganizationService) Insert(height int64, time int64, txHash string, contractAddress string, voteContractAddress string, voteTemplateName string, president string, orgName string, orgId uint32) error {
	insertOrg := model.DaoOrganization{
		Height:              height,
		Time:                time,
		TxHash:              txHash,
		ContractAddress:     contractAddress,
		VoteContractAddress: voteContractAddress,
		VoteTemplateName:    voteTemplateName,
		OrgName:             orgName,
		OrgId:               orgId,
		President:           president,
		Status:              model.OrgStatusNormal,
	}

	_, err := mongo.MongoDB.Collection(mongo.CollectionDaoOrganization).InsertOne(context.TODO(), insertOrg)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoOrganization, err)
		return err
	}

	insertPresident := model.DaoMember{
		Height:          height,
		Time:            time,
		ContractAddress: contractAddress,
		Role:            model.MemberRolePresident,
		Address:         president,
		Status:          model.MemberStatusAgreed,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).InsertOne(context.TODO(), insertPresident)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoMember, err)
		return err
	}

	ContractAddressCache[contractAddress] = contractAddress

	return nil
}

func (daoOrganizationService DaoOrganizationService) UpdateOrgName(height int64, contractAddress string, newName string) error {
	var org model.DaoOrganization
	filter := bson.M{
		"contract_address": contractAddress,
	}
	err := mongo.FindOne(mongo.CollectionDaoOrganization, filter, &org)
	if err != nil {
		return err
	}
	update := bson.M{
		"org_name": newName,
	}

	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoOrganization).UpdateOne(context.TODO(), filter, bson.M{"$set": update})

	err = rollbackService.Insert(mongo.CollectionDaoOrganization, org.ID, height, "org_name", org.OrgName, newName)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoOrganization, err)
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["old_org_name"] = org.OrgName
	additionalInfo["new_org_name"] = newName
	jsonStr, _ := json.Marshal(additionalInfo)
	err = daoMessageService.SaveMessage(height, constant.MessageCategoryModifyOrgName, constant.MessageTypeReadOnly, constant.MessagePositionBoth, contractAddress, "", string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

func (daoOrganizationService DaoOrganizationService) CloseOrg(height int64, contractAddress string) error {
	var org model.DaoOrganization
	filter := bson.M{
		"contract_address": contractAddress,
		"status":           model.OrgStatusNormal,
	}
	err := mongo.FindOne(mongo.CollectionDaoOrganization, filter, &org)
	if err != nil {
		if response.IsDataNotExistError(err) {
			return nil
		}
		return err
	}

	update := bson.M{
		"status": model.OrgStatusClosed,
	}

	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoOrganization).UpdateOne(context.TODO(), filter, bson.M{"$set": update})

	err = rollbackService.Insert(mongo.CollectionDaoOrganization, org.ID, height, "status", model.OrgStatusNormal, model.OrgStatusClosed)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoOrganization, err)
		return err
	}

	delete(ContractAddressCache, contractAddress)

	err = daoMessageService.SaveMessage(height, constant.MessageCategoryCloseOrg, constant.MessageTypeReadOnly, constant.MessagePositionWeb, contractAddress, "", "{}")
	if err != nil {
		return err
	}

	return nil
}

func (daoOrganizationService DaoOrganizationService) UpdatePresident(height int64, contractAddress string, newPresident string) error {
	var org model.DaoOrganization
	filter := bson.M{
		"contract_address": contractAddress,
	}
	err := mongo.FindOne(mongo.CollectionDaoOrganization, filter, &org)
	if err != nil {
		return err
	}

	update := bson.M{
		"president": newPresident,
	}

	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoOrganization).UpdateOne(context.TODO(), filter, bson.M{"$set": update})

	err = rollbackService.Insert(mongo.CollectionDaoOrganization, org.ID, height, "president", org.President, newPresident)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoOrganization, err)
		return err
	}

	err = daoMessageService.SaveMessage(height, constant.MessageCategoryBePresident, constant.MessageTypeReadOnly, constant.MessagePositionWeb, contractAddress, newPresident, "{}")
	if err != nil {
		return err
	}

	err = daoMessageService.SaveMessage(height, constant.MessageCategoryBeenRemoved, constant.MessageTypeReadOnly, constant.MessagePositionWeb, contractAddress, org.President, "{}")
	if err != nil {
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["old_president"] = org.President
	additionalInfo["new_president"] = newPresident
	jsonStr, _ := json.Marshal(additionalInfo)
	err = daoMessageService.SaveMessage(height, constant.MessageCategoryChangePresident, constant.MessageTypeReadOnly, constant.MessagePositionDao, contractAddress, "", string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

// 遍历交易中是否有给组织打钱的交易
func (daoOrganizationService DaoOrganizationService) ReceiveAsset(height int64, rawTx []rpcjson.TxResult, vTx []rpcjson.TxResult) error {
	daoReceiveAssetSlice := make([]DaoReceiveAsset, 0)
	// 正常交易
	daoReceiveAssetSlice = daoOrganizationService.AssembleDaoReceiveAsset(rawTx, daoReceiveAssetSlice)

	// 虚拟交易
	daoReceiveAssetSlice = daoOrganizationService.AssembleDaoReceiveAsset(vTx, daoReceiveAssetSlice)
	// 记录消息
	for _, v := range daoReceiveAssetSlice {
		// 这个if条件排除了两种情况：
		// 1、dao组织发币时，senderAddress为空
		// 2、dao组织给另外一个dao组织转钱时，找零的交易senderAddress == dao组织地址
		if v.SenderAddress != "" && v.SenderAddress != v.ContractAddress {
			additionalInfo := make(map[string]interface{})
			additionalInfo["asset"] = v.Asset
			additionalInfo["amount"] = v.Amount
			additionalInfo["sender_address"] = v.SenderAddress
			jsonStr, _ := json.Marshal(additionalInfo)
			err := daoMessageService.SaveMessage(height, constant.MessageCategoryReceiveAsset, constant.MessageTypeReadOnly, constant.MessagePositionBoth, v.ContractAddress, "", string(jsonStr))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (daoOrganizationService DaoOrganizationService) AssembleDaoReceiveAsset(transaction []rpcjson.TxResult, daoReceiveAssetSlice []DaoReceiveAsset) []DaoReceiveAsset {
	for _, tx := range transaction {
		containFlag := false
		// 检查vout包含的地址是否是dao组织地址
		for i := 0; i < len(tx.Vout); i++ {
			v := tx.Vout[i]
			for _, address := range v.ScriptPubKey.Addresses {
				if _, ok := ContractAddressCache[address]; ok {
					if v.Value > 0 {
						containFlag = true
						daoReceiveAsset := DaoReceiveAsset{
							ContractAddress: address,
							SenderAddress:   "",
							Asset:           v.Asset,
							Amount:          v.Value,
						}
						daoReceiveAssetSlice = append(daoReceiveAssetSlice, daoReceiveAsset)
					}
				}
			}
		}

		if containFlag {
			for index, _ := range daoReceiveAssetSlice {
				for i := 0; i < len(tx.Vin); i++ {
					v := tx.Vin[i]
					if v.PrevOut != nil {
						daoReceiveAssetSlice[index].SenderAddress = strings.Join(v.PrevOut.Addresses, ",")
					}
				}
			}
		}
	}
	return daoReceiveAssetSlice
}
