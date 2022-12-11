package keeper_test

import (
	appparams "github.com/mokitanetwork/katana/app/params"
)

func (s *IntegrationTestSuite) TestSetReserves() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// get initial reserves
	reserves := app.LeverageKeeper.GetReserves(ctx, katanaDenom)
	require.Equal(coin(katanaDenom, 0), reserves)

	// artifically reserve 200 katana
	s.setReserves(coin(katanaDenom, 200_000000))
	// get new reserves
	reserves = app.LeverageKeeper.GetReserves(ctx, appparams.BondDenom)
	require.Equal(coin(katanaDenom, 200_000000), reserves)
}

func (s *IntegrationTestSuite) TestRepayBadDebt() {
	app, ctx, require := s.app, s.ctx, s.Require()

	// Creating a supplier so module account has some ukatana
	addr := s.newAccount(coin(katanaDenom, 200_000000))
	s.supply(addr, coin(katanaDenom, 200_000000))

	// Using an address with no assets
	addr2 := s.newAccount()

	// Create an uncollateralized debt position
	badDebt := coin(katanaDenom, 100_000000)
	err := s.tk.SetBorrow(ctx, addr2, badDebt)
	require.NoError(err)

	// Manually mark the bad debt for repayment
	require.NoError(s.tk.SetBadDebtAddress(ctx, addr2, katanaDenom, true))

	// Manually set reserves to 60 katana
	reserve := coin(katanaDenom, 60_000000)
	s.setReserves(reserve)

	// Sweep all bad debts, which should repay 60 katana of the bad debt (partial repayment)
	err = app.LeverageKeeper.SweepBadDebts(ctx)
	require.NoError(err)

	// Confirm that a debt of 40 katana remains
	remainingDebt := app.LeverageKeeper.GetBorrow(ctx, addr2, katanaDenom)
	require.Equal(coin(katanaDenom, 40_000000), remainingDebt)

	// Confirm that reserves are exhausted
	remainingReserves := app.LeverageKeeper.GetReserves(ctx, katanaDenom)
	require.Equal(coin(katanaDenom, 0), remainingReserves)

	// Manually set reserves to 70 katana
	reserve = coin(katanaDenom, 70_000000)
	s.setReserves(reserve)

	// Sweep all bad debts, which should fully repay the bad debt this time
	err = app.LeverageKeeper.SweepBadDebts(ctx)
	require.NoError(err)

	// Confirm that the debt is eliminated
	remainingDebt = app.LeverageKeeper.GetBorrow(ctx, addr2, katanaDenom)
	require.Equal(coin(katanaDenom, 0), remainingDebt)

	// Confirm that reserves are now at 30 katana
	remainingReserves = app.LeverageKeeper.GetReserves(ctx, katanaDenom)
	require.Equal(coin(katanaDenom, 30_000000), remainingReserves)

	// Sweep all bad debts - but there are none
	err = app.LeverageKeeper.SweepBadDebts(ctx)
	require.NoError(err)
}
