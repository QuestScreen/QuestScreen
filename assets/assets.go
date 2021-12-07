package assets

import "embed"

//go:embed *
var Data embed.FS

func MustRead(name string) []byte {
	ret, err := Data.ReadFile(name);
	if err != nil {
		panic(err)
	}
	return ret
}