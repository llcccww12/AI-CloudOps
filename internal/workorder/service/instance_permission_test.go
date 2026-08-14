package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestCanUserOperate(t *testing.T) {
	t.Parallel()

	s := &instanceService{logger: zap.NewNop()}
	assignee := 10
	other := 20
	zeroAssignee := 0

	stepUser := &model.ProcessStep{
		AssigneeType: model.AssigneeTypeUser,
		AssigneeIDs:  []int{10, 11},
	}
	stepEmpty := &model.ProcessStep{
		AssigneeType: model.AssigneeTypeUser,
		AssigneeIDs:  nil,
	}
	stepSystem := &model.ProcessStep{
		AssigneeType: model.AssigneeTypeGroup,
		AssigneeIDs:  []int{10},
	}
	stepNoType := &model.ProcessStep{
		AssigneeType: "",
		AssigneeIDs:  []int{10},
	}

	cases := []struct {
		name       string
		step       *model.ProcessStep
		operatorID int
		assigneeID *int
		want       bool
	}{
		{"nil step", nil, 10, nil, false},
		{"assigned match", stepUser, 10, &assignee, true},
		{"assigned mismatch", stepUser, other, &assignee, false},
		{"assigned zero treated as unassigned", stepUser, 10, &zeroAssignee, true},
		{"user in list", stepUser, 10, nil, true},
		{"user not in list", stepUser, other, nil, false},
		{"empty assignees deny process", stepEmpty, 10, nil, false},
		{"system with list", stepSystem, 10, nil, true},
		{"system not in list", stepSystem, other, nil, false},
		{"empty type deny", stepNoType, 10, nil, false},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := s.canUserOperate(c.step, c.operatorID, c.assigneeID)
			if got != c.want {
				t.Fatalf("got %v want %v", got, c.want)
			}
		})
	}
}

func TestCanUserClaim(t *testing.T) {
	s := &instanceService{logger: zap.NewNop()}
	stepListed := &model.ProcessStep{AssigneeIDs: []int{10, 11}}
	stepEmpty := &model.ProcessStep{}

	if !s.canUserClaim(stepEmpty, 99) {
		t.Fatal("unconfigured step should allow claim")
	}
	if !s.canUserClaim(stepListed, 10) {
		t.Fatal("listed user should claim")
	}
	if s.canUserClaim(stepListed, 99) {
		t.Fatal("unlisted user should not claim")
	}
	if s.canUserClaim(nil, 10) {
		t.Fatal("nil step should not claim")
	}
}
