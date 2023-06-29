package web3

import "github.com/ethereum/go-ethereum/ethclient"

type Web3 struct {
	client *ethclient.Client
}

func NewWeb3Client(url string) (*Web3, error) {
	cli, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	return &Web3{client: cli}, nil
}

func (w3 *Web3) Cli() *ethclient.Client {
	return w3.client
}
