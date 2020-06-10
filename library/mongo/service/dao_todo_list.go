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
	"reflect"
)

type DaoTodoListService struct{}

func (daoTodoListService DaoTodoListService) Insert(height int64, time int64, contractAddress string, operator string, todoType uint8, endTime int64, inviteRole uint8) error {
	insert := model.DaoTodoList{
		Height:          height,
		Time:            time,
		ContractAddress: contractAddress,
		Operator:        operator,
		TodoType:        todoType,
		EndTime:         endTime,
		Operated:        false,
	}

	_, err := mongo.MongoDB.Collection(mongo.CollectionDaoTodoList).InsertOne(context.TODO(), insert)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoTodoList, err)
		return err
	}

	var org model.DaoOrganization
	filter := bson.M{
		"contract_address": contractAddress,
	}
	err = mongo.FindOne(mongo.CollectionDaoOrganization, filter, &org)
	if err != nil {
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["invite_role"] = inviteRole
	jsonStr, _ := json.Marshal(additionalInfo)
	err = daoMessageService.SaveMessage(height, constant.MessageCategoryInvited, constant.MessageTypeReadOnly, constant.MessagePositionWeb, contractAddress, operator, string(jsonStr))
	if err != nil {
		return err
	}

	return nil
}

func (daoTodoListService DaoTodoListService) InsertMany(height int64, time int64, contractAddress string, todoType uint8, endTime int64, todoId int64) error {
	inserts := make([]interface{}, 0)
	filter := bson.M{
		"contract_address": contractAddress,
		"status":           model.MemberStatusAgreed,
	}
	members, err := mongo.Find(mongo.CollectionDaoMember, filter, reflect.TypeOf(model.DaoMember{}), reflect.TypeOf(&model.DaoMember{}))
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoTodoList, err)
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["proposal_id"] = todoId
	jsonStr, _ := json.Marshal(additionalInfo)

	for _, v := range members.([]*model.DaoMember) {
		insert := model.DaoTodoList{
			Height:          height,
			Time:            time,
			ContractAddress: contractAddress,
			Operator:        v.Address,
			TodoType:        todoType,
			TodoId:          todoId,
			EndTime:         endTime,
			Operated:        false,
		}
		inserts = append(inserts, insert)

		err = daoMessageService.SaveMessage(height, constant.MessageCategoryNewVote, constant.MessageTypeVote, constant.MessagePositionWeb, contractAddress, v.Address, string(jsonStr))
		if err != nil {
			return err
		}
	}

	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoTodoList).InsertMany(context.TODO(), inserts)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoTodoList, err)
		return err
	}
	return nil
}

func (daoTodoListService DaoTodoListService) Release(height int64, contractAddress string, memberAddress string, todoType uint8) error {
	var todo model.DaoTodoList
	filter := bson.M{
		"contract_address": contractAddress,
		"operator":         memberAddress,
		"todo_type":        todoType,
		"operated":         false,
	}

	err := mongo.FindOne(mongo.CollectionDaoTodoList, filter, &todo)
	if err != nil {
		if response.IsDataNotExistError(err) {
			common.Logger.Infof("can't find one result from dao_todo_list with filter: %v", filter)
			return nil
		}
		return err
	}

	update := bson.M{
		"operated": true,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoTodoList).UpdateOne(context.TODO(), filter, bson.M{"$set": update})
	if err != nil {
		return err
	}

	err = rollbackService.Insert(mongo.CollectionDaoTodoList, todo.ID, height, "operated", false, true)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoTodoList, err)
		return err
	}

	return nil
}

func (daoTodoListService DaoTodoListService) ReleaseById(height int64, contractAddress string, memberAddress string, todoType uint8, voteId int64) error {
	var todo model.DaoTodoList
	filter := bson.M{
		"contract_address": contractAddress,
		"operator":         memberAddress,
		"todo_type":        todoType,
		"todo_id":          voteId,
		"operated":         false,
	}

	err := mongo.FindOne(mongo.CollectionDaoTodoList, filter, &todo)
	if err != nil {
		return err
	}

	update := bson.M{
		"operated": true,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoTodoList).UpdateOne(context.TODO(), filter, bson.M{"$set": update})
	if err != nil {
		return err
	}

	err = rollbackService.Insert(mongo.CollectionDaoTodoList, todo.ID, height, "operated", false, true)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoTodoList, err)
		return err
	}

	return nil
}
