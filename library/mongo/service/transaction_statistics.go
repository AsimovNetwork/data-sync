package service

import (
	"context"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo"
	"github.com/AsimovNetwork/data-sync/library/mongo/model"
	"github.com/AsimovNetwork/data-sync/library/response"
	"go.mongodb.org/mongo-driver/bson"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
)

type TransactionStatisticsService struct{}

func (transactionStatisticsService TransactionStatisticsService) Exist(key string) (bool, error) {
	exist, err := mongo.Exist(mongo.CollectionTransactionCount, bson.M{"key": key})
	return exist, err
}

func (transactionStatisticsService TransactionStatisticsService) Record(collection string, models []interface{}) error {
	if len(models) <= 0 {
		return nil
	}
	_, err := mongo.MongoDB.Collection(collection).InsertMany(context.TODO(), models)
	if err != nil {
		common.Logger.Errorf("insert %s error. err: %s", collection, err)
		return err
	}
	return nil
}

func (transactionStatisticsService TransactionStatisticsService) Drop(collection string, category uint8) error {
	txCountFilter := bson.M{
		"tx_count": bson.M{
			"$gt": 1000,
		},
		"category": category,
	}
	txCounts, err := mongo.Find(mongo.CollectionTransactionCount, txCountFilter, reflect.TypeOf(model.TransactionCount{}), reflect.TypeOf(&model.TransactionCount{}))
	if err != nil {
		return err
	}
	for _, txCount := range txCounts.([]*model.TransactionCount) {
		contractTxFilter := bson.M{
			"key": txCount.Key,
		}
		skip := int64(1000)
		// find the 1000th record
		findOption := options.FindOneOptions{
			Skip: &skip,
			Sort: bson.M{"_id": -1},
		}
		var result model.TransactionList
		err := mongo.FindOne(collection, contractTxFilter, &result, &findOption)
		if err != nil {
			if !response.IsDataNotExistError(err) {
				return err
			}
		}
		deleteFilter := bson.M{
			"key": txCount.Key,
			"_id": bson.M{
				"$lte": result.ID,
			},
		}
		_, err = mongo.MongoDB.Collection(collection).DeleteMany(context.TODO(), deleteFilter)
		if err != nil {
			common.Logger.Errorf("delete from %s error. err: %s", collection, err)
			return err
		}
	}
	return nil
}

func (transactionStatisticsService TransactionStatisticsService) InsertOrUpdate(category uint8, transactionTxCountSlice []model.TransactionCount) error {
	var operations []mongoDriver.WriteModel
	for _, v := range transactionTxCountSlice {
		filter := bson.M{
			"key": v.Key,
		}
		insertOrUpdate := bson.M{}
		if category == model.CountContract {
			insertOrUpdate = bson.M{
				"$setOnInsert": bson.M{
					"category":      category,
					"tx_hash":       v.TxHash,
					"time":          v.Time,
					"creator":       v.Creator,
					"template_type": v.TemplateType,
					"template_name": v.TemplateTName,
				},
				"$inc": bson.M{"tx_count": 1},
			}
		} else {
			insertOrUpdate = bson.M{
				"$setOnInsert": bson.M{
					"category": category,
				},
				"$inc": bson.M{"tx_count": 1},
			}
		}

		operation := mongoDriver.NewUpdateOneModel()
		operation.SetFilter(filter)
		operation.SetUpdate(insertOrUpdate)
		operation.SetUpsert(true)
		operations = append(operations, operation)
	}
	if len(operations) > 0 {
		_, err := mongo.MongoDB.Collection(mongo.CollectionTransactionCount).BulkWrite(
			context.Background(),
			operations,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
