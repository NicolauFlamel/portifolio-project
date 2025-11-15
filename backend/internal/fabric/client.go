package fabric

import (
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/hyperledger/fabric-gateway/pkg/client"
    "github.com/hyperledger/fabric-gateway/pkg/identity"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

type Params struct {
    Channel    string
    Chaincode  string
    Org        string // "org1" | "org2"
    MSPID      string // e.g. "Org1MSP"
    PeerHost   string // e.g. "peer0.org1.example.com"
    PeerTarget string // e.g. "localhost:7051"
}

func Connect(p Params) (*client.Gateway, *client.Contract, error) {
    // Wallet path for this org (kept in network/)
    mspBase := filepath.Clean(fmt.Sprintf("../blockchain/network/wallet/%s/appUser/msp", p.Org))
    certPath := filepath.Join(mspBase, "signcerts", "cert.pem")
    keyDir   := filepath.Join(mspBase, "keystore")

    // Peer TLS CA (from org1 peer — OK if you only hit org1’s peer)
    tlsCertPath := filepath.Clean("../blockchain/network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt")

    // --- identity
    certPEM, err := os.ReadFile(certPath)
    if err != nil { return nil, nil, err }
    block, _ := pem.Decode(certPEM)
    x509Cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil { return nil, nil, err }
    id, err := identity.NewX509Identity(p.MSPID, x509Cert)
    if err != nil { return nil, nil, err }

    files, err := os.ReadDir(keyDir)
    if err != nil || len(files) == 0 { return nil, nil, fmt.Errorf("no private key in %s", keyDir) }
    keyPEM, err := os.ReadFile(filepath.Join(keyDir, files[0].Name()))
    if err != nil { return nil, nil, err }
    keyBlock, _ := pem.Decode(keyPEM)
    if keyBlock == nil { return nil, nil, fmt.Errorf("invalid key PEM") }
		key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
				// fallback: EC private key
				key, err = x509.ParseECPrivateKey(keyBlock.Bytes)
				if err != nil {
						return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
				}
		}
    signer, err := identity.NewPrivateKeySign(key)
    if err != nil { return nil, nil, err }

    // --- TLS / gRPC
    peerTLS, err := os.ReadFile(tlsCertPath)
    if err != nil { return nil, nil, err }
    cp := x509.NewCertPool(); cp.AppendCertsFromPEM(peerTLS)
    creds := credentials.NewClientTLSFromCert(cp, p.PeerHost)

    // note: grpc.Dial is deprecated; NewClient is preferred, but Dial still works. Use whichever you’ve already updated to.
    conn, err := grpc.NewClient(p.PeerTarget, grpc.WithTransportCredentials(creds))
    if err != nil { return nil, nil, err }

    gw, err := client.Connect(
        id,
        client.WithSign(signer),
        client.WithClientConnection(conn),
        client.WithEvaluateTimeout(5*time.Second),
        client.WithEndorseTimeout(15*time.Second),
        client.WithSubmitTimeout(5*time.Second),
    )
    if err != nil { return nil, nil, err }

    network := gw.GetNetwork(p.Channel)
    contract := network.GetContract(p.Chaincode)
    return gw, contract, nil
}