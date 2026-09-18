package data

import "testing"

func TestIsSecretPointType(t *testing.T) {
	for _, typ := range []string{PointTypePass, PointTypeAuthToken,
		PointTypeEnrollToken, PointTypeSID, PointTypePSK} {
		if !IsSecretPointType(typ) {
			t.Errorf("%v should be a secret", typ)
		}
	}
	for _, typ := range []string{PointTypeDescription, PointTypeEmail,
		PointTypeTokenHash, PointTypePubKey} {
		if IsSecretPointType(typ) {
			t.Errorf("%v should not be a secret", typ)
		}
	}
}
