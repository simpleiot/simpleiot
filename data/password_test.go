package data

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal("HashPassword returned error:", err)
	}

	if !PasswordIsHashed(hash) {
		t.Fatal("hash not recognized as hashed:", hash)
	}

	ok, needsRehash := CheckPassword(hash, "secret")
	if !ok {
		t.Fatal("correct password rejected against hash")
	}
	if needsRehash {
		t.Fatal("hashed password should not need rehash")
	}

	ok, _ = CheckPassword(hash, "wrong")
	if ok {
		t.Fatal("wrong password accepted against hash")
	}
}

func TestPasswordLegacyPlaintext(t *testing.T) {
	if PasswordIsHashed("secret") {
		t.Fatal("plaintext recognized as hashed")
	}

	ok, needsRehash := CheckPassword("secret", "secret")
	if !ok {
		t.Fatal("correct legacy password rejected")
	}
	if !needsRehash {
		t.Fatal("legacy password match should request rehash")
	}

	ok, _ = CheckPassword("secret", "wrong")
	if ok {
		t.Fatal("wrong legacy password accepted")
	}
}
