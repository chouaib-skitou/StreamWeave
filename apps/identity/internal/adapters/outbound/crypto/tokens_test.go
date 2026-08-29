package crypto

import "testing"

func TestOpaqueTokensAreRandomAndHashed(t *testing.T) {
	service := NewTokenService()
	first, firstHash, err := service.NewOpaque(32)
	if err != nil {
		t.Fatal(err)
	}
	second, secondHash, err := service.NewOpaque(32)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || string(firstHash) == string(secondHash) {
		t.Fatal("opaque tokens are not random")
	}
	if string(firstHash) != string(service.HashOpaque(first)) {
		t.Fatal("opaque hash is not stable")
	}
	if _, _, err := service.NewOpaque(16); err == nil {
		t.Fatal("short opaque token accepted")
	}
}
