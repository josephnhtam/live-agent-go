package audio

type Options struct {
	volume     float64
	duckVolume float64
	duck       bool
	loop       bool
}

func NewOptions() *Options {
	return &Options{
		volume:     1.0,
		duckVolume: 0.2,
	}
}

func (o *Options) WithLoop() *Options {
	o.loop = true
	return o
}

func (o *Options) WithVolume(v float64) *Options {
	o.volume = v
	return o
}

func (o *Options) WithDuckVolume(v float64) *Options {
	o.duckVolume = v
	return o
}

func (o *Options) WithDuck() *Options {
	o.duck = true
	return o
}

func (o *Options) Volume() float64     { return o.volume }
func (o *Options) DuckVolume() float64 { return o.duckVolume }
func (o *Options) Duck() bool          { return o.duck }
func (o *Options) Loop() bool          { return o.loop }
