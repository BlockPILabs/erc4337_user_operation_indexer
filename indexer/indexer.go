package indexer

import "golang.org/x/sync/errgroup"

func Run(cfg *Config) error {
	backend := NewBackend(cfg)
	server := NewServer(cfg, backend.db)

	wg := errgroup.Group{}
	wg.Go(func() error {
		return backend.Run()
	})
	wg.Go(func() error {
		return server.Run()
	})

	return wg.Wait()
}
