package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	gravitytypes "github.com/Gravity-Bridge/Gravity-Bridge/module/x/gravity/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ory/dockertest/v3/docker"
)

func (s *IntegrationTestSuite) deployERC20Token(baseDenom string) string {
	s.T().Logf("deploying ERC20 token contract: %s", baseDenom)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.orchResources[0].Container.ID,
		User:         "root",
		Env:          []string{"PEGGO_ETH_PK=" + ethMinerPK},
		Cmd: []string{
			"peggo",
			"bridge",
			"deploy-erc20",
			s.gravityContractAddr,
			baseDenom,
			"--eth-rpc",
			fmt.Sprintf("http://%s:8545", s.ethResource.Container.Name[1:]),
			"--cosmos-chain-id",
			s.chain.id,
			"--cosmos-grpc",
			fmt.Sprintf("tcp://%s:9090", s.valResources[0].Container.Name[1:]),
			"--tendermint-rpc",
			fmt.Sprintf("http://%s:26657", s.valResources[0].Container.Name[1:]),
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(
		err,
		"failed to get ERC20 deployment logs; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	re := regexp.MustCompile(`Transaction: (0x.+)`)
	tokens := re.FindStringSubmatch(errBuf.String())
	s.Require().Lenf(tokens, 2, "stderr: %s", errBuf.String())

	txHash := tokens[1]
	s.Require().NotEmpty(txHash)

	s.Require().Eventually(
		func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := queryEthTx(ctx, s.ethClient, txHash); err != nil {
				return false
			}

			return true
		},
		6*time.Minute,
		time.Second,
		"failed to confirm ERC20 deployment transaction",
	)

	katanaAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[0].GetHostPort("1317/tcp"))

	var erc20Addr string
	s.Require().Eventually(
		func() bool {
			addr, cosmosNative, err := queryDenomToERC20(katanaAPIEndpoint, baseDenom)
			if err != nil {
				return false
			}

			if cosmosNative && len(addr) > 0 {
				erc20Addr = addr
				return true
			}

			return false
		},
		2*time.Minute,
		time.Second,
		"failed to query ERC20 contract address",
	)

	s.T().Logf("deployed %s contract: %s", baseDenom, erc20Addr)

	return erc20Addr
}

func (s *IntegrationTestSuite) sendFromKatanaToEth(valIdx int, ethDest, amount, katanaFee, gravityFee string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	valAddr, err := s.chain.validators[valIdx].keyInfo.GetAddress()
	s.Require().NoError(err)

	s.T().Logf(
		"sending tokens from Katana to Ethereum; from: %s, to: %s, amount: %s, katanaFee: %s, gravityFee: %s",
		valAddr, ethDest, amount, katanaFee, gravityFee,
	)

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.valResources[valIdx].Container.ID,
		User:         "root",
		Cmd: []string{
			"katanad",
			"tx",
			"gravity",
			"send-to-eth",
			ethDest,
			amount,
			gravityFee,
			fmt.Sprintf("--%s=%s", flags.FlagFrom, s.chain.validators[valIdx].keyInfo.Name),
			fmt.Sprintf("--%s=%s", flags.FlagChainID, s.chain.id),
			fmt.Sprintf("--%s=%s", flags.FlagFees, katanaFee),
			"--keyring-backend=test",
			"--broadcast-mode=sync",
			"--output=json",
			"-y",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(err, "stdout: %s, stderr: %s", outBuf.String(), errBuf.String())

	var broadcastResp map[string]interface{}
	s.Require().NoError(json.Unmarshal(outBuf.Bytes(), &broadcastResp), outBuf.String())

	endpoint := fmt.Sprintf("http://%s", s.valResources[valIdx].GetHostPort("1317/tcp"))
	txHash := broadcastResp["txhash"].(string)

	s.Require().Eventuallyf(
		func() bool {
			return queryKatanaTx(endpoint, txHash) == nil
		},
		2*time.Minute,
		5*time.Second,
		"stdout: %s, stderr: %s",
		outBuf.String(), errBuf.String(),
	)
}

func (s *IntegrationTestSuite) sendFromKatanaToEthCheck(
	katanaValIdxSender,
	orchestratorIdxReceiver int,
	ethTokenAddr string,
	amount, katanaFee, gravityFee sdk.Coin,
) {
	if !strings.EqualFold(amount.Denom, gravityFee.Denom) {
		s.T().Error("Amount and gravityFee should be the same denom", amount, gravityFee)
	}

	// if all the coins are on the same denom
	allSameDenom := strings.EqualFold(amount.Denom, katanaFee.Denom) && strings.EqualFold(amount.Denom, gravityFee.Denom)
	var katanaFeeBalanceBeforeSend sdk.Coin
	if !allSameDenom {
		katanaFeeBalanceBeforeSend, _ = s.queryKatanaBalance(katanaValIdxSender, katanaFee.Denom)
	}

	katanaAmountBalanceBeforeSend, ethBalanceBeforeSend, _, ethAddr := s.queryKatanaEthBalance(katanaValIdxSender, orchestratorIdxReceiver, amount.Denom, ethTokenAddr) // 3300000000

	s.sendFromKatanaToEth(katanaValIdxSender, ethAddr, amount.String(), katanaFee.String(), gravityFee.String())
	katanaAmountBalanceAfterSend, ethBalanceAfterSend, _, _ := s.queryKatanaEthBalance(katanaValIdxSender, orchestratorIdxReceiver, amount.Denom, ethTokenAddr) // 3299999693

	if allSameDenom {
		s.Require().Equal(katanaAmountBalanceBeforeSend.Sub(amount).Sub(katanaFee).Sub(gravityFee).Amount.Int64(), katanaAmountBalanceAfterSend.Amount.Int64())
	} else { // the katanaFee and amount have different denom
		s.Require().Equal(katanaAmountBalanceBeforeSend.Sub(amount).Sub(gravityFee).Amount.Int64(), katanaAmountBalanceAfterSend.Amount.Int64())
		katanaFeeBalanceAfterSend, _ := s.queryKatanaBalance(katanaValIdxSender, katanaFee.Denom)
		s.Require().Equal(katanaFeeBalanceBeforeSend.Sub(katanaFee).Amount.Int64(), katanaFeeBalanceAfterSend.Amount.Int64())
	}

	// require the Ethereum recipient balance increased
	// peggo needs time to read the event and cross the tx
	ethLatestBalance := ethBalanceAfterSend
	expectedAmount := (ethBalanceBeforeSend + int64(amount.Amount.Int64()))
	s.Require().Eventuallyf(
		func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			b, err := queryEthTokenBalance(ctx, s.ethClient, ethTokenAddr, ethAddr)
			if err != nil {
				return false
			}

			ethLatestBalance = b

			// The balance could differ if the receiving address was the orchestrator
			// that sent the batch tx and got the gravity fee.
			return b >= expectedAmount && b <= expectedAmount+gravityFee.Amount.Int64()
		},
		2*time.Minute,
		5*time.Second,
		"unexpected balance: %d", ethLatestBalance,
	)
}

func (s *IntegrationTestSuite) sendFromEthToKatanaCheck(
	orchestratorIdxSender,
	katanaValIdxReceiver int,
	ethTokenAddr,
	katanaTokenDenom string,
	amount uint64,
) {
	katanaBalanceBeforeSend, ethBalanceBeforeSend, katanaAddr, _ := s.queryKatanaEthBalance(katanaValIdxReceiver, orchestratorIdxSender, katanaTokenDenom, ethTokenAddr)
	s.sendFromEthToKatana(orchestratorIdxSender, ethTokenAddr, katanaAddr, fmt.Sprintf("%d", amount))
	katanaBalanceAfterSend, ethBalanceAfterSend, _, _ := s.queryKatanaEthBalance(katanaValIdxReceiver, orchestratorIdxSender, katanaTokenDenom, ethTokenAddr)

	s.Require().Equal(ethBalanceBeforeSend-int64(amount), ethBalanceAfterSend)

	katanaEndpoint := fmt.Sprintf("http://%s", s.valResources[katanaValIdxReceiver].GetHostPort("1317/tcp"))
	// require the original sender's (validator) balance increased
	// peggo needs time to read the event and cross the tx
	katanaLatestBalance := katanaBalanceAfterSend.Amount
	s.Require().Eventuallyf(
		func() bool {
			b, err := queryKatanaDenomBalance(katanaEndpoint, katanaAddr, katanaTokenDenom)
			if err != nil {
				s.T().Logf("Error at sendFromEthToKatanaCheck.queryKatanaDenomBalance %+v", err)
				return false
			}

			katanaLatestBalance = b.Amount

			return katanaBalanceBeforeSend.Amount.AddRaw(int64(amount)).Equal(katanaLatestBalance)
		},
		2*time.Minute,
		5*time.Second,
		"unexpected balance: %d", katanaLatestBalance.Int64(),
	)
}

func (s *IntegrationTestSuite) sendFromEthToKatana(valIdx int, tokenAddr, toKatanaAddr, amount string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf(
		"sending tokens from Ethereum to Katana; from: %s, to: %s, amount: %s, contract: %s",
		s.chain.orchestrators[valIdx].ethereumKey.address, toKatanaAddr, amount, tokenAddr,
	)

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.orchResources[valIdx].Container.ID,
		User:         "root",
		Env:          []string{"PEGGO_ETH_PK=" + s.chain.orchestrators[valIdx].ethereumKey.privateKey},
		Cmd: []string{
			"peggo",
			"bridge",
			"send-to-cosmos",
			s.gravityContractAddr,
			tokenAddr,
			toKatanaAddr,
			amount,
			"--eth-rpc",
			fmt.Sprintf("http://%s:8545", s.ethResource.Container.Name[1:]),
			"--cosmos-chain-id",
			s.chain.id,
			"--cosmos-grpc",
			fmt.Sprintf("tcp://%s:9090", s.valResources[valIdx].Container.Name[1:]),
			"--tendermint-rpc",
			fmt.Sprintf("http://%s:26657", s.valResources[valIdx].Container.Name[1:]),
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(err, "stdout: %s, stderr: %s", outBuf.String(), errBuf.String())

	re := regexp.MustCompile(`Transaction: (0x.+)`)
	tokens := re.FindStringSubmatch(errBuf.String())
	s.Require().Len(tokens, 2)

	txHash := tokens[1]
	s.Require().NotEmpty(txHash)

	s.Require().Eventuallyf(
		func() bool {
			return queryEthTx(ctx, s.ethClient, txHash) == nil
		},
		5*time.Minute,
		5*time.Second,
		"stdout: %s, stderr: %s",
		outBuf.String(), errBuf.String(),
	)
}

func (s *IntegrationTestSuite) connectIBCChains() {
	s.T().Logf("connecting %s and %s chains via IBC", s.chain.id, gaiaChainID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.hermesResource.Container.ID,
		User:         "root",
		Cmd: []string{
			"hermes",
			"create",
			"channel",
			s.chain.id,
			gaiaChainID,
			"--port-a=transfer",
			"--port-b=transfer",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(
		err,
		"failed connect chains; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	s.Require().Containsf(
		errBuf.String(),
		"successfully opened init channel",
		"failed to connect chains via IBC: %s", errBuf.String(),
	)

	s.T().Logf("connected %s and %s chains via IBC", s.chain.id, gaiaChainID)
}

func (s *IntegrationTestSuite) sendIBC(srcChainID, dstChainID, recipient string, token sdk.Coin) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf("sending %s from %s to %s (%s)", token, srcChainID, dstChainID, recipient)

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.hermesResource.Container.ID,
		User:         "root",
		Cmd: []string{
			"hermes",
			"tx",
			"raw",
			"ft-transfer",
			dstChainID,
			srcChainID,
			"transfer",  // source chain port ID
			"channel-0", // since only one connection/channel exists, assume 0
			token.Amount.String(),
			fmt.Sprintf("--denom=%s", token.Denom),
			fmt.Sprintf("--receiver=%s", recipient),
			"--timeout-height-offset=1000",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(
		err,
		"failed to send IBC tokens; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	s.T().Log("successfully sent IBC tokens")
}

func queryKatanaTx(endpoint, txHash string) error {
	resp, err := http.Get(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", endpoint, txHash))
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("tx query returned non-200 status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	txResp := result["tx_response"].(map[string]interface{})
	if v := txResp["code"]; v.(float64) != 0 {
		return fmt.Errorf("tx %s failed with status code %v", txHash, v)
	}

	return nil
}

func queryKatanaAllBalances(endpoint, addr string) (sdk.Coins, error) {
	resp, err := http.Get(fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", endpoint, addr))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var balancesResp banktypes.QueryAllBalancesResponse
	if err := cdc.UnmarshalJSON(bz, &balancesResp); err != nil {
		return nil, err
	}

	return balancesResp.Balances, nil
}

func queryKatanaDenomBalance(endpoint, addr, denom string) (sdk.Coin, error) {
	var zeroCoin sdk.Coin

	path := fmt.Sprintf(
		"%s/cosmos/bank/v1beta1/balances/%s/by_denom?denom=%s",
		endpoint, addr, denom,
	)
	resp, err := http.Get(path)
	if err != nil {
		return zeroCoin, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return zeroCoin, err
	}

	var balanceResp banktypes.QueryBalanceResponse
	if err := cdc.UnmarshalJSON(bz, &balanceResp); err != nil {
		return zeroCoin, err
	}

	return *balanceResp.Balance, nil
}

func queryDenomToERC20(endpoint, denom string) (string, bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s/gravity/v1beta/cosmos_originated/denom_to_erc20?denom=%s", endpoint, denom))
	if err != nil {
		return "", false, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	var denomToERC20Resp gravitytypes.QueryDenomToERC20Response
	if err := cdc.UnmarshalJSON(bz, &denomToERC20Resp); err != nil {
		return "", false, err
	}

	return denomToERC20Resp.Erc20, denomToERC20Resp.CosmosOriginated, nil
}

func queryEthTx(ctx context.Context, c *ethclient.Client, txHash string) error {
	_, pending, err := c.TransactionByHash(ctx, ethcmn.HexToHash(txHash))
	if err != nil {
		return err
	}

	if pending {
		return fmt.Errorf("ethereum tx %s is still pending", txHash)
	}

	return nil
}

func queryEthTokenBalance(ctx context.Context, c *ethclient.Client, contractAddr, recipientAddr string) (int64, error) {
	data, err := ethABI.Pack(abiMethodNameBalanceOf, ethcmn.HexToAddress(recipientAddr))
	if err != nil {
		return 0, fmt.Errorf("failed to pack ABI method call: %w", err)
	}

	token := ethcmn.HexToAddress(contractAddr)
	callMsg := ethereum.CallMsg{
		To:   &token,
		Data: data,
	}

	bz, err := c.CallContract(ctx, callMsg, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call Ethereum contract: %w", err)
	}

	balance, err := strconv.ParseInt(ethcmn.Bytes2Hex(bz), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse balance: %w", err)
	}

	return balance, nil
}

func (s *IntegrationTestSuite) queryKatanaBalance(
	katanaValIdx int,
	katanaTokenDenom string,
) (katanaBalance sdk.Coin, katanaAddr string) {
	katanaEndpoint := fmt.Sprintf("http://%s", s.valResources[katanaValIdx].GetHostPort("1317/tcp"))
	katanaAddress, err := s.chain.validators[katanaValIdx].keyInfo.GetAddress()
	s.Require().NoError(err)
	katanaAddr = katanaAddress.String()

	katanaBalance, err = queryKatanaDenomBalance(katanaEndpoint, katanaAddr, katanaTokenDenom)
	s.Require().NoError(err)
	s.T().Logf(
		"Katana Balance of tokens validator; index: %d, addr: %s, amount: %s, denom: %s",
		katanaValIdx, katanaAddr, katanaBalance.String(), katanaTokenDenom,
	)

	return katanaBalance, katanaAddr
}

func (s *IntegrationTestSuite) queryKatanaEthBalance(
	katanaValIdx,
	orchestratorIdx int,
	katanaTokenDenom,
	ethTokenAddr string,
) (katanaBalance sdk.Coin, ethBalance int64, katanaAddr, ethAddr string) {
	katanaBalance, katanaAddr = s.queryKatanaBalance(katanaValIdx, katanaTokenDenom)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ethAddr = s.chain.orchestrators[orchestratorIdx].ethereumKey.address

	ethBalance, err := queryEthTokenBalance(ctx, s.ethClient, ethTokenAddr, ethAddr)
	s.Require().NoError(err)
	s.T().Logf(
		"ETh Balance of tokens; index: %d, addr: %s, amount: %d, denom: %s, erc20Addr: %s",
		orchestratorIdx, ethAddr, ethBalance, katanaTokenDenom, ethTokenAddr,
	)

	return katanaBalance, ethBalance, katanaAddr, ethAddr
}
