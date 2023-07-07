package indexer

import "golang.org/x/sync/errgroup"

func Run(cfg *Config) error {
	backend := NewBackend(cfg)

	wg := errgroup.Group{}
	wg.Go(func() error {
		return backend.Run()
	})
	wg.Go(func() error {
		return NewServer(cfg, backend.db).Run()
	})
	wg.Go(func() error {
		return NewGrpcServer(cfg, backend.db).Run()
	})

	return wg.Wait()
}
