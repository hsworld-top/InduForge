package sceneasset

import (
	"strings"
	"testing"
)

func TestContentObjectReferenceGuardsIncludeTombstonedFileNodes(t *testing.T) {
	if !strings.Contains(contentObjectReferenceGuards, "scene_file_nodes") {
		t.Fatal("孤立对象判定必须检查场景文件节点引用")
	}
	if strings.Contains(contentObjectReferenceGuards, "deleted_at") {
		t.Fatal("孤立对象判定不能忽略 tombstone 文件节点，否则删除会违反外键约束")
	}
}
