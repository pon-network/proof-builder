#!/bin/bash
set -Eeuo pipefail

# Source environment variables
source ./vars.env

# Set variables
data_dir="/path/to/data/dir"
network_port="12345"
http_port="8545"
auth_port="8551"
beacon_endpoints="http://127.0.0.1:5052"
builder_port="10000"
bls_key=$PROOF_BUILDER_BLS_SECRET_KEY
wallet_private_key=$PROOF_BUILDER_WALLET_PRIVATE_KEY
builder_accesspoint="https://builder.ponrelay.com"
relay_endpoint="https://0xC74A620dd38e19B89395841D50e16D1e215C397e@relayer-goerli.ponrelay.com"
rpbs_port="3000"

# Function to check if a specific process is running
check_process() {
    local process_name="$1"
    if pgrep -x "$process_name" >/dev/null; then
        return 0  # Process is running
    else
        return 1  # Process is not running
    fi
}

# Check if Docker daemon is running
if ! check_process "docker"; then
    echo "Docker daemon is not running. Starting Docker daemon..."
    
    # Start Docker daemon (modify this command as needed based on your system)
    sudo systemctl start docker
    
    # Wait for a few seconds to allow Docker to start (adjust as needed)
    sleep 5
    
    # Check again if Docker daemon is running
    if check_process "docker"; then
        echo "Docker daemon is now running."
    else
        echo "Failed to start Docker daemon."
    fi
fi


# Check if RPBS service container is running
if sudo docker ps --format '{{.Image}}' | grep -q "blockswap/rpbs-hosted-service"; then
    echo "RPBS service container is already running."
else
    echo "RPBS service container is not running. Starting RPBS service..."
    
    # Start RPBS Hosted Service
    sudo docker run -d -p "127.0.0.1:$rpbs_port:$rpbs_port" --name rpbs-hosted-service blockswap/rpbs-hosted-service
    
    # Wait for a few seconds to allow the container to start (adjust as needed)
    sleep 10
    
    # Check again if the container is running
    if sudo docker ps --format '{{.Image}}' | grep -q "blockswap/rpbs-hosted-service"; then
        echo "RPBS service container is now running."
    else
        echo "Failed to start RPBS service container."
    fi
fi

# Start PoN Proof Builder
geth \
    --mainnet \
    --datadir "$data_dir" \
    --http \
    --http.addr 0.0.0.0 \
    --http.port "$http_port" \
    --http.vhosts=* \
    --http.api="engine,eth,web3,net,debug,mev" \
    --syncmode=full \
    --authrpc.port "$auth_port" \
    --authrpc.vhosts=* \
    --authrpc.addr 0.0.0.0 \
    --builder \
    --builder.beacon_endpoints "$beacon_endpoints" \
    --builder.relay_endpoint "$relay_endpoint" \
    --builder.secret_key "$bls_key" \
    --builder.wallet_private_key "$wallet_private_key" \
    --builder.public_accesspoint "$builder_accesspoint" \
    --builder.listen_addr "127.0.0.1:$builder_port" \
    --builder.rpbs "http://localhost:$rpbs_port" \
    --builder.metrics \
    --builder.bundles 
    # --builder.metrics_reset \
    # --builder.bundles_reset

# Display completion messages
echo "Completed PoN Proof Builder start"
echo "PoN Builder is running on http://localhost:$builder_port"
echo "PoN Builder public access point is $builder_accesspoint"
echo "PoN Builder is connected to beacon node at $beacon_endpoints"
echo "PoN Builder is connected to relay at $relay_endpoint"
