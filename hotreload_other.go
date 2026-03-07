//go:build !unix

package guardgo

func (e *Engine) startSignalHotReload() {}
