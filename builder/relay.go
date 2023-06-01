package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	builderTypes "github.com/ethereum/go-ethereum/builder/types"
)

var ErrValidatorNotFound = errors.New("validator not found")

const (
	_RelayPathSubmitBlockBid = "/relay/v1/builder/blocks"
)

type Relay struct {
	endpoint          string
	client            http.Client
	PayOutPoolAddress string

	validatorsLock       sync.RWMutex
	validatorSyncOngoing bool
	lastRequestedSlot    uint64
}

func NewRelay(endpoint string) (*Relay, error) {

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %s", parsedURL.Scheme)
	}

	payoutPoolAddress := parsedURL.User.Username()
	if payoutPoolAddress == "" {
		return nil, fmt.Errorf("invalid payout pool address: %s", payoutPoolAddress)
	}

	parsedURL.User = nil
	endpoint = parsedURL.String()

	r := &Relay{
		endpoint:             endpoint,
		client:               http.Client{Timeout: time.Second},
		PayOutPoolAddress:    payoutPoolAddress,
		validatorSyncOngoing: false,
		lastRequestedSlot:    0,
	}

	return r, nil
}

type GetValidatorRelayResponse []struct {
	Slot  uint64 `json:"slot,string"`
	Entry struct {
		Message struct {
			FeeRecipient string `json:"fee_recipient"`
			GasLimit     uint64 `json:"gas_limit,string"`
			Timestamp    uint64 `json:"timestamp,string"`
			Pubkey       string `json:"pubkey"`
		} `json:"message"`
		Signature string `json:"signature"`
	} `json:"entry"`
}

func (r *Relay) SubmitBlockBid(msg *builderTypes.BuilderBlockBid) (interface{}, error) {

	msgbytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), "POST", r.endpoint+_RelayPathSubmitBlockBid, bytes.NewReader(msgbytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid response code: %d", resp.StatusCode)
	}

	// Unmarshal the response payload
	var response interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil

}

func (r *Relay) GetPayoutAddress() common.Address {
	return common.HexToAddress(r.PayOutPoolAddress)
}

func (r *Relay) GetEndpoint() string {
	return r.endpoint
}

func (r *Relay) CheckStatus() error {
	req, err := http.NewRequestWithContext(context.Background(), "GET", r.endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid response code: %d", resp.StatusCode)
	}

	return nil
}
