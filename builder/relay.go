package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
)

var ErrValidatorNotFound = errors.New("validator not found")

const (
	_RelayPathSubmitBlockBid = "/relay/v1/builder/blocks"
	_RelayPathSubmitBlockBountyBid = "/relay/v1/builder/bounty_bids"
	_RelayStatusPath         = "/eth/v1/builder/status"
)

type Relay struct {
	endpoint          string
	client            http.Client
	PayOutPoolAddress string
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
	}

	return r, nil
}

func (r *Relay) SubmitBlockBid(msg *builderTypes.BuilderBlockBid, bounty bool) (interface{}, error) {

	msgbytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var url string
	if bounty {
		url = r.endpoint + _RelayPathSubmitBlockBountyBid
	} else {
		url = r.endpoint + _RelayPathSubmitBlockBid
	}
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(msgbytes))
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
	req, err := http.NewRequestWithContext(context.Background(), "GET", r.endpoint + _RelayStatusPath, nil)
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