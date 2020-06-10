package service

import (
	"context"
	"encoding/json"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	"go.mongodb.org/mongo-driver/bson"
)

type DaoProposalService struct{}

func (daoProposalService DaoProposalService) Insert(height int64, time int64, txHash string, contractAddress string, endTime int64, proposalId int64, proposalType uint8) error {
	var org model.DaoOrganization
	filter := bson.M{
		"contract_address": contractAddress,
	}
	err := mongo.FindOne(mongo.CollectionDaoOrganization, filter, &org)
	if err != nil {
		return err
	}
	insert := model.DaoProposal{
		Height:          height,
		Time:            time,
		TxHash:          txHash,
		EndTime:         endTime,
		ContractAddress: contractAddress,
		ProposalId:      proposalId,
		Address:         org.President,
		ProposalType:    proposalType,
		Status:          model.ProposalStatusOnGoing,
	}

	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoProposal).InsertOne(context.TODO(), insert)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", mongo.CollectionDaoProposal, err)
		return err
	}

	return nil
}

func (daoProposalService DaoProposalService) Update(height int64, contractAddress string, proposalId int64, status uint8) error {
	var proposal model.DaoProposal
	filter := bson.M{
		"contract_address": contractAddress,
		"proposal_id":      proposalId,
	}
	err := mongo.FindOne(mongo.CollectionDaoProposal, filter, &proposal)
	if err != nil {
		return err
	}

	update := bson.M{
		"status": status,
	}
	_, err = mongo.MongoDB.Collection(mongo.CollectionDaoProposal).UpdateOne(context.TODO(), filter, bson.M{"$set": update})
	if err != nil {
		common.Logger.Errorf("change dao proposal status error. err: %s", err)
		return err
	}

	err = rollbackService.Insert(mongo.CollectionDaoProposal, proposal.ID, height, "status", proposal.Status, status)
	if err != nil {
		common.Logger.Errorf("insert rollback error. rollback collection: %s, err: %s", mongo.CollectionDaoProposal, err)
		return err
	}

	additionalInfo := make(map[string]interface{})
	additionalInfo["proposal_id"] = proposalId
	additionalInfo["proposal_type"] = model.ProposalTypeIssueAsset
	jsonStr, _ := json.Marshal(additionalInfo)

	if status == model.ProposalStatusReject {
		err = daoMessageService.SaveMessage(height, constant.MessageCategoryProposalRejected, constant.MessageTypeReadOnly, constant.MessagePositionBoth, contractAddress, "", string(jsonStr))
		if err != nil {
			return err
		}
	}

	return nil
}
