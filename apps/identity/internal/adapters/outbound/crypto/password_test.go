package crypto

import "testing"

func TestPasswordHasherRoundTripAndMalformedInputs(t *testing.T) {
	hasher := NewPasswordHasher()
	encoded, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !hasher.Verify("correct horse battery staple", encoded) {
		t.Fatal("expected password to verify")
	}
	if hasher.Verify("wrong password", encoded) {
		t.Fatal("wrong password verified")
	}
	for _, malformed := range []string{"", "plain", "$argon2id$v=19$m=1,t=1,p=1$bad$bad", "$argon2i$v=19$m=1,t=1,p=1$bad$bad"} {
		if hasher.Verify("anything", malformed) {
			t.Fatalf("malformed hash verified: %q", malformed)
		}
	}
}
