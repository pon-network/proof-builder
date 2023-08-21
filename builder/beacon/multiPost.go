package multibeaconClient

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	capella "github.com/attestantio/go-eth2-client/spec/capella"

	"github.com/ethereum/go-ethereum/builder/database"
	"github.com/ethereum/go-ethereum/log"
)

func (b *MultiBeaconClient) PublishBlock(ctx context.Context, block capella.SignedBeaconBlock, metricsEnabled bool, db *database.DatabaseService) (err error) {
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
	var successfulCount int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-submissionError:
			// Channel receives a response from each client, so increment the response count
			responseCount++
			switch e {
			case nil:
				// Successful submission, so increment the successful count
				successfulCount++
				if responseCount == len(b.Clients) {
					// All clients have responded, so return the error (if any)
					// There has been at least one successful submission,
					// since succesfulCount has definitely been incremented in this case
					log.Info("Successfully submitted block to beacon chain", "successes", successfulCount, "failures", len(b.Clients)-successfulCount)
					err = nil

					if metricsEnabled {
						// There was at least one successful submission, don't record an error
						submissionErr := sql.NullString{
							String: "",
							Valid:  false,
						}

						signedBeaconBlock := database.SignedBeaconBlockSubmissionEntry{
							InsertedAt:        time.Now(),
							Signature:         block.Signature.String(),
							SignedBeaconBlock: block.Message.String(),
							SubmittedToChain:  true,
							SubmissionError:   submissionErr,
						}
						go db.InsertBeaconBlock(signedBeaconBlock, block.Message.Body.ExecutionPayload.BlockHash.String())
					}

					return err
				}

			default:
				// Error received, so set the error and continue
				err = e
				if responseCount == len(b.Clients) {
					// All clients have responded, so return the error (if any)

					if successfulCount == 0 {
						if err == nil {
							err = errors.New("failed to submit block to any clients")
						}
					} else {
						log.Info("Successfully submitted block to beacon chain", "successes", successfulCount, "failures", len(b.Clients)-successfulCount)
						err = nil
					}

					if metricsEnabled {
						var submissionErr sql.NullString
						if err == nil && successfulCount > 0 {
							// If there was at least one successful submission, then don't record the error
							submissionErr = sql.NullString{
								String: "",
								Valid:  false,
							}
						} else {
							submissionErr = sql.NullString{
								String: err.Error(),
								Valid:  err != nil,
							}
						}

						signedBeaconBlock := database.SignedBeaconBlockSubmissionEntry{
							InsertedAt:        time.Now(),
							Signature:         block.Signature.String(),
							SignedBeaconBlock: block.Message.String(),
							SubmittedToChain:  successfulCount > 0,
							SubmissionError:   submissionErr,
						}
						go db.InsertBeaconBlock(signedBeaconBlock, block.Message.Body.ExecutionPayload.BlockHash.String())
					}

					return err
				}
			}
		}
	}
}

func publishAsync(ctx context.Context, clientUpdate *sync.Mutex, client BeaconClient, block capella.SignedBeaconBlock, submissionError chan<- error) {

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
