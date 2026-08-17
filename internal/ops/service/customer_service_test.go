package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestAllowedStageTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{model.OpsCustomerStageLead, model.OpsCustomerStageIntent, true},
		{model.OpsCustomerStageLead, model.OpsCustomerStageFormal, false},
		{model.OpsCustomerStageIntent, model.OpsCustomerStageTrial, true},
		{model.OpsCustomerStageTrial, model.OpsCustomerStageFormal, true},
		{model.OpsCustomerStageFormal, model.OpsCustomerStageClosed, true},
		{model.OpsCustomerStageClosed, model.OpsCustomerStageIntent, true},
	}
	for _, c := range cases {
		got := allowedStageTransitions[c.from][c.to]
		if got != c.ok {
			t.Fatalf("%s -> %s got %v want %v", c.from, c.to, got, c.ok)
		}
	}
}
