package service

import (
	"context"
	"encoding/json"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	"github.com/AsimovNetwork/data-sync/library/response"
	"gopkg.in/mgo.v2/bson"
)

type DaoMemberService struct{}

func (daoMemberService DaoMemberService) Insert(height int64, time int64, txHash string, contractAddress string, role uint8, address string, status uint8) error {
	var member model.DaoMember
	filter := bson.M{
		"contract_address": contractAddress,
		"address":          address,
	}

	err := mongo.FindOne(mongo.CollectionFoundationMember, filter, &member)
	if err == nil {
		update := bson.M{
			"status": model.MemberStatusInvited,
		}

		_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).UpdateOne(context.TODO(), filter, update)

		err = rollbackService.Insert(mongo.CollectionDaoMember, member.ID, height, "status", member.Status, status)
		if err != nil {
			common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoMember, err)
			return err
		}

	} else if response.IsDataNotExistError(err) {
		insert := model.DaoMember{
			Height:          height,
			Time:            time,
			ContractAddress: contractAddress,
			Role:            role,
			Address:         address,
			Status:          status,
		}
		_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).InsertOne(context.TODO(), insert)

	} else {
		return err
	}

	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoMember, err)
		return err
	}
	return nil
}

func (daoMemberService DaoMemberService) InsertMany(height int64, time int64, contractAddress string, members []string) error {
	inserts := make([]interface{}, 0)
	for _, v := range members {
		insert := model.DaoMember{
			Height:          height,
			Time:            time,
			ContractAddress: contractAddress,
			Role:            model.MemberRoleOrdinary,
			Address:         v,
			Status:          model.MemberStatusAgreed,
		}
		inserts = append(inserts, insert)
	}

	_, err := mongo.MongoDB.Collection(mongo.CollectionDaoMember).InsertMany(context.TODO(), inserts)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoMember, err)
		return err
	}

	return nil
}

func (daoMemberService DaoMemberService) Update(height int64, contractAddress string, address string, status uint8) error {
	var member model.DaoMember
	filter := bson.M{
		"contract_address": contractAddress,
		"address":          address,
	}
	err := mongo.FindOne(mongo.CollectionDaoMember, filter, &member)
	if err != nil {
		return err
	}

	update := bson.M{
		"status": status,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).UpdateOne(context.TODO(), filter, bson.M{"$set": update})

	err = rollbackService.Insert(mongo.CollectionDaoMember, member.ID, height, "status", member.Status, status)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoMember, err)
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["target_address"] = address
	jsonStr, _ := json.Marshal(additionalInfo)
	if status == model.MemberStatusAgreed {
		err = daoMessageService.SaveMessage(height, constant.MessageCategoryBeMember, constant.MessageTypeReadOnly, constant.MessagePositionWeb, contractAddress, address, string(jsonStr))
		if err != nil {
			return err
		}

		err = daoMessageService.SaveMessage(height, constant.MessageCategoryAddNewMember, constant.MessageTypeReadOnly, constant.MessagePositionDao, contractAddress, "", string(jsonStr))
		if err != nil {
			return err
		}

	} else if status == model.MemberStatusRemoved {
		err = daoMessageService.SaveMessage(height, constant.MessageCategoryBeenRemoved, constant.MessageTypeReadOnly, constant.MessagePositionWeb, contractAddress, address, string(jsonStr))
		if err != nil {
			return err
		}

		err = daoMessageService.SaveMessage(height, constant.MessageCategoryRemoveMember, constant.MessageTypeReadOnly, constant.MessagePositionDao, contractAddress, "", string(jsonStr))
		if err != nil {
			return err
		}
	}

	return nil
}

func (daoMemberService DaoMemberService) UpdatePresident(height int64, time int64, txHash string, contractAddress string, address string) error {
	var member model.DaoMember
	filter := bson.M{
		"contract_address": contractAddress,
		"address":          address,
	}
	err := mongo.FindOne(mongo.CollectionDaoMember, filter, &member)
	if err != nil {
		if !response.IsDataNotExistError(err) {
			return err
		} else {
			insert := model.DaoMember{
				Height:          height,
				Time:            time,
				ContractAddress: contractAddress,
				Role:            model.MemberRolePresident,
				Address:         address,
				Status:          model.MemberStatusAgreed,
			}
			_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).InsertOne(context.TODO(), insert)
		}
	}

	update := bson.M{
		"role":   model.MemberRolePresident,
		"status": model.MemberStatusAgreed,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).UpdateOne(context.TODO(), filter, bson.M{"$set": update})

	err = rollbackService.Insert(mongo.CollectionDaoMember, member.ID, height, "role", member.Role, model.MemberRolePresident)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoMember, err)
		return err
	}

	if member.Status != model.MemberStatusAgreed {
		err = rollbackService.Insert(mongo.CollectionDaoMember, member.ID, height, "status", member.Status, model.MemberStatusAgreed)
		if err != nil {
			common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoMember, err)
			return err
		}
	}

	return nil
}

func (daoMemberService DaoMemberService) RemovePresident(height int64, contractAddress string) error {
	var org model.DaoOrganization
	filter := bson.M{
		"contract_address": contractAddress,
	}
	err := mongo.FindOne(mongo.CollectionDaoOrganization, filter, &org)
	if err != nil {
		return err
	}

	var member model.DaoMember
	filterMember := bson.M{
		"contract_address": contractAddress,
		"address":          org.President,
	}
	err = mongo.FindOne(mongo.CollectionDaoMember, filterMember, &member)
	if err != nil {
		return err
	}

	update := bson.M{
		"status": model.MemberStatusRemoved,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoMember).UpdateOne(context.TODO(), filterMember, bson.M{"$set": update})

	err = rollbackService.Insert(mongo.CollectionDaoMember, member.ID, height, "status", member.Status, model.MemberStatusRemoved)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoMember, err)
		return err
	}

	return nil
}
