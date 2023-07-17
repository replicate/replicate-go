package replicate

type Status string

const (
	Starting   Status = "starting"
	Processing Status = "processing"
	Succeeded  Status = "succeeded"
	Failed     Status = "failed"
	Canceled   Status = "canceled"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) Terminated() bool {
	return s == Succeeded || s == Failed || s == Canceled
}
