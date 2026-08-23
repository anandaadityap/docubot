package util

import "testing"

func TestSlugify_TokoKita(t *testing.T) {
	if got := Slugify("Toko Kita!"); got != "toko-kita" {
		t.Fatalf("Slugify(Toko Kita!) = %q", got)
	}
}

func TestSlugify_DashesFallbackSeed(t *testing.T) {
	if got := Slugify("---"); got != "" {
		t.Fatalf("Slugify(---) = %q, want empty so AllocateSlug can fall back", got)
	}
}

func TestAllocateSlug_Dashes(t *testing.T) {
	got := AllocateSlug("---", 1, func(string) bool { return false })
	if got != "bot" {
		t.Fatalf("AllocateSlug(---) = %q, want bot", got)
	}
}

func TestAllocateSlug_Reserved(t *testing.T) {
	got := AllocateSlug("Admin", 3, func(string) bool { return false })
	if got != "bot" {
		t.Fatalf("AllocateSlug(Admin) = %q, want bot", got)
	}
}

func TestAllocateSlug_Collision(t *testing.T) {
	taken := map[string]bool{"nanda": true, "bot-2": true}
	got := AllocateSlug("Nanda", 2, func(s string) bool { return taken[s] })
	if got != "bot-2-2" {
		t.Fatalf("got %q, want bot-2-2", got)
	}
}

func TestValidSlug(t *testing.T) {
	if ValidSlug("ab") || ValidSlug("admin") || ValidSlug("Toko") || !ValidSlug("toko-kita") {
		t.Fatal("ValidSlug cases failed")
	}
}
