package fabric

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/gov-spending/backend/internal/config"
)

type GatewayManager struct {
	config      *config.Config
	connections map[string]*ChannelConnection
	mu          sync.RWMutex
}

type ChannelConnection struct {
	Gateway    *client.Gateway
	GrpcConn   *grpc.ClientConn
	Contract   *client.Contract
	Network    *client.Network
	ChannelCfg config.ChannelConfig
}

func NewGatewayManager(cfg *config.Config) *GatewayManager {
	return &GatewayManager{
		config:      cfg,
		connections: make(map[string]*ChannelConnection),
	}
}

func (gm *GatewayManager) GetConnection(channelKey string) (*ChannelConnection, error) {
	gm.mu.RLock()
	conn, exists := gm.connections[channelKey]
	gm.mu.RUnlock()

	if exists {
		return conn, nil
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()

	if conn, exists = gm.connections[channelKey]; exists {
		return conn, nil
	}

	channelCfg, ok := gm.config.GetChannelConfig(channelKey)
	if !ok {
		return nil, fmt.Errorf("unknown channel: %s", channelKey)
	}

	conn, err := gm.createConnection(channelCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection for %s: %w", channelKey, err)
	}

	gm.connections[channelKey] = conn
	return conn, nil
}

func (gm *GatewayManager) createConnection(channelCfg config.ChannelConfig) (*ChannelConnection, error) {
	networkPath := gm.config.Fabric.NetworkPath
	domain := getDomain(channelCfg.CryptoPath)

	certPath := filepath.Join(networkPath, "crypto-config", channelCfg.CryptoPath,
		fmt.Sprintf("users/Admin@%s/msp/signcerts", domain))
	cert, err := loadCertificate(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate from %s: %w", certPath, err)
	}

	id, err := identity.NewX509Identity(channelCfg.MspID, cert)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	keyPath := filepath.Join(networkPath, "crypto-config", channelCfg.CryptoPath,
		fmt.Sprintf("users/Admin@%s/msp/keystore", domain))
	privateKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from %s: %w", keyPath, err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	grpcConn, err := gm.createGrpcConnection(channelCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	gateway, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(grpcConn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		grpcConn.Close()
		return nil, fmt.Errorf("failed to connect gateway: %w", err)
	}

	network := gateway.GetNetwork(channelCfg.Name)
	contract := network.GetContract(gm.config.Fabric.ChaincodeName)

	return &ChannelConnection{
		Gateway:    gateway,
		GrpcConn:   grpcConn,
		Contract:   contract,
		Network:    network,
		ChannelCfg: channelCfg,
	}, nil
}

func (gm *GatewayManager) createGrpcConnection(channelCfg config.ChannelConfig) (*grpc.ClientConn, error) {
	networkPath := gm.config.Fabric.NetworkPath

	tlsCertPath := filepath.Join(networkPath, "crypto-config", channelCfg.CryptoPath,
		fmt.Sprintf("peers/%s/tls/ca.crt", channelCfg.PeerHostAlias))

	tlsCert, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TLS certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(tlsCert) {
		return nil, fmt.Errorf("failed to add TLS certificate to pool")
	}

	transportCredentials := credentials.NewClientTLSFromCert(certPool, channelCfg.PeerHostAlias)

	return grpc.Dial(
		channelCfg.PeerEndpoint,
		grpc.WithTransportCredentials(transportCredentials),
	)
}

func (gm *GatewayManager) Close() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for key, conn := range gm.connections {
		conn.Gateway.Close()
		conn.GrpcConn.Close()
		delete(gm.connections, key)
	}
}

func (gm *GatewayManager) GetContract(channelKey string) (*client.Contract, error) {
	conn, err := gm.GetConnection(channelKey)
	if err != nil {
		return nil, err
	}
	return conn.Contract, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func loadCertificate(certDir string) (*x509.Certificate, error) {
	files, err := os.ReadDir(certDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		certPath := filepath.Join(certDir, file.Name())
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			continue
		}
		return identity.CertificateFromPEM(certPEM)
	}

	return nil, fmt.Errorf("no certificate found in %s", certDir)
}

func loadPrivateKey(keyDir string) (interface{}, error) {
	files, err := os.ReadDir(keyDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		keyPath := filepath.Join(keyDir, file.Name())
		keyPEM, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}
		return identity.PrivateKeyFromPEM(keyPEM)
	}

	return nil, fmt.Errorf("no private key found in %s", keyDir)
}

func getDomain(cryptoPath string) string {
	return filepath.Base(cryptoPath)
}

// =============================================================================
// Transaction Context Helpers
// =============================================================================

func WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}