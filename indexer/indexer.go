package indexer

import "golang.org/x/sync/errgroup"

func Run(cfg *Config) error {
	db := NewDb(cfg.Db.Engin, cfg.Db.Ds)
	wg := errgroup.Group{}

	if !cfg.Readonly {
		for _, chain := range cfg.Chains {
			backend := NewBackend(cfg.Headers, cfg.EntryPoints, chain, db)
			wg.Go(func() error {
				return backend.Run()
			})
		}
	}

	wg.Go(func() error {
		return NewServer(cfg, db).Run()
	})
	wg.Go(func() error {
		return NewGrpcServer(cfg, db).Run()
	})

	return wg.Wait()
}
