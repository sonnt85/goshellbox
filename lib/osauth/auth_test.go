package osauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyPasswordHashPass(t *testing.T) {
	hashPassword := "$6$CMWxpgkq.ZosUW8N$gN/MkheCdS9SsPrFS6oOd/k.TMvY2KHztJE5pDMRdN35zr00dyxQr3pYGM4rtPPduUIrEFCwuB7oVgzDbiMfN." //nolint:gosec
	passwd := "123"

	result := new(OSAuth).VerifyPasswordHash(hashPassword, passwd)

	assert.True(t, result)
}

func TestVerifyPasswordHashFail(t *testing.T) {
	hashPassword := "$6$CMWxpgkq.ZosUW8N$gN/MkheCdS9SsPrFS6oOd/k.TMvY2KHztJE5pDMRdN35zr00dyxQr3pYGM4rtPPduUIrEFCwuB7oVgzDbiMfN." //nolint:gosec
	passwd := "test"

	result := new(OSAuth).VerifyPasswordHash(hashPassword, passwd)

	assert.False(t, result)
}

func TestVerifyPasswordHashMD5Pass(t *testing.T) {
	hashPassword := "$1$YW4a91HG$31CtH9bzW/oyJ1VOD.H/d/" //nolint:gosec
	passwd := "test"

	result := new(OSAuth).VerifyPasswordHash(hashPassword, passwd)

	assert.True(t, result)
}

func TestShowdow(t *testing.T) {
	hashPassword := "$y$j9T$X.jBzTjm8vo9pTzQ00fR20$kToSUE0rYx2bzgBVrX6tgG.ANSrPQsBNMvNZ3DfxU89"
	passwd := "testonly"

	result := new(OSAuth).VerifyPasswordHash(hashPassword, passwd)

	assert.True(t, result)

	// shadowfile := "/tmp/shadow"
	// f, e := os.Open(shadowfile)
	// assert.True(t, e == nil)
	// defer f.Close()
	// result := new(OSAuth).AuthUserFromShadow("root", "testonly", f)
	// assert.True(t, result)
}
