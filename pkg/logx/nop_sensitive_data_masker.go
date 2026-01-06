package logx

type NopSensitiveDataMasker struct{}

func NewNopSensitiveDataMasker() NopSensitiveDataMasker {
	return NopSensitiveDataMasker{}
}

func (NopSensitiveDataMasker) Mask(input []byte) []byte {
	return input
}
