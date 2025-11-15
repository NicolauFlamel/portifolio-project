package services

import (
    "encoding/json"
    "errors"
    "fmt"

    "github.com/NicolauFlamel/portifolio-project-backend/internal/config"
    "github.com/NicolauFlamel/portifolio-project-backend/internal/fabric"
		"github.com/hyperledger/fabric-gateway/pkg/client"
)

type DocService struct {
    cfg config.NetConfig
}

func NewDocService(cfg config.NetConfig) *DocService { return &DocService{cfg: cfg} }

func orgToMSP(org string) (mspID, peerHost, peerTarget string) {
    switch org {
    case "org2":
        return "Org2MSP", "peer0.org2.example.com", "localhost:9051"
    default:
        return "Org1MSP", "peer0.org1.example.com", "localhost:7051"
    }
}

func (s *DocService) GetAll(channel, org string) ([]byte, error) {
    if channel == "" { channel = s.cfg.DefaultChannel }
    if org == "" { org = s.cfg.DefaultOrg }
    msp, host, target := orgToMSP(org)

    gw, contract, err := fabric.Connect(fabric.Params{
        Channel: channel, Chaincode: s.cfg.DefaultChaincode, Org: org,
        MSPID: msp, PeerHost: host, PeerTarget: target,
    })
    if err != nil { return nil, err }
    defer gw.Close()

    return contract.EvaluateTransaction("GetAllDocuments")
}

func (s *DocService) Get(channel, org, docID string) ([]byte, error) {
    if docID == "" { return nil, errors.New("docID required") }
    if channel == "" { channel = s.cfg.DefaultChannel }
    if org == "" { org = s.cfg.DefaultOrg }
    msp, host, target := orgToMSP(org)

    gw, contract, err := fabric.Connect(fabric.Params{
        Channel: channel, Chaincode: s.cfg.DefaultChaincode, Org: org,
        MSPID: msp, PeerHost: host, PeerTarget: target,
    })
    if err != nil { return nil, err }
    defer gw.Close()

    return contract.EvaluateTransaction("GetDocument", docID)
}

func (s *DocService) Create(channel, org string, body []byte) (string, error) {
    if channel == "" { channel = s.cfg.DefaultChannel }
    if org == "" { org = s.cfg.DefaultOrg }
    msp, host, target := orgToMSP(org)

    gw, contract, err := fabric.Connect(fabric.Params{
        Channel: channel, Chaincode: s.cfg.DefaultChaincode, Org: org,
        MSPID: msp, PeerHost: host, PeerTarget: target,
    })
    if err != nil { return "", err }
    defer gw.Close()

    // body is a JSON doc your chaincode expects: {docID,docType,sourceGov,payload,...}
    // pass raw JSON as string if your chaincode's CreateDocument takes a single JSON arg
    var pretty map[string]interface{}
    if err := json.Unmarshal(body, &pretty); err != nil {
        return "", fmt.Errorf("invalid JSON payload: %w", err)
    }
    // re-encode to compact JSON argument
    compact, _ := json.Marshal(pretty)

  	_, commit, err := contract.SubmitAsync(
    "CreateDocument",
    client.WithArguments(string(compact)),
		)
		if err != nil {
				return "", fmt.Errorf("submit proposal failed: %w", err)
		}

		// Wait for commit confirmation (optional but recommended)
		status, err := commit.Status()
		if err != nil {
				return "", fmt.Errorf("commit status error: %w", err)
		}
		if !status.Successful {
				return "", fmt.Errorf("commit failed with status code %d", int(status.Code))
		}

		return commit.TransactionID(), nil
}