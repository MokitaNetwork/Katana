package e2e

import (
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/mokitanetwork/katana/app/params"
)

func (s *IntegrationTestSuite) TestIBCTokenTransfer() {
	var ibcStakeDenom string

	valAddr, err := s.chain.validators[0].keyInfo.GetAddress()
	s.Require().NoError(err)

	s.Run("send_stake_to_katana", func() {
		recipient := valAddr.String()
		token := sdk.NewInt64Coin("stake", 3300000000) // 3300stake
		s.sendIBC(gaiaChainID, s.chain.id, recipient, token)

		katanaAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[0].GetHostPort("1317/tcp"))

		// require the recipient account receives the IBC tokens (IBC packets ACKd)
		var (
			balances sdk.Coins
			err      error
		)
		s.Require().Eventually(
			func() bool {
				balances, err = queryKatanaAllBalances(katanaAPIEndpoint, recipient)
				s.Require().NoError(err)

				return balances.Len() == 3
			},
			time.Minute,
			5*time.Second,
		)

		for _, c := range balances {
			if strings.Contains(c.Denom, "ibc/") {
				ibcStakeDenom = c.Denom
				s.Require().Equal(token.Amount.Int64(), c.Amount.Int64())
				break
			}
		}

		s.Require().NotEmpty(ibcStakeDenom)
	})

	var ibcStakeERC20Addr string
	s.Run("deploy_stake_erc20 ibcStakeERC20Addr", func() {
		s.T().Skip("paused due to Ethereum PoS migration and PoW fork")
		s.Require().NotEmpty(ibcStakeDenom)
		ibcStakeERC20Addr = s.deployERC20Token(ibcStakeDenom)
	})

	// send 300 stake tokens from Katana to Ethereum
	s.Run("send_stake_tokens_to_eth", func() {
		s.T().Skip("paused due to Ethereum PoS migration and PoW fork")
		katanaValIdxSender := 0
		orchestratorIdxReceiver := 1
		amount := sdk.NewCoin(ibcStakeDenom, math.NewInt(300))
		katanaFee := sdk.NewCoin(appparams.BondDenom, math.NewInt(10000))
		gravityFee := sdk.NewCoin(ibcStakeDenom, math.NewInt(7))

		s.sendFromKatanaToEthCheck(katanaValIdxSender, orchestratorIdxReceiver, ibcStakeERC20Addr, amount, katanaFee, gravityFee)
	})

	// send 300 stake tokens from Ethereum back to Katana
	s.Run("send_stake_tokens_from_eth", func() {
		s.T().Skip("paused due to Ethereum PoS migration and PoW fork")
		katanaValIdxReceiver := 0
		orchestratorIdxSender := 1
		amount := uint64(300)

		s.sendFromEthToKatanaCheck(orchestratorIdxSender, katanaValIdxReceiver, ibcStakeERC20Addr, ibcStakeDenom, amount)
	})
}

func (s *IntegrationTestSuite) TestPhotonTokenTransfers() {
	s.T().Skip("paused due to Ethereum PoS migration and PoW fork")
	// deploy photon ERC20 token contact
	var photonERC20Addr string
	s.Run("deploy_photon_erc20", func() {
		photonERC20Addr = s.deployERC20Token(photonDenom)
	})

	// send 100 photon tokens from Katana to Ethereum
	s.Run("send_photon_tokens_to_eth", func() {
		katanaValIdxSender := 0
		orchestratorIdxReceiver := 1
		amount := sdk.NewCoin(photonDenom, math.NewInt(100))
		katanaFee := sdk.NewCoin(appparams.BondDenom, math.NewInt(10000))
		gravityFee := sdk.NewCoin(photonDenom, math.NewInt(3))

		s.sendFromKatanaToEthCheck(katanaValIdxSender, orchestratorIdxReceiver, photonERC20Addr, amount, katanaFee, gravityFee)
	})

	// send 100 photon tokens from Ethereum back to Katana
	s.Run("send_photon_tokens_from_eth", func() {
		s.T().Skip("paused due to Ethereum PoS migration and PoW fork")
		katanaValIdxReceiver := 0
		orchestratorIdxSender := 1
		amount := uint64(100)

		s.sendFromEthToKatanaCheck(orchestratorIdxSender, katanaValIdxReceiver, photonERC20Addr, photonDenom, amount)
	})
}

func (s *IntegrationTestSuite) TestKatanaTokenTransfers() {
	s.T().Skip("paused due to Ethereum PoS migration and PoW fork")
	// deploy katana ERC20 token contract
	var katanaERC20Addr string
	s.Run("deploy_katana_erc20", func() {
		katanaERC20Addr = s.deployERC20Token(appparams.BondDenom)
	})

	// send 300 katana tokens from Katana to Ethereum
	s.Run("send_ukatana_tokens_to_eth", func() {
		katanaValIdxSender := 0
		orchestratorIdxReceiver := 1
		amount := sdk.NewCoin(appparams.BondDenom, math.NewInt(300))
		katanaFee := sdk.NewCoin(appparams.BondDenom, math.NewInt(10000))
		gravityFee := sdk.NewCoin(appparams.BondDenom, math.NewInt(7))

		s.sendFromKatanaToEthCheck(katanaValIdxSender, orchestratorIdxReceiver, katanaERC20Addr, amount, katanaFee, gravityFee)
	})

	// send 300 katana tokens from Ethereum back to Katana
	s.Run("send_ukatana_tokens_from_eth", func() {
		katanaValIdxReceiver := 0
		orchestratorIdxSender := 1
		amount := uint64(300)

		s.sendFromEthToKatanaCheck(orchestratorIdxSender, katanaValIdxReceiver, katanaERC20Addr, appparams.BondDenom, amount)
	})
}
