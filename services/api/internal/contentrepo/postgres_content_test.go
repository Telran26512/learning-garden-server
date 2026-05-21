package contentrepo

import (
	"strings"
	"testing"
)

func TestContentGraphRelationsSQLDoesNotCastIndexedColumns(t *testing.T) {
	for _, forbidden := range []string{"source_id::text = ANY", "target_id::text = ANY"} {
		if strings.Contains(contentGraphRelationsSQL, forbidden) {
			t.Fatalf("contentGraphRelationsSQL contains %q, which casts indexed columns", forbidden)
		}
	}
	if !strings.Contains(contentGraphRelationsSQL, "source_id = ANY($1::uuid[])") {
		t.Fatalf("contentGraphRelationsSQL must compare source_id as uuid:\n%s", contentGraphRelationsSQL)
	}
	if !strings.Contains(contentGraphRelationsSQL, "target_id = ANY($1::uuid[])") {
		t.Fatalf("contentGraphRelationsSQL must compare target_id as uuid:\n%s", contentGraphRelationsSQL)
	}
}
