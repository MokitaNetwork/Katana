//go:build norace
// +build norace

package tests

import (
	"testing"

	gravitytypes "github.com/Gravity-Bridge/Gravity-Bridge/module/x/gravity/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	katanaapp "github.com/mokitanetwork/katana/app"
)

func TestIntegrationTestSuite(t *testing.T) {
	cfg := katanaapp.IntegrationTestNetworkConfig()
	cfg.NumValidators = 2
	cfg.Mnemonics = []string{
		"empower ridge mystery shrimp predict alarm swear brick across funny vendor essay antique vote place lava proof gaze crush head east arch twin lady",
		"clean target advice dirt onion correct original vibrant actor upon waste eternal color barely shrimp aspect fall material wait repeat bench demise length seven",
	}

	var gravityGenState gravitytypes.GenesisState
	require.NoError(t, cfg.Codec.UnmarshalJSON(cfg.GenesisState[gravitytypes.ModuleName], &gravityGenState))

	gravityGenState.DelegateKeys = []gravitytypes.MsgSetOrchestratorAddress{
		{
			Validator:    "katanavaloper1t3ass54lpra0klz047k0dax33ckahym5phdrpg",
			Orchestrator: "katana1t3ass54lpra0klz047k0dax33ckahym5pn2vsz",
			EthAddress:   "0x9fc56f2e851e1ab2b4c0fc4f6344800f29652ffe",
		},
		{
			Validator:    "katanavaloper1kqh6nt4f48vptvq4j5cgr0nfd2x4z9ulvrtqrh",
			Orchestrator: "katana1kqh6nt4f48vptvq4j5cgr0nfd2x4z9ulv8v0ja",
			EthAddress:   "0xddfda961410b2815b48679377baa0009ace173a2",
		},
	}

	bz, err := cfg.Codec.MarshalJSON(&gravityGenState)
	require.NoError(t, err)

	cfg.GenesisState[gravitytypes.ModuleName] = bz

	suite.Run(t, NewIntegrationTestSuite(cfg))
}
