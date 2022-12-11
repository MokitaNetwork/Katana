package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/leverage module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding as
	// Amino is still used for that purpose.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}

// RegisterLegacyAminoCodec registers the necessary x/leverage interfaces and
// concrete types on the provided LegacyAmino codec. These types are used for
// Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSupply{}, "katana/leverage/MsgSupply", nil)
	cdc.RegisterConcrete(&MsgWithdraw{}, "katana/leverage/MsgWithdraw", nil)
	cdc.RegisterConcrete(&MsgCollateralize{}, "katana/leverage/MsgCollateralize", nil)
	cdc.RegisterConcrete(&MsgDecollateralize{}, "katana/leverage/MsgDecollateralize", nil)
	cdc.RegisterConcrete(&MsgBorrow{}, "katana/leverage/MsgBorrow", nil)
	cdc.RegisterConcrete(&MsgRepay{}, "katana/leverage/MsgRepay", nil)
	cdc.RegisterConcrete(&MsgLiquidate{}, "katana/leverage/MsgLiquidate", nil)
	cdc.RegisterConcrete(&MsgGovUpdateRegistry{}, "katana/leverage/MsgGovUpdateRegistry", nil)
	cdc.RegisterConcrete(&MsgSupplyCollateral{}, "katana/leverage/MsgSupplyCollateral", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgSupply{},
		&MsgWithdraw{},
		&MsgCollateralize{},
		&MsgDecollateralize{},
		&MsgBorrow{},
		&MsgRepay{},
		&MsgLiquidate{},
		&MsgGovUpdateRegistry{},
		&MsgSupplyCollateral{},
	)

	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&MsgGovUpdateRegistry{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
