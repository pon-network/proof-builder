package rpbs

import (
	"bytes"
	"encoding/json"
	"strings"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/log"

	rpbsTypes "github.com/bsn-eng/pon-golang-types/rpbs"
)

type RPBSService struct {
	Endpoint string
}

func NewRPBSService(endpoint string) *RPBSService {
	return &RPBSService{
		Endpoint: endpoint,
	}
}

func (r *RPBSService) RpbsSignatureGeneration(commitMsg rpbsTypes.RPBSCommitMessage) (*rpbsTypes.EncodedRPBSSignature, error) {

	url := r.Endpoint + "/generateSignature"

	data := fmt.Sprintf("BuilderWalletAddress:%s,Slot:%d,Amount:%d,Transaction:%s", commitMsg.BuilderWalletAddress, commitMsg.Slot, commitMsg.Amount, commitMsg.PayoutTxBytes)
	data = strings.ToLower(data)

	log.Info("RPBS signature generation request", "url", url, "data", data)

	body := map[string]string{
		"commonInfo": data,
		"Message":    strings.Join(commitMsg.TxBytes, ","),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Error("RPBS signature generation error: could not marshal body", "error", err)
		return nil, err
	}

	rpbsReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Error("RPBS signature generation error: could not create request", "error", err)
		return nil, err
	}

	rpbsReq.Header.Set("accept", "application/json")
	rpbsReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(rpbsReq)
	if err != nil {
		log.Error("RPBS signature generation error: could not send request", "error", err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("RPBS signature generation error: could not read response body", "error", err)
			return nil, err
		}
		log.Error("RPBS commit error: could not commit", "error", string(bodyBytes))
		return nil, fmt.Errorf("could not commit: %s", string(bodyBytes))
	}

	var response rpbsTypes.EncodedRPBSSignature
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Error("RPBS signature generation error: could not decode response", "error", err)
		return nil, err
	}

	log.Info("RPBS signature generation success")

	return &response, nil
}

func (r *RPBSService) PublicKey() (string, error) {

	url := r.Endpoint + "/publicKey"

	req, err := http.NewRequest("GET", url, bytes.NewReader(nil))
	if err != nil {
		log.Error("RPBS public key error: could not create request", "error", err)
		return "", err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("RPBS public key error: could not send request", "error", err)
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("RPBS public key error: could not read response body", "error", err)
			return "", err
		}
		log.Error("RPBS public key error: could not get public key", "error", string(bodyBytes))
		return "", fmt.Errorf("could not get public key: %s", string(bodyBytes))
	}

	var response string
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Error("RPBS public key error: could not decode response", "error", err)
		return "", err
	}

	return response, nil
}
