package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) TestGetEligibleLiquidationTargets_OneAddrOneAsset() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// creates account which has supplied and collateralized 1000 ukatana
	addr := s.newAccount(coin(katanaDenom, 1000))
	s.supply(addr, coin(katanaDenom, 1000))
	s.collateralize(addr, coin("u/"+katanaDenom, 1000))

	// user borrows 250 katana (max current allowed)
	s.borrow(addr, coin(katanaDenom, 250))

	zeroAddresses, err := app.LeverageKeeper.GetEligibleLiquidationTargets(ctx)
	require.NoError(err)
	require.Equal([]sdk.AccAddress{}, zeroAddresses)

	// Note: Setting katana liquidation threshold to 0.05 to make the user eligible to liquidation
	katanaToken := newToken("ukatana", "KATANA", 6)
	katanaToken.CollateralWeight = sdk.MustNewDecFromStr("0.05")
	katanaToken.LiquidationThreshold = sdk.MustNewDecFromStr("0.05")

	require.NoError(app.LeverageKeeper.SetTokenSettings(ctx, katanaToken))

	targetAddress, err := app.LeverageKeeper.GetEligibleLiquidationTargets(ctx)
	require.NoError(err)
	require.Equal([]sdk.AccAddress{addr}, targetAddress)
}

func (s *IntegrationTestSuite) TestGetEligibleLiquidationTargets_OneAddrTwoAsset() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// creates account which has supplied and collateralized 1000 ukatana
	addr := s.newAccount(coin(katanaDenom, 1000))
	s.supply(addr, coin(katanaDenom, 1000))
	s.collateralize(addr, coin("u/"+katanaDenom, 1000))

	// user borrows 250 katana (max current allowed)
	s.borrow(addr, coin(katanaDenom, 250))

	zeroAddresses, err := app.LeverageKeeper.GetEligibleLiquidationTargets(ctx)
	require.NoError(err)
	require.Equal([]sdk.AccAddress{}, zeroAddresses)

	// mints and send to addr 100 atom and already
	// enable 50 u/atom as collateral.
	s.fundAccount(addr, coin(atomDenom, 100_000000))
	s.supply(addr, coin(atomDenom, 50_000000))
	s.collateralize(addr, coin("u/"+atomDenom, 50_000000))

	// user borrows 4 atom (max current allowed - 1) user amount enabled as collateral * CollateralWeight
	// = (50 * 0.1) - 1
	// = 4app.
	s.borrow(addr, coin(atomDenom, 4_000000))

	// Note: Setting katana liquidation threshold to 0.05 to make the user eligible for liquidation
	katanaToken := newToken("ukatana", "KATANA", 6)
	katanaToken.CollateralWeight = sdk.MustNewDecFromStr("0.05")
	katanaToken.LiquidationThreshold = sdk.MustNewDecFromStr("0.05")

	require.NoError(app.LeverageKeeper.SetTokenSettings(s.ctx, katanaToken))

	// Note: Setting atom collateral weight to 0.01 to make the user eligible for liquidation
	atomIBCToken := newToken(atomDenom, "ATOM", 6)
	atomIBCToken.CollateralWeight = sdk.MustNewDecFromStr("0.01")
	atomIBCToken.LiquidationThreshold = sdk.MustNewDecFromStr("0.01")

	require.NoError(app.LeverageKeeper.SetTokenSettings(s.ctx, atomIBCToken))

	targetAddr, err := app.LeverageKeeper.GetEligibleLiquidationTargets(ctx)
	require.NoError(err)
	require.Equal([]sdk.AccAddress{addr}, targetAddr)
}

func (s *IntegrationTestSuite) TestGetEligibleLiquidationTargets_TwoAddr() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// creates account which has supplied and collateralized 1000 ukatana
	addr := s.newAccount(coin(katanaDenom, 1000))
	s.supply(addr, coin(katanaDenom, 1000))
	s.collateralize(addr, coin("u/"+katanaDenom, 1000))

	// user borrows 250 katana (max current allowed)
	s.borrow(addr, coin(katanaDenom, 250))

	zeroAddresses, err := app.LeverageKeeper.GetEligibleLiquidationTargets(ctx)
	require.NoError(err)
	require.Equal([]sdk.AccAddress{}, zeroAddresses)

	// creates another account which has supplied and collateralized 100 uatom
	addr2 := s.newAccount(coin(atomDenom, 100))
	s.supply(addr2, coin(atomDenom, 100))
	s.collateralize(addr2, coin("u/"+atomDenom, 100))

	// borrows atom (max current allowed - 1)
	s.borrow(addr2, coin(atomDenom, 24))

	// Note: Setting katana liquidation threshold to 0.05 to make the first supplier eligible for liquidation
	katanaToken := newToken("ukatana", "KATANA", 6)
	katanaToken.CollateralWeight = sdk.MustNewDecFromStr("0.05")
	katanaToken.LiquidationThreshold = sdk.MustNewDecFromStr("0.05")

	require.NoError(app.LeverageKeeper.SetTokenSettings(s.ctx, katanaToken))

	// Note: Setting atom collateral weight to 0.01 to make the second supplier eligible for liquidation
	atomIBCToken := newToken(atomDenom, "ATOM", 6)
	atomIBCToken.CollateralWeight = sdk.MustNewDecFromStr("0.01")
	atomIBCToken.LiquidationThreshold = sdk.MustNewDecFromStr("0.01")

	require.NoError(app.LeverageKeeper.SetTokenSettings(s.ctx, atomIBCToken))

	targets, err := app.LeverageKeeper.GetEligibleLiquidationTargets(ctx)
	require.NoError(err)
	require.Equal([]sdk.AccAddress{addr, addr2}, targets)
}
