package mongo

import (
	"context"
	"fmt"
	"github.com/AsimovNetwork/data-sync/library/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

var MongoDB *mongo.Database

const (
	// ascan collection start
	CollectionBlock               = "block"
	CollectionTrading             = "trading"
	CollectionTransaction         = "transaction" // Sharding
	CollectionVirtualTransaction  = "virtual_transaction"
	CollectionAsset               = "asset"
	CollectionAssetIssue          = "asset_issue"
	CollectionTransactionCount    = "transaction_count"
	CollectionAddressAssetBalance = "address_asset_balance"
	// ascan collection end

	// validator collection begin
	CollectionValidator         = "validator"
	CollectionBtcMiner          = "btc_miner"
	CollectionValidatorRelation = "validator_relation"
	CollectionEarning           = "earning"
	CollectionEarningAsset      = "earning_asset"
	// validator collection end

	// foundation collection start
	CollectionFoundationBalanceSheet = "foundation_balance_sheet"
	CollectionFoundationMember       = "foundation_member"
	CollectionFoundationProposal     = "foundation_proposal"
	CollectionFoundationVote         = "foundation_vote"
	CollectionFoundationTodoList     = "foundation_todo_list"
	// foundation collection end

	// miner collection start
	CollectionMinerSignUp   = "miner_sign_up"
	CollectionMinerProposal = "miner_proposal"
	CollectionMinerVote     = "miner_vote"
	CollectionMinerRound    = "miner_round"
	CollectionMinerTodoList = "miner_todo_list"
	CollectionMinerMember   = "miner_member"
	// miner collection end

	// dao collection start
	CollectionDaoOrganization      = "dao_organization"
	CollectionDaoOrganizationAsset = "dao_organization_asset"
	CollectionDaoProposal          = "dao_proposal"
	CollectionDaoMember            = "dao_member"
	CollectionDaoTodoList          = "dao_todo_list"
	CollectionDaoVote              = "dao_vote"
	// dao collection end

	CollectionRollback         = "rollback"
	CollectionContractTemplate = "contract_template"
)

func init() {
	// Set client options
	clientOptions := options.Client().ApplyURI(common.Cfg.MongodbUrl)
	clientOptions.SetAuth(options.Credential{
		AuthMechanism: "SCRAM-SHA-1",
		AuthSource:    common.Cfg.MongodbDB,
		Username:      common.Cfg.MongodbUser,
		Password:      common.Cfg.MongodbPassword})

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		common.Logger.ErrorPanic("mongodb connect error: ", err)
	}
	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		common.Logger.ErrorPanic("mongodb ping error: ", err)
	}

	common.Logger.Info("connected to mongodb!")

	MongoDB = client.Database(common.Cfg.MongodbDB)

	// CollectionBlock Index
	ensureIndex(CollectionBlock, bsonx.Doc{{Key: "height", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionBlock, bsonx.Doc{{Key: "hash", Value: bsonx.Int32(1)}}, true)
	// CollectionBlock Index

	// Collection Address Index
	ensureIndex(CollectionTransactionCount, bsonx.Doc{{Key: "category", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionTransactionCount, bsonx.Doc{{Key: "key", Value: bsonx.Int32(1)}}, true)
	// Collection Address Index

	// CollectionTrading Index
	ensureIndex(CollectionTrading, bsonx.Doc{{Key: "time", Value: bsonx.Int32(1)}, {Key: "asset", Value: bsonx.Int32(1)}}, false)
	// CollectionTrading Index

	//  cannot create unique index over { hash: 1 } with shard key pattern { height: "hashed" }
	ensureIndex(CollectionTransaction, bsonx.Doc{{Key: "hash", Value: bsonx.Int32(1)}}, true)
	// cluster mod
	// ensureIndex(CollectionTransaction, bsonx.Doc{{Key: "hash", Value: bsonx.Int32(1)}}, false)
	// CollectionTransaction Index

	// Collection Asset Index
	ensureIndex(CollectionAsset, bsonx.Doc{{Key: "asset", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionAsset, bsonx.Doc{{Key: "name", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionAssetIssue, bsonx.Doc{{Key: "asset", Value: bsonx.Int32(1)}}, false)
	// Collection Asset Index

	// Validator Index
	ensureIndex(CollectionValidator, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionBtcMiner, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionValidatorRelation, bsonx.Doc{{Key: "btc_miner_address", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionValidatorRelation, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionEarning, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}, {Key: "time", Value: bsonx.Int32(-1)}}, false)
	ensureIndex(CollectionEarningAsset, bsonx.Doc{{Key: "asset", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionEarningAsset, bsonx.Doc{{Key: "time", Value: bsonx.Int32(1)}, {Key: "asset", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionEarningAsset, bsonx.Doc{{Key: "earning_id", Value: bsonx.Int32(1)}, {Key: "asset", Value: bsonx.Int32(1)}}, false)
	// Validator Index

	// Foundation Index
	ensureIndex(CollectionFoundationBalanceSheet, bsonx.Doc{{Key: "time", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionFoundationMember, bsonx.Doc{{Key: "in_service", Value: bsonx.Int32(1)}, {Key: "time", Value: bsonx.Int32(-1)}}, false)
	ensureIndex(CollectionFoundationMember, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionFoundationProposal, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionFoundationProposal, bsonx.Doc{{Key: "proposal_id", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionFoundationProposal, bsonx.Doc{{Key: "tx_hash", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionFoundationVote, bsonx.Doc{{Key: "tx_hash", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionFoundationVote, bsonx.Doc{{Key: "proposal_id", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionFoundationVote, bsonx.Doc{{Key: "voter", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionFoundationTodoList, bsonx.Doc{{Key: "operator", Value: bsonx.Int32(1)}, {Key: "proposal_type", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionFoundationTodoList, bsonx.Doc{{Key: "todo_id", Value: bsonx.Int32(1)}, {Key: "operator", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionAddressAssetBalance, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}, {Key: "asset", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionAddressAssetBalance, bsonx.Doc{{Key: "asset", Value: bsonx.Int32(1)}, {Key: "balance", Value: bsonx.Int32(1)}}, false)
	// Foundation Index

	ensureIndex(CollectionRollback, bsonx.Doc{{Key: "height", Value: bsonx.Int32(1)}, {Key: "collection", Value: bsonx.Int32(1)}, {Key: "field", Value: bsonx.Int32(1)}, {Key: "time", Value: bsonx.Int32(-1)}}, false)

	// Miner Index
	ensureIndex(CollectionMinerSignUp, bsonx.Doc{{Key: "round", Value: bsonx.Int32(1)}, {Key: "address", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionMinerSignUp, bsonx.Doc{{Key: "round", Value: bsonx.Int32(1)}, {Key: "produced", Value: bsonx.Int32(-1)}, {Key: "efficiency", Value: bsonx.Int32(-1)}}, false)
	ensureIndex(CollectionMinerProposal, bsonx.Doc{{Key: "proposal_id", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionMinerProposal, bsonx.Doc{{Key: "tx_hash", Value: bsonx.Int32(1)}, {Key: "status", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionMinerProposal, bsonx.Doc{{Key: "address", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionMinerProposal, bsonx.Doc{{Key: "effective_height", Value: bsonx.Int32(1)}, {Key: "status", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionMinerVote, bsonx.Doc{{Key: "tx_hash", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionMinerVote, bsonx.Doc{{Key: "proposal_id", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionMinerRound, bsonx.Doc{{Key: "round", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionMinerTodoList, bsonx.Doc{{Key: "round", Value: bsonx.Int32(1)}, {Key: "operator", Value: bsonx.Int32(1)}, {Key: "action_id", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionMinerTodoList, bsonx.Doc{{Key: "operator", Value: bsonx.Int32(1)}, {Key: "operated", Value: bsonx.Int32(1)}, {Key: "time", Value: bsonx.Int32(-1)}}, false)
	ensureIndex(CollectionMinerTodoList, bsonx.Doc{{Key: "operator", Value: bsonx.Int32(1)}, {Key: "operated", Value: bsonx.Int32(1)}, {Key: "action_type", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionMinerTodoList, bsonx.Doc{{Key: "action_id", Value: bsonx.Int32(1)}}, false)
	ensureIndex(CollectionMinerMember, bsonx.Doc{{Key: "round", Value: bsonx.Int32(1)}, {Key: "address", Value: bsonx.Int32(1)}}, true)
	ensureIndex(CollectionMinerMember, bsonx.Doc{{Key: "round", Value: bsonx.Int32(1)}, {Key: "produced", Value: bsonx.Int32(1)}}, false)
}

func ensureIndex(collection string, keys bsonx.Doc, unique bool) {
	index := mongo.IndexModel{}
	index.Keys = keys
	if unique {
		indexOption := options.IndexOptions{
			Unique: &unique,
		}
		index.Options = &indexOption
	}

	block := MongoDB.Collection(collection)
	indexName, err := block.Indexes().CreateOne(context.TODO(), index)
	if err != nil {
		common.Logger.ErrorPanic(fmt.Sprintf("mongodb create index on %s error: ", collection), err)
	}
	common.Logger.Infof("create mongodb index %s on %s", indexName, collection)
}
