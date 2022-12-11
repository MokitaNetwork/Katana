package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/mokitanetwork/katana/x/leverage/types"
)

func (s *IntegrationTestSuite) TestGetCollateralAmount() {
	ctx, require := s.ctx, s.Require()
	uDenom := types.ToUTokenDenom(katanaDenom)

	// get u/katana collateral amount of empty account address
	collateral := s.tk.GetCollateral(ctx, sdk.AccAddress{}, uDenom)
	require.Equal(coin(uDenom, 0), collateral)

	// fund an account
	addr := s.newAccount(coin(katanaDenom, 1000))

	// get u/katana collateral amount of non-empty account address
	collateral = s.tk.GetCollateral(ctx, addr, uDenom)
	require.Equal(coin(uDenom, 0), collateral)

	// supply 1000 u/ukatana but do not collateralize
	s.supply(addr, coin(katanaDenom, 1000))

	// confirm collateral amount is 0 u/ukatana
	collateral = s.tk.GetCollateral(ctx, addr, uDenom)
	require.Equal(coin(uDenom, 0), collateral)

	// enable u/katana as collateral
	s.collateralize(addr, coin(uDenom, 1000))

	// confirm collateral amount is 1000 u/ukatana
	collateral = s.tk.GetCollateral(ctx, addr, uDenom)
	require.Equal(coin(uDenom, 1000), collateral)

	// collateral amount of non-utoken denom (zero)
	collateral = s.tk.GetCollateral(ctx, addr, katanaDenom)
	require.Equal(coin(katanaDenom, 0), collateral)

	// collateral amount of unregistered denom (zero)
	collateral = s.tk.GetCollateral(ctx, addr, "abcd")
	require.Equal(coin("abcd", 0), collateral)

	// disable u/katana as collateral
	s.decollateralize(addr, coin(uDenom, 1000))

	// confirm collateral amount is 0 u/ukatana
	collateral = s.tk.GetCollateral(ctx, addr, uDenom)
	require.Equal(coin(uDenom, 0), collateral)

	// we do not test empty denom, as that will cause a panic
}

func (s *IntegrationTestSuite) TestSetCollateralAmount() {
	ctx, require := s.ctx, s.Require()
	uDenom := types.ToUTokenDenom(katanaDenom)

	// set u/katana collateral amount of empty account address (error)
	err := s.tk.SetCollateral(ctx, sdk.AccAddress{}, coin(uDenom, 0))
	require.ErrorIs(err, types.ErrEmptyAddress)

	addr := s.newAccount()

	// force invalid denom
	err = s.tk.SetCollateral(ctx, addr, sdk.Coin{Denom: "", Amount: sdk.ZeroInt()})
	require.ErrorContains(err, "invalid denom")

	// base denom
	err = s.tk.SetCollateral(ctx, addr, sdk.Coin{Denom: katanaDenom, Amount: sdk.ZeroInt()})
	require.ErrorIs(err, types.ErrNotUToken)

	// negative amount
	err = s.tk.SetCollateral(ctx, addr, sdk.Coin{Denom: uDenom, Amount: sdk.NewInt(-1)})
	require.ErrorContains(err, "negative coin amount")

	// set u/katana collateral amount
	require.NoError(s.tk.SetCollateral(ctx, addr, coin(uDenom, 10)))

	// confirm effect
	collateral := s.tk.GetCollateral(ctx, addr, uDenom)
	require.Equal(coin(uDenom, 10), collateral)

	// set u/katana collateral amount to zero
	require.NoError(s.tk.SetCollateral(ctx, addr, coin(uDenom, 0)))

	// confirm effect
	collateral = s.tk.GetCollateral(ctx, addr, uDenom)
	require.Equal(coin(uDenom, 0), collateral)

	// set unregistered token collateral amount
	require.NoError(s.tk.SetCollateral(ctx, addr, coin("u/abcd", 10)))

	// confirm effect
	collateral = s.tk.GetCollateral(ctx, addr, "u/abcd")
	require.Equal(coin("u/abcd", 10), collateral)

	// set unregistered token collateral amount to zero
	require.NoError(s.tk.SetCollateral(ctx, addr, coin("u/abcd", 0)))

	// confirm effect
	collateral = s.tk.GetCollateral(ctx, addr, "u/abcd")
	require.Equal(coin("u/abcd", 0), collateral)

	// we do not test empty denom, as that will cause a panic
}

func (s *IntegrationTestSuite) TestTotalCollateral() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// not a uToken
	collateral := app.LeverageKeeper.GetTotalCollateral(ctx, katanaDenom)
	require.Equal(sdk.Coin{}, collateral, "not a uToken")

	// Test zero collateral
	uDenom := types.ToUTokenDenom(katanaDenom)
	collateral = app.LeverageKeeper.GetTotalCollateral(ctx, uDenom)
	require.Equal(coin(uDenom, 0), collateral, "zero collateral")

	// create a supplier which will have 100 u/KATANA collateral
	addr := s.newAccount(coin(katanaDenom, 100_000000))
	s.supply(addr, coin(katanaDenom, 100_000000))
	s.collateralize(addr, coin(uDenom, 100_000000))

	// Test nonzero collateral
	collateral = app.LeverageKeeper.GetTotalCollateral(ctx, uDenom)
	require.Equal(coin(uDenom, 100_000000), collateral, "nonzero collateral")
}
