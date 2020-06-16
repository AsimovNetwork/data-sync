package main

import (
	"fmt"
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/AsimovNetwork/data-sync/library/mongo/event"
	"github.com/AsimovNetwork/data-sync/library/mongo/service"
	"github.com/AsimovNetwork/data-sync/library/response"

	mysqlService "github.com/AsimovNetwork/data-sync/library/mysql/service"

	"time"
)

var blockService = service.BlockService{}
var transactionService = service.TransactionService{}
var ecologyService = service.EcologyService{}
var eventLogService = event.EventLogService{}
var updatableService = mysqlService.UpdatableService{}
var assetService = service.AssetService{
	AssetInfo: mysqlService.GetAsset,
}
var minerProposalService = service.MinerProposalService{}
var rollbackService = service.RollbackService{}
var mysqlRollbackService = mysqlService.RollbackService{}
var initService = service.InitService{
	HeightInitHandler:     []service.HeightInitService{assetService},
	ProjectStartupHandler: []service.ProjectStartupService{},
}
var chainNodeService = mysqlService.ChainNodeService{}
var daoOrganizationService = service.DaoOrganizationService{}
var transactionStatisticsService = service.TransactionStatisticsService{}

func main() {
	// Get current synchronized block height
	handledHeight, err := blockService.GetHandledBlockHeight()
	if err != nil {
		if response.IsDataNotExistError(err) {
			handledHeight = -1
		} else {
			common.Logger.ErrorPanic("get handled block height failed. ", err)
		}
	}

	// Business data initialization
	err = initService.Init(int32(handledHeight) + 1)
	if err != nil {
		common.Logger.ErrorPanic("project startup failed. ", err)
	}

	// Use a single thread to synchronize data
	done := make(chan *int64, 1)
	done <- &handledHeight
	for {
		select {
		case data := <-done:
			go func() {
				var currentHandleHeight int64
				// If data is nil, get current synchronized block height
				if data == nil {
					common.Logger.Info("already newest, sleep 5 seconds")
					time.Sleep(time.Duration(5) * time.Second)

					handledHeight, err := blockService.GetHandledBlockHeight()
					if err != nil {
						common.Logger.Error("get handled block failed, loop => done == nil")
						done <- nil
						return
					}
					currentHandleHeight = handledHeight
				} else {
					// Otherwise, currentHandleHeight = data
					currentHandleHeight = *data
				}

				// Get best block height from chain
				bestHeight, err := blockService.GetBestBlockHeight()
				if err != nil {
					common.Logger.Error("get best block failed, loop => done == nil")
					done <- nil
					return
				}

				if int32(currentHandleHeight) == bestHeight {
					done <- nil
					return
				} else if int32(currentHandleHeight) > bestHeight {
					shouldReturnedHeight := calculateShouldReturnedHeight(bestHeight)

					// Handle zero bound value, rollback to the previous block of the correct block
					prevCorrectHeight := shouldReturnedHeight
					if prevCorrectHeight > -1 {
						prevCorrectHeight = shouldReturnedHeight - 1
					}

					// rollback block
					err = rollback(prevCorrectHeight)
					if err != nil {
						common.SendDingTalk(fmt.Sprintf("roll back to %d failed, system shutdown", prevCorrectHeight))
						common.Logger.ErrorfPanic(err, "roll back to %d failed", prevCorrectHeight)
					}
					var lastCorrectHandledHeight = int64(prevCorrectHeight)
					done <- &lastCorrectHandledHeight
					return
				} else {
					// Check whether last correctly synchronized block is currentHandleHeight
					lastCorrectHeight := calculateShouldReturnedHeight(int32(currentHandleHeight))
					if lastCorrectHeight != int32(currentHandleHeight) {
						common.Logger.Errorf("handled block is not math remote, last correct: %d, handled: %d", lastCorrectHeight, currentHandleHeight)

						prevCorrectHeight := lastCorrectHeight
						// Handle zero bound value, rollback to the previous block of the correct block
						if prevCorrectHeight > -1 {
							prevCorrectHeight = prevCorrectHeight - 1
						}

						err = rollback(prevCorrectHeight)
						if err != nil {
							common.SendDingTalk(fmt.Sprintf("roll back to %d failed, system shutdown.", prevCorrectHeight))
							common.Logger.ErrorfPanic(err, "roll back to %d failed, system shutdown", prevCorrectHeight)
						}
						var lastCorrectHandledHeight = int64(prevCorrectHeight)
						done <- &lastCorrectHandledHeight
						return
					}

					var nextHeight int64
					nextHeight = currentHandleHeight + 1
					handledHeight, err := syncData(int32(nextHeight), 10)
					if err != nil {
						common.Logger.Errorf("sync height %d failed, err: %v", *handledHeight, err)
						common.SendDingTalk(fmt.Sprintf("sync height %d failed, err: %v", *handledHeight, err))
						// the previous block of the correct block
						// var lastCorrectHandledHeight = *handledHeight - 1
						var lastCorrectHandledHeight = *handledHeight - 1
						// Handle zero bound value, rollback to the previous block of the correct block
						if lastCorrectHandledHeight > -1 {
							lastCorrectHandledHeight = lastCorrectHandledHeight - 1
						}
						// rollback, go back to the previous block where synchronization failed
						err = rollback(int32(lastCorrectHandledHeight))
						if err != nil {
							common.SendDingTalk(fmt.Sprintf("roll back to %d failed, system shutdown", lastCorrectHandledHeight))
							common.Logger.ErrorfPanic(err, "roll back to %d failed, system shutdown", lastCorrectHandledHeight)
						}
						done <- &lastCorrectHandledHeight
						return
					}
					done <- handledHeight
					return
				}
			}()
		}
	}
}

func syncData(offset int32, count int32) (*int64, error) {
	var handledHeight = int64(offset)
	blocks, err := blockService.FetchBlocks(offset, count)
	if err != nil {
		if response.IsCallBlockChainError(err) {
			var currentHeight = int64(offset - 1)
			common.SendDingTalk(fmt.Sprintf("sync height %d failed, err: %v", handledHeight, err))
			common.Logger.Errorf("sync height %d failed, err: %v", handledHeight, err)
			time.Sleep(time.Duration(5) * time.Second)
			return &currentHeight, nil
		}
		return &handledHeight, err
	}
	// var handledHeight *int64 = nil
	for i := 0; i < len(blocks); i++ {
		handledHeight = blocks[i].Height
		common.Logger.Infof("handle block %d start", blocks[i].Height)

		// 1. Handle event log
		err = eventLogService.HandleEventLog(blocks[i].Height, blocks[i].Time, blocks[i].Receipts)
		if err != nil {
			return &handledHeight, err
		}

		// 2. Save transaction
		err = transactionService.Insert(blocks[i].Hash, blocks[i].Height, blocks[i].Time, blocks[i].RawTx, blocks[i].Vtxs)
		if err != nil {
			return &handledHeight, err
		}

		// 3、Drop collection data before one day
		if handledHeight%720 == 0 {
			go func() {
				err := ecologyService.DropOneDayBeforeData()
				if err != nil {
					common.Logger.Errorf("drop trading data error in height %d,err: %s", handledHeight, err)
				}
			}()
		}

		// 4. Statistics official website ecological data
		err = ecologyService.Analyze(blocks[i])
		if err != nil {
			return &handledHeight, err
		}

		// 5. Update mysql transaction status via receipt
		// TODO notify sync routine when update failed
		go func() {
			err := updatableService.HandleUpdateStatus(blocks[i].Height, blocks[i].Receipts)
			if err != nil {
				common.SendDingTalk(fmt.Sprintf("update mysql tx_status failed, height: %d", blocks[i].Height))
			}
		}()

		// 6. Save asset information
		err = assetService.Insert(blocks[i].Hash, blocks[i].Height, blocks[i].RawTx, blocks[i].Vtxs)
		if err != nil {
			return &handledHeight, err
		}

		// 7. Update proposal status
		err = minerProposalService.UpdateStatusByHeight(blocks[i].Height)
		if err != nil {
			return &handledHeight, err
		}

		// 8. Get all nodes
		chainNodeService.Insert(blocks[i].Height)

		// 9. Monitor transfer transaction of dao organization received
		err = daoOrganizationService.ReceiveAsset(blocks[i].Height, blocks[i].RawTx, blocks[i].Vtxs)
		if err != nil {
			return &handledHeight, err
		}

		// 10、Save block
		_, err := blockService.Insert(blocks[i])
		if err != nil {
			return &handledHeight, err
		}

		common.Logger.Infof("handle block %d end", blocks[i].Height)
	}
	return &handledHeight, nil
}

// Rollback to block n, [0,n]
func rollback(height int32) error {
	// Current synchronized block height
	handledHeight, _ := blockService.GetHandledBlockHeight()

	// Send rollback message to ding talk
	common.SendDingTalk(fmt.Sprintf("rollback to %d from local %d", height, handledHeight))
	common.Logger.Infof("rollback to %d from local %d", height, handledHeight)

	// rollback
	err := rollbackService.Rollback(int64(height))
	if err != nil {
		return err
	}
	err = mysqlRollbackService.Rollback(int64(height))
	if err != nil {
		return err
	}

	// Initialize
	err = initService.Init(height + 1)
	if err != nil {
		return err
	}
	return nil
}

func calculateShouldReturnedHeight(bestHeight int32) int32 {
	var shouldReturnedHeight int32 = -1
	for i := bestHeight; i >= 0; i-- {
		localBlock, err := blockService.GetBlockByHeight(int64(i))
		if err != nil {
			common.SendDingTalk(fmt.Sprintf("calculate should returned height %d failed, system shutdown", i))
			common.Logger.ErrorfPanic(err, "calculate should returned height %d failed, system shutdown", i)
		}
		remoteBlock, err := blockService.FetchBlocks(i, 1)
		if err != nil || len(remoteBlock) != 1 {
			common.SendDingTalk(fmt.Sprintf("calculate should returned height %d failed, system shutdown", i))
			common.Logger.ErrorfPanic(err, "calculate should returned height %d failed, system shutdown", i)
		}

		if localBlock.Hash == remoteBlock[0].Hash {
			shouldReturnedHeight = i
			break
		}
	}

	if shouldReturnedHeight != bestHeight {
		common.Logger.Errorf("calculate should returned height from %d to %d", bestHeight, shouldReturnedHeight)
	}
	return shouldReturnedHeight
}
