//go:build unix

package guardgo

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func (e *Engine) startSignalHotReload() {
	if !e.cfg.AutoReloadOnSIGHUP || e.ruleset == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)

	e.hotReloadStop = func() {
		signal.Stop(ch)
		cancel()
	}

	go e.ruleset.ReloadOnSignal(ctx, ch, e.cfg.OnError)
}
