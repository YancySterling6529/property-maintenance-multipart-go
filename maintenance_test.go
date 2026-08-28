package main

import "testing"

func TestPlanUploadUsesLargerPartsForLargeInspectionMedia(t *testing.T) {
	small := planUpload("req-small", 100*1024*1024)
	large := planUpload("req-large", 600*1024*1024)
	if small.PartSize != 8*1024*1024 {
		t.Fatalf("small media part size = %d", small.PartSize)
	}
	if large.PartSize != 32*1024*1024 {
		t.Fatalf("large media part size = %d", large.PartSize)
	}
	if large.Key != "maintenance/req-large.bin" {
		t.Fatalf("unexpected object key %q", large.Key)
	}
}
