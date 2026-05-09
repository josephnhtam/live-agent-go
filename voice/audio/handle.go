package audio

type Handle interface {
	SetVolume(v float64)
	Stop()
}
