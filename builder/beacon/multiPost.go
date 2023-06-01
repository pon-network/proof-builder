package multibeaconClient

import (
	"context"
	"sync"
	"time"
	"database/sql"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	"github.com/ethereum/go-ethereum/builder/database"
	"github.com/ethereum/go-ethereum/log"
)

func (b *MultiBeaconClient) PublishBlock(ctx context.Context, block beaconTypes.SignedBeaconBlock, metricsEnabled bool, db *database.DatabaseService) (err error) {
	/*
		Post a block to beacon chain using all clients
		No penalty for multiple submissions
		Increased reliability in case of some clients being down
	*/
	defer b.postBeaconCall()
	// Create a channel to receive errors from the clients
	submissionError := make(chan error, len(b.Clients))

	for _, client := range b.Clients {
		// Have all clients publish the block asynchronously
		go publishAsync(ctx, &b.clientUpdate, client, block, submissionError)
	}

	var responseCount int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-submissionError:
			// Channel receives a response from each client, so increment the response count
			responseCount++
			switch e {
			case nil:
				// Successful submission, so error is nil
				return nil
			default:
				// Error received, so set the error and continue
				err = e
				if responseCount == len(b.Clients) {
					// All clients have responded, so return the error (if any)
					if metricsEnabled {
						signedBeaconBlock := database.SignedBeaconBlockSubmissionEntry{
							InsertedAt:        time.Now(),
							Signature:         block.Signature.String(),
							SignedBeaconBlock: block.Message.String(),
							SubmittedToChain:  err == nil,
							SubmissionError: sql.NullString{
								String: err.Error(),
								Valid:  err != nil,
							},
						}
						go db.InsertBeaconBlock(signedBeaconBlock, block.Message.Body.ExecutionPayload.BlockHash.String())
					}

					return err
				}
			}
		}
	}
}

func publishAsync(ctx context.Context, clientUpdate *sync.Mutex, client BeaconClient, block beaconTypes.SignedBeaconBlock, submissionError chan<- error) {
	err := client.Node.PublishBlock(ctx, block)
	if err != nil {
		log.Warn("failed to publish block", "err", err, "endpoint", client.Node.BaseEndpoint())
		clientUpdate.Lock()
		client.LastResponseStatus = 500
		client.LastUsedTime = time.Now()
		clientUpdate.Unlock()
	}
	clientUpdate.Lock()
	client.LastResponseStatus = 200
	client.LastUsedTime = time.Now()
	clientUpdate.Unlock()
	submissionError <- err
}
