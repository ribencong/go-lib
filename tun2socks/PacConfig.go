package tun2socks

var (
	SysConfig = &PacConfig{
		IsGlobal: false,
		PacCache: make(map[string]struct{}),
	}
)

type PacConfig struct {
	IsGlobal bool
	PacCache map[string]struct{}
}

func (c *PacConfig) NeedProxy(target string) bool {
	if c.IsGlobal {
		return true
	}

	if _, ok := c.PacCache[target]; ok {
		return true
	}

	return false
}
