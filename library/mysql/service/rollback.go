package service

import (
	"fmt"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mysql"
	models "github.com/AsimovNetwork/data-sync/library/mysql/model"
	"github.com/xormplus/xorm"
	"strings"
)

type RollbackService struct{}

func (rollbackService RollbackService) Insert(
	height int64,
	recordId int64,
	tableName string,
	originalValue int,
	expectValue int) error {
	rollback := models.TRollback{
		Id:            mysql.GlobalIdService.NextId(),
		Height:        height,
		RecordId:      recordId,
		TableName:     tableName,
		OriginalValue: originalValue,
		ExpectValue:   expectValue,
		CreateTime:    common.NowSecond(),
	}

	_, err := mysql.Engine.InsertOne(rollback)
	return err
}

func (rollbackService RollbackService) Rollback(height int64) error {
	session := mysql.Engine.NewSession()
	defer session.Close()

	// 本地开发、测试环境清数据时，删除所有
	err := dropAll(session, height)
	if err != nil {
		return err
	}

	//  1、获取需要回滚的数据
	list, err := list(height)
	if err != nil {
		return nil
	}
	if len(list) == 0 {
		return nil
	}

	// 遍历数据，依次操作对应的主表
	err = session.Begin()
	if err != nil {
		return err
	}
	for _, v := range list {
		err = modifyRecord(session, v)
		if err != nil {
			err = session.Rollback()
			return err
		}
	}

	// 删除rollback数据
	err = drop(session, height)
	if err != nil {
		err = session.Rollback()
		return err
	}
	// delete from t_dao_message
	err = dropMessage(session, height)
	if err != nil {
		err = session.Rollback()
		return err
	}

	return session.Commit()
}

func list(height int64) ([]models.TRollback, error) {
	list := make([]models.TRollback, 0)
	var sql []string
	sql = append(sql, "select a.* ")
	sql = append(sql, "from t_rollback a ")
	sql = append(sql, "inner join (")
	sql = append(sql, "select table_name,record_id,max(create_time) create_time ")
	sql = append(sql, "from t_rollback ")
	sql = append(sql, "group by table_name,record_id")
	sql = append(sql, ") b on a.table_name = b.table_name and a.record_id = b.record_id and a.create_time = b.create_time ")
	sql = append(sql, "where a.height <= ?")

	err := mysql.Engine.SQL(strings.Join(sql, ""), height).Find(&list)
	if err != nil {
		common.Logger.Error(err)
		return list, err
	}
	return list, nil
}

func modifyRecord(session *xorm.Session, rollback models.TRollback) error {
	sql := "update " + rollback.TableName + " set tx_status = ?, update_time = ? where id = ? "
	_, err := session.Exec(sql, rollback.ExpectValue, rollback.CreateTime, rollback.RecordId)
	return err
}

func drop(session *xorm.Session, height int64) error {
	_, err := session.Exec("delete from t_rollback where height > ?", height)
	return err
}

func dropMessage(session *xorm.Session, height int64) error {
	_, err := session.Exec("delete from t_dao_message where height > ?", height)
	return err
}

func dropAll(session *xorm.Session, height int64) error {
	// 测试环境清数据时清空所有mysql表
	if (common.Cfg.Env == common.ENV_DEVELOP_NET || common.Cfg.Env == common.ENV_TEST_NET) && height <= 0 {
		tables, err := mysql.Engine.DBMetas()
		if err != nil {
			panic(err)
		}
		for _, v := range tables {
			_, err := session.Exec(fmt.Sprintf("delete from %s ", v.Name))
			if err != nil {
				return err
			}
		}

		// 清数据之后t_dao_asset表插入主币的信息，避免系统shutdown
		insertSql := "insert into t_dao_asset(id, tx_hash, contract_address, asset, name, symbol, description, logo, asset_status, create_time, update_time) values (0, '', '', '000000000000000000000000', 'ASIM', 'ASIM', '','https://fingocn.oss-cn-hangzhou.aliyuncs.com/daoTest20/e9/eb/20e9eb87ad204cd0a8e261a71800fd63', 1, now(),now())"
		_, err = mysql.Engine.Exec(insertSql)
		if err != nil {
			return err
		}
	}

	return nil
}
