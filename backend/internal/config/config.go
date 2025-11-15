package config

type NetConfig struct {
    DefaultChannel  string
    DefaultChaincode string
    DefaultOrg      string
}

func New() NetConfig {
    return NetConfig{
        DefaultChannel:   "union-channel",
        DefaultChaincode: "publicdocs",
        DefaultOrg:       "org1", 
    }
}