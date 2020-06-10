package service

import (
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mysql"
	"github.com/AsimovNetwork/data-sync/library/mysql/constant"
	models "github.com/AsimovNetwork/data-sync/library/mysql/model"
)

type DaoMessageService struct{}

func (daoMessageService DaoMessageService) SaveMessage(height int64, category int, messageType int, messagePosition int, contractAddress string, receiver string, additionalInfo string) error {
	now := common.NowSecond()
	message := models.TDaoMessage{
		Id:              mysql.GlobalIdService.NextId(),
		Category:        category,
		Type:            messageType,
		MessagePosition: messagePosition,
		ContractAddress: contractAddress,
		Receiver:        receiver,
		AdditionalInfo:  additionalInfo,
		State:           constant.MessageStateUnread,
		CreateTime:      now,
		UpdateTime:      now,
	}
	_, err := mysql.Engine.InsertOne(message)
	return err
}
