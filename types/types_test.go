package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionStatusString(t *testing.T) {
	assert.Equal(t, "STORED", STORED.String(), "STORED constant should match")
	assert.Equal(t, "CANCELED", CANCELED.String(), "CANCELED constant should match")
	assert.Equal(t, "SPEDUP", SPEDUP.String(), "SPEDUP constant should match")
	assert.Equal(t, "FAILED", FAILED.String(), "FAILED constant should match")
	assert.Equal(t, "BROADCASTED", BROADCASTED.String(), "BROADCASTED constant should match")
}
