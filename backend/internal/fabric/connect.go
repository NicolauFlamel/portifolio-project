package fabric

import (
	"fmt"
	"path/filepath"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

// Connect returns a Fabric contract instance for the given channel and chaincode name
func Connect(channel string, chaincode string, identity string) (*gateway.Contract, error) {
	ccpPath := filepath.Clean(fmt.Sprintf("./config/fabric-%s.yaml", channel))

	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %v", err)
	}

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, identity),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gateway: %v", err)
	}

	network, err := gw.GetNetwork(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %v", err)
	}

	contract := network.GetContract(chaincode)
	return contract, nil
}