#!/bin/bash -eux

# Using an already running chain, starts a governance proposal to upgrade to a new binary version,
# votes 'yes' on that proposal, waits to reach to reach an upgrade height and kills the process id
# received by the parameter $KATANAD_V1_PID_FILE

# USAGE: KATANAD_V1_PID_FILE=$katana_pid_file ./upgrade-test-single-node.sh

CWD="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

KATANAD_V1_PID_FILE="${KATANAD_V1_PID_FILE:-""}"

if [ ! -f $KATANAD_V1_PID_FILE ]; then
  echo "You need to specify the file with process id of katanad v1 inside of a file by setting KATANAD_V1_PID_FILE"
  exit 1
fi

CHAIN_ID="${CHAIN_ID:-888}"
CHAIN_DIR="${CHAIN_DIR:-$CWD/node-data}"
NODE_NAME="${NODE_NAME:-n0}"
LOG_LEVEL="${LOG_LEVEL:-info}"
NODE_URL="${NODE_URL:-"tcp://localhost:26657"}"
BLOCK_TIME="${BLOCK_TIME:-6}"
UPGRADE_TITLE="${UPGRADE_TITLE:-cosmwasm}"

KATANAD_BUILD_PATH="${KATANAD_BUILD_PATH:-$CWD/katanad-builds}"
KATANAD_BIN_V1="${KATANAD_BIN_V1:-$KATANAD_BUILD_PATH/katanad-fix-testnet-halt}"
KATANAD_BIN_V2="${KATANAD_BIN_V2:-$KATANAD_BUILD_PATH/katanad-cosmwasm}"

VOTING_PERIOD=${VOTING_PERIOD:-8}

hdir="$CHAIN_DIR/$CHAIN_ID"

# Loads another sources
. $CWD/blocks.sh

# Folders for nodes
nodeDir="$hdir/$NODE_NAME"

# Home flag for folder
nodeHomeFlag="--home $nodeDir"
nodeUrlFlag="--node $NODE_URL"

# Common flags
kbt="--keyring-backend test"
cid="--chain-id $CHAIN_ID"

CURRENT_HEIGHT=$(CHAIN_ID=$CHAIN_ID KATANAD_BIN=$KATANAD_BIN_V1 get_block_current_height)
echo blockchain CURRENT_HEIGHT is $CURRENT_HEIGHT

UPGRADE_HEIGHT=$(($CURRENT_HEIGHT + 30))
echo blockchain UPGRADE_HEIGHT is $UPGRADE_HEIGHT

proposal_path=$CWD/proposal.json

admin_addr=$($KATANAD_BIN_V2 $kbt $nodeHomeFlag keys show admin -a)

echo '
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "katana10d07y265gmmuvt4z0w9aw880jnsr700jg5w6jp",
      "plan": {
        "name": "'$UPGRADE_TITLE'",
        "height": "'$UPGRADE_HEIGHT'",
        "info": "{\"binaries\": {\"linux/amd64\":\"https://example.com/gaia.zip?checksum=sha256:aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f\",\"linux/arm64\":\"https://example.com/gaia.zip?checksum=sha256:aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f\",\"darwin/amd64\":\"https://example.com/gaia.zip?checksum=sha256:aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f\"}}"
      }
    }
  ],
  "deposit": "1000000000ukatana"
}

' > $proposal_path

echo "Submitting the software-upgrade proposal..."
$KATANAD_BIN_V1 tx gov submit-proposal $proposal_path -b block $nodeHomeFlag --from admin $nodeUrlFlag $kbt --yes --fees 100000ukatana --gas 300000

# $KATANAD_BIN_V1 tx gov submit-proposal software-upgrade $proposal_path $UPGRADE_TITLE  --title yeet --description megayeet $cid --deposit 1000000000ukatana \
#   --upgrade-height $UPGRADE_HEIGHT --upgrade-info "doing an upgrade '-'" \
#   -b block $nodeHomeFlag --from admin $nodeUrlFlag $kbt --yes --fees 100000ukatana

##
PROPOSAL_ID=$($KATANAD_BIN_V1 q gov $nodeUrlFlag proposals -o json | jq ".proposals[-1].id" -r)
echo proposal ID is $PROPOSAL_ID

echo "Voting on proposaal : $PROPOSAL_ID"
$KATANAD_BIN_V1 tx gov vote $PROPOSAL_ID yes -b block --from admin $nodeHomeFlag $cid $nodeUrlFlag $kbt  --yes --fees 100000ukatana

echo "..."
echo "Finished voting on the proposal"
echo "Waiting to reach the upgrade height"
echo "..."
CHAIN_ID=$CHAIN_ID KATANAD_BIN=$KATANAD_BIN_V1 wait_until_block $UPGRADE_HEIGHT

CURRENT_HEIGHT=$(CHAIN_ID=$CHAIN_ID KATANAD_BIN=$KATANAD_BIN_V1 get_block_current_height)

echo "Reached upgrade block height: $CURRENT_HEIGHT == $UPGRADE_HEIGHT"

node_pid_value=$(cat $KATANAD_V1_PID_FILE)

echo "Kill the process ID '$node_pid_value'"

kill -s 15 $node_pid_value

sleep 5

echo "...."
echo "Upgrade finished"
echo "...."
sleep $VOTING_PERIOD

# Starts a different file for logging
nodeLogPath=$hdir.katanad-v2.log

$KATANAD_BIN_V2 $nodeHomeFlag start --grpc.address="0.0.0.0:9090" --grpc-web.enable=false --log_level $LOG_LEVEL > $nodeLogPath 2>&1 &

# Gets the node pid
echo $! > $KATANAD_V1_PID_FILE

echo
echo "Logs:"
echo "  * tail -f $nodeLogPath"
