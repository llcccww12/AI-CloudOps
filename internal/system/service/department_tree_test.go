package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestBuildDepartmentTree(t *testing.T) {
	t.Parallel()

	nodes := []*model.Department{
		{Model: model.Model{ID: 1}, Name: "总部", Code: "HQ", ParentID: 0, Sort: 1},
		{Model: model.Model{ID: 2}, Name: "研发", Code: "RD", ParentID: 1, Sort: 1},
		{Model: model.Model{ID: 3}, Name: "后端", Code: "BE", ParentID: 2, Sort: 1},
		{Model: model.Model{ID: 4}, Name: "市场", Code: "MKT", ParentID: 1, Sort: 2},
	}

	tree := buildDepartmentTree(nodes)
	if len(tree) != 1 {
		t.Fatalf("root count = %d, want 1", len(tree))
	}
	if tree[0].Name != "总部" {
		t.Fatalf("root name = %s, want 总部", tree[0].Name)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("children count = %d, want 2", len(tree[0].Children))
	}
	if tree[0].Children[0].Name != "研发" || tree[0].Children[1].Name != "市场" {
		t.Fatalf("unexpected children: %+v", tree[0].Children)
	}
	if len(tree[0].Children[0].Children) != 1 || tree[0].Children[0].Children[0].Name != "后端" {
		t.Fatalf("unexpected grandchild: %+v", tree[0].Children[0].Children)
	}
}
