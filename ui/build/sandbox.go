package build

type Sandbox bool

const (
	noSandbox            = false
	globalSandbox        = false
	dumpvarsSandbox      = false
	soongSandbox         = false
	katiSandbox          = false
	katiCleanSpecSandbox = false
)

func (c *Cmd) sandboxSupported() bool {
	return false
}

func (c *Cmd) wrapSandbox() {
}
