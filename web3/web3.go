package web3

import "github.com/ethereum/go-ethereum/ethclient"

type Web3 struct {
	url    string
	client *ethclient.Client
}

func NewWeb3Client(url string) (*Web3, error) {
	cli, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Web3{client: cli, url: url}, nil
}

func (w3 *Web3) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		w3.client.Client().SetHeader(k, v)
	}
}

func (w3 *Web3) Cli() *ethclient.Client {
	return w3.client
}

func (w3 *Web3) Url() string {
	return w3.url
}
