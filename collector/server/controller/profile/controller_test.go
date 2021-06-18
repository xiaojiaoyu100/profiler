package profile

import "testing"

func TestUploadPath(t *testing.T) {
	t.Logf(UploadPath("abc", "bcf", "cpu", "efg"))
}
