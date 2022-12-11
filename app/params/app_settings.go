package params

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// Name defines the application name of the Katana network.
	Name = "katana"

	// BondDenom defines the native staking token denomination.
	BondDenom = "ukatana"

	// DisplayDenom defines the name, symbol, and display value of the katana token.
	DisplayDenom = "KATANA"

	// DefaultGasLimit - set to the same value as cosmos-sdk flags.DefaultGasLimit
	// this value is currently only used in tests.
	DefaultGasLimit = 200000
)

var (
	// ProtocolMinGasPrice is a consensus controlled gas price. Each validator must set his
	// `minimum-gas-prices` in app.toml config to value above ProtocolMinGasPrice.
	// Transactions with gas-price smaller than ProtocolMinGasPrice will fail during DeliverTx.
	ProtocolMinGasPrice = sdk.NewDecCoinFromDec(BondDenom, sdk.MustNewDecFromStr("0.00"))
)

func init() {
	// XXX: If other upstream or external application's depend on any of Katana's
	// CLI or command functionality, then this would require us to move the
	// SetAddressConfig call to somewhere external such as the root command
	// constructor and anywhere else we contract the app.
	SetAddressConfig()

	if AccountAddressPrefix != Name {
		log.Fatal("AccountAddresPrefix must equal Name")
	}
}
