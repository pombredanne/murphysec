package semerr

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSemErr(t *testing.T) {
	var ErrT = New("TE")
	e := ErrT.Decorate(errors.New("awsl"))
	assert.True(t, errors.Is(e, ErrT))
	assert.True(t, errors.Is(ErrT, e))
	assert.True(t, errors.Is(e, e))
	assert.False(t, errors.Is(errors.New("awsl"), errors.New("awsl")))
	var ErrE = New("EE")
	e = ErrE.Decorate(e)
	assert.True(t, errors.Is(e, ErrE))
	assert.True(t, errors.Is(e, ErrT))
	var ErrQ = New("QE")
	assert.False(t, errors.Is(e, ErrQ))
}