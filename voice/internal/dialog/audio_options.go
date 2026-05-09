package dialog

type AudioOptions struct {
	volume     float64
	duckVolume float64
	duck       bool
	loop       bool
}

func NewAudioOptions() *AudioOptions {
	return &AudioOptions{
		volume:     1.0,
		duckVolume: 0.2,
	}
}

func (o *AudioOptions) WithLoop() *AudioOptions {
	o.loop = true
	return o
}

func (o *AudioOptions) WithVolume(v float64) *AudioOptions {
	o.volume = v
	return o
}

func (o *AudioOptions) WithDuckVolume(v float64) *AudioOptions {
	o.duckVolume = v
	return o
}

func (o *AudioOptions) WithDuck() *AudioOptions {
	o.duck = true
	return o
}

func (o *AudioOptions) Volume() float64     { return o.volume }
func (o *AudioOptions) DuckVolume() float64 { return o.duckVolume }
func (o *AudioOptions) Duck() bool          { return o.duck }
func (o *AudioOptions) Loop() bool          { return o.loop }
