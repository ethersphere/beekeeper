package swap

import (
	"context"
	"fmt"
)

type Service struct{}

func NewService(backend string, privateKeyHex string, tokenAddress string) (s *Service, err error) {
	fmt.Println("backend", backend)
	fmt.Println("privateKeyHex", privateKeyHex)
	fmt.Println("tokenAddress", tokenAddress)
	return
}

func (s *Service) Fund(ctx context.Context, name string) (err error) {
	fmt.Printf("%s funded\n", name)
	return
}
