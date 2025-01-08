package ecies

type KeyOption struct {
	Dirpath            string `json:"dirpath" default:"."`
	PrivateKeyFilename string `json:"private.pem" default:"private.pem"`
	PublicKeyFilename  string `json:"public.pem" default:"public.pem"`
}

func WithDirpath(dirpath string) func(*KeyOption) {
	return func(o *KeyOption) {
		o.Dirpath = dirpath
	}
}

func WithPrivateKeyFilename(filename string) func(*KeyOption) {
	return func(o *KeyOption) {
		o.PrivateKeyFilename = filename
	}
}

func WithPublicKeyFilename(filename string) func(*KeyOption) {
	return func(o *KeyOption) {
		o.PublicKeyFilename = filename
	}
}
