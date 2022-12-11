package keeper_test

import (
	appparams "github.com/mokitanetwork/katana/app/params"
	"github.com/mokitanetwork/katana/x/leverage/keeper"
	"github.com/mokitanetwork/katana/x/leverage/types"
)

func (s *IntegrationTestSuite) TestReserveAmountInvariant() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// artificially set reserves
	s.setReserves(coin(appparams.BondDenom, 300_000000))

	// check invariants
	_, broken := keeper.ReserveAmountInvariant(app.LeverageKeeper)(ctx)
	require.False(broken)
}

func (s *IntegrationTestSuite) TestCollateralAmountInvariant() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// creates account which has supplied and collateralized 1000 KATANA
	addr := s.newAccount(coin(katanaDenom, 1000_000000))
	s.supply(addr, coin(katanaDenom, 1000_000000))
	s.collateralize(addr, coin("u/"+katanaDenom, 1000_000000))

	// check invariant
	_, broken := keeper.InefficientCollateralAmountInvariant(app.LeverageKeeper)(ctx)
	require.False(broken)

	uTokenDenom := types.ToUTokenDenom(appparams.BondDenom)

	// withdraw the supplied katana in the initBorrowScenario
	s.withdraw(addr, coin(uTokenDenom, 1000_000000))

	// check invariant
	_, broken = keeper.InefficientCollateralAmountInvariant(app.LeverageKeeper)(ctx)
	require.False(broken)
}

func (s *IntegrationTestSuite) TestBorrowAmountInvariant() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// creates account which has supplied and collateralized 1000 KATANA
	addr := s.newAccount(coin(katanaDenom, 1000_000000))
	s.supply(addr, coin(katanaDenom, 1000_000000))
	s.collateralize(addr, coin("u/"+katanaDenom, 1000_000000))

	// user borrows 20 katana
	s.borrow(addr, coin(appparams.BondDenom, 20_000000))

	// check invariant
	_, broken := keeper.InefficientBorrowAmountInvariant(app.LeverageKeeper)(ctx)
	require.False(broken)

	// user repays 30 katana, actually only 20 because is the min between
	// the amount borrowed and the amount repaid
	_, err := app.LeverageKeeper.Repay(ctx, addr, coin(appparams.BondDenom, 30_000000))
	require.NoError(err)

	// check invariant
	_, broken = keeper.InefficientBorrowAmountInvariant(app.LeverageKeeper)(ctx)
	require.False(broken)
}
