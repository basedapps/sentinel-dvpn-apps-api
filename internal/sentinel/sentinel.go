package sentinel

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type Sentinel struct {
	APIEndpoint string
	RPCEndpoint string

	ProviderWalletAddress string
	ProviderMnemonic      string
	ProviderPlanID        string

	MainSubscriberWalletAddress string
	MainSubscriberMnemonic      string

	DefaultDenom string
	ChainID      string
	GasPrice     string
}

func (s Sentinel) FetchNodes(offset int, limit int) (*[]SentinelNode, error) {
	type blockchainResponse struct {
		Success bool            `json:"success"`
		Result  *[]SentinelNode `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&limit=%d&offset=%d&status=%s",
		s.RPCEndpoint,
		s.ChainID,
		limit,
		offset,
		"Active",
	)

	url := s.APIEndpoint + "/api/v1/nodes" + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API when fetching nodes")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API when fetching nodes")
	}

	return response.Result, nil
}

func (s Sentinel) FetchNodeStatus(node SentinelNode) (*SentinelNodeStatus, error) {
	type nodeResponse struct {
		Success bool                `json:"success"`
		Result  *SentinelNodeStatus `json:"result"`
	}

	url := fmt.Sprintf("%s/status", node.RemoteURL)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response nodeResponse

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("failed to retrieve Sentinel node status for (got success false)")
	}

	return response.Result, nil
}

func (s Sentinel) FetchBalance(walletAddress string) (int64, error) {
	type blockchainResponse struct {
		Success bool               `json:"success"`
		Result  *[]SentinelBalance `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s",
		s.RPCEndpoint,
		s.ChainID,
	)

	url := s.APIEndpoint + "/api/v1/accounts/" + walletAddress + "/balances" + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		return 0, errors.New("status code " + res.Status + " returned from Sentinel API when fetching balance for wallet " + walletAddress)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}

	if response.Success == false {
		return 0, errors.New("success `false` returned from Sentinel API when fetching balance for wallet " + walletAddress)
	}

	var walletBalance int64
	for _, balance := range *response.Result {
		if balance.Denom == s.DefaultDenom {
			walletBalance, _ = strconv.ParseInt(balance.Amount, 10, 64)
		}
	}

	return walletBalance, nil
}

func (s Sentinel) GrantTokens(walletAddresses []string, amount int64) error {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelTransaction `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic       string   `json:"mnemonic"`
		ToAccAddresses []string `json:"to_acc_addresses"`
		Amounts        []string `json:"amounts"`
	}

	var amountsArr []string = make([]string, len(walletAddresses))
	for i := 0; i < len(walletAddresses); i++ {
		amountsArr[i] = strconv.FormatInt(amount, 10) + s.DefaultDenom
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic:       s.MainSubscriberMnemonic,
		ToAccAddresses: walletAddresses,
		Amounts:        amountsArr,
	})
	if err != nil {
		return err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/balances" + args
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New("status code " + res.Status + " returned from Sentinel API when topping up wallets")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Success == false {
		return errors.New("success `false` returned from Sentinel API when topping up wallets")
	}

	return nil
}

func (s Sentinel) FetchSessions(walletAddress string, offset int, limit int) (*[]SentinelSession, error) {
	type blockchainResponse struct {
		Success bool               `json:"success"`
		Result  *[]SentinelSession `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&limit=%d&offset=%d",
		s.RPCEndpoint,
		s.ChainID,
		limit,
		offset,
	)

	url := s.APIEndpoint + "/api/v1/accounts/" + walletAddress + "/sessions" + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API when fetching sessions for wallet " + walletAddress)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API when fetching sessions for wallet " + walletAddress)
	}

	return response.Result, nil
}

func (s Sentinel) FetchSubscriptions(walletAddress string, offset int, limit int) (*[]SentinelSubscription, error) {
	type blockchainResponse struct {
		Success bool                    `json:"success"`
		Result  *[]SentinelSubscription `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&limit=%d&offset=%d",
		s.RPCEndpoint,
		s.ChainID,
		limit,
		offset,
	)

	url := s.APIEndpoint + "/api/v1/accounts/" + walletAddress + "/subscriptions" + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API when fetching subscriptions for wallet " + walletAddress)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API when fetching subscriptions for wallet " + walletAddress)
	}

	return response.Result, nil
}

func (s Sentinel) FindSubscriptionForNode(walletAddress string, nodeAddress string) (*SentinelSubscription, error) {
	var fetchInProgress bool
	var limit int
	var offset int

	fetchInProgress = true
	limit = 100
	offset = 0

	var subscriptions []SentinelSubscription

	for fetchInProgress {
		s, err := s.FetchSubscriptions(walletAddress, offset, limit)
		if err != nil {
			return nil, err
		}

		if s == nil {
			fetchInProgress = false
		} else {
			subscriptions = append(subscriptions, *s...)
		}

		offset += limit
	}

	for _, subscription := range subscriptions {
		if subscription.NodeAddress == nodeAddress {
			return &subscription, nil
		}
	}

	return nil, nil

}

func (s Sentinel) FindSubscriptionByID(subscriptionID int64) (*SentinelSubscription, error) {
	type blockchainResponse struct {
		Success bool                  `json:"success"`
		Result  *SentinelSubscription `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s",
		s.RPCEndpoint,
		s.ChainID,
	)

	url := s.APIEndpoint + "/api/v1/subscriptions/" + strconv.FormatInt(subscriptionID, 10) + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API when fetching subscription with ID" + strconv.FormatInt(subscriptionID, 10))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API when fetching subscription with ID" + strconv.FormatInt(subscriptionID, 10))
	}

	return response.Result, nil
}

func (s Sentinel) CreateNodeSubscription(walletAddress string, mnemonic string, nodeAddress string, gigabytes int64, hours int64) (*SentinelSubscription, error) {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelTransaction `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic  string `json:"mnemonic"`
		Denom     string `json:"denom"`
		Gigabytes int64  `json:"gigabytes,omitempty"`
		Hours     int64  `json:"hours,omitempty"`
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic:  mnemonic,
		Denom:     s.DefaultDenom,
		Gigabytes: gigabytes,
		Hours:     hours,
	})

	if err != nil {
		return nil, err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/nodes/" + nodeAddress + "/subscriptions" + args
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API during creation of subscription for wallet " + walletAddress)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned  from Sentinel API during creation of subscription for wallet " + walletAddress)
	}

	for _, event := range response.Result.Events {
		if event.Type == "sentinel.node.v2.EventCreateSubscription" {
			for _, attribute := range event.Attributes {

				keyBytes, err := base64.StdEncoding.DecodeString(attribute.Key)
				if err != nil {
					return nil, err
				}

				if string(keyBytes) == "id" {
					valueBytes, err := base64.StdEncoding.DecodeString(attribute.Value)
					if err != nil {
						return nil, err
					}

					value := string(valueBytes)
					subscriptionID, err := strconv.ParseInt(value[1:len(value)-1], 10, 64)
					if err != nil {
						return nil, err
					}

					return s.FindSubscriptionByID(subscriptionID)
				}
			}
		}
	}

	return nil, errors.New("No subscription ID found in events returned from Sentinel API during creation of subscription for wallet " + walletAddress)
}

func (s Sentinel) FetchAllocationsForSubscription(subscriptionID int64) (*SentinelAllocation, error) {
	type blockchainResponse struct {
		Success bool                  `json:"success"`
		Result  *[]SentinelAllocation `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s",
		s.RPCEndpoint,
		s.ChainID,
	)

	url := s.APIEndpoint + "/api/v1/subscriptions/" + strconv.FormatInt(subscriptionID, 10) + "/allocations" + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API when fetching allocation for subscription with ID " + strconv.FormatInt(subscriptionID, 10))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API when fetching allocation for subscription with ID " + strconv.FormatInt(subscriptionID, 10))
	}

	if response.Result == nil {
		return nil, nil
	}

	lastIndex := len(*response.Result) - 1
	return &(*response.Result)[lastIndex], nil
}

func (s Sentinel) CreateCredentials(nodeAddress string, subscriptionID int64, mnemonic string) (*SentinelCredentials, error) {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelCredentials `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic string `json:"mnemonic"`
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic: mnemonic,
	})
	if err != nil {
		return nil, err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/nodes/" + nodeAddress + "/sessions/" + strconv.FormatInt(subscriptionID, 10) + "/keys" + args
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API during creation of credentials for node " + nodeAddress)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API during creation of credentials for node " + nodeAddress)
	}

	return response.Result, nil
}

func (s Sentinel) ProxyManualCredentialsRequest(remoteURL string, walletAddress string, sessionID int64, payload []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/accounts/%s/sessions/%d", remoteURL, walletAddress, sessionID)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: transport,
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

/// New methods

func (s Sentinel) FetchPlanNodes(limit int, offset int) (*[]SentinelNode, error) {
	type blockchainResponse struct {
		Success bool            `json:"success"`
		Result  *[]SentinelNode `json:"result"`
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&limit=%d&offset=%d",
		s.RPCEndpoint,
		s.ChainID,
		limit,
		offset,
	)

	url := s.APIEndpoint + "/api/v1/plans/" + s.ProviderPlanID + "/nodes" + args
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API when fetching nodes for plan")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned from Sentinel API when fetching nodes for plan")
	}

	return response.Result, nil
}

func (s Sentinel) AddNodeToPlan(nodeAddresses []string) error {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelTransaction `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic    string   `json:"mnemonic"`
		NodeAddress []string `json:"node_addresses"`
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic:    s.ProviderMnemonic,
		NodeAddress: nodeAddresses,
	})

	if err != nil {
		return err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/plans/" + s.ProviderPlanID + "/nodes" + args
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New("status code " + res.Status + " returned from Sentinel API while adding nodes to plan " + s.ProviderPlanID)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Success == false {
		return errors.New("success `false` returned from Sentinel API while adding nodes to plan " + s.ProviderPlanID)
	}

	return nil
}

func (s Sentinel) RemoveNodeFromPlan(nodeAddress string) error {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelTransaction `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic string `json:"mnemonic"`
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic: s.ProviderMnemonic,
	})

	if err != nil {
		return err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/plans/" + s.ProviderPlanID + "/nodes/" + nodeAddress + args
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New("status code " + res.Status + " returned from Sentinel API while removing node  " + nodeAddress + " from plan " + s.ProviderPlanID)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Success == false {
		return errors.New("success `false` returned from Sentinel API while removing node  " + nodeAddress + " from plan " + s.ProviderPlanID)
	}

	return nil
}

func (s Sentinel) EnrollWalletToSubscription(walletAddresses []string, subscriptionID int64) error {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelTransaction `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic     string   `json:"mnemonic"`
		AccAddresses []string `json:"acc_addresses"`
		Bytes        []int64  `json:"bytes"`
	}

	var bytesArr []int64 = make([]int64, len(walletAddresses))
	for i := 0; i < len(walletAddresses); i++ {
		bytesArr[i] = 100000000000000
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic:     s.MainSubscriberMnemonic,
		AccAddresses: walletAddresses,
		Bytes:        bytesArr,
	})

	if err != nil {
		return err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/subscriptions/" + strconv.FormatInt(subscriptionID, 10) + "/allocations" + args
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New("status code " + res.Status + " returned from Sentinel API while adding wallets to subscription")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Success == false {
		return errors.New("success `false` returned from Sentinel API while adding wallets to subscription")
	}

	return nil
}

func (s Sentinel) CreatePlanSubscription() (*SentinelSubscription, error) {
	type blockchainResponse struct {
		Success bool                 `json:"success"`
		Result  *SentinelTransaction `json:"result"`
	}

	type blockchainRequest struct {
		Mnemonic string `json:"mnemonic"`
		Denom    string `json:"denom"`
	}

	payload, err := json.Marshal(blockchainRequest{
		Mnemonic: s.MainSubscriberMnemonic,
		Denom:    s.DefaultDenom,
	})

	if err != nil {
		return nil, err
	}

	args := fmt.Sprintf(
		"?rpc_address=%s&chain_id=%s&gas_prices=%s",
		s.RPCEndpoint,
		s.ChainID,
		s.GasPrice+s.DefaultDenom,
	)

	url := s.APIEndpoint + "/api/v1/plans/" + s.ProviderPlanID + "/subscriptions" + args
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("status code " + res.Status + " returned from Sentinel API during creation of subscription for plan " + s.ProviderPlanID)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var response *blockchainResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Success == false {
		return nil, errors.New("success `false` returned  from Sentinel API during creation of subscription for plan " + s.ProviderPlanID)
	}

	for _, event := range response.Result.Events {
		if event.Type == "sentinel.plan.v2.EventCreateSubscription" {
			for _, attribute := range event.Attributes {

				keyBytes, err := base64.StdEncoding.DecodeString(attribute.Key)
				if err != nil {
					return nil, err
				}

				if string(keyBytes) == "id" {
					valueBytes, err := base64.StdEncoding.DecodeString(attribute.Value)
					if err != nil {
						return nil, err
					}

					value := string(valueBytes)
					subscriptionID, err := strconv.ParseInt(value[1:len(value)-1], 10, 64)
					if err != nil {
						return nil, err
					}

					return s.FindSubscriptionByID(subscriptionID)
				}
			}
		}
	}

	return nil, errors.New("No subscription ID found in events returned from Sentinel API during creation of subscription for plan " + s.ProviderPlanID)
}
