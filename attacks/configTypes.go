package main

import (
	"fmt"
	"strings"
	"tpm_sync/tpm_learnRules"
	"tpm_sync/tpm_stimHandlers"
)

type BaseSettings struct {
	TpmType         string   `json:"tpm_type"`
	MaxSessionCount int      `json:"max_session_count"`
	MaxIterations   int      `json:"max_iterations"`
	MaxWorkerCount  int      `json:"max_worker_count"`
	LearnRules      []string `json:"learn_rules"`
	MConfigs        []int    `json:"m_configs"`
	LConfigs        []int    `json:"l_configs"`
}

type OverlappedSettings struct {
	BaseSettings
	KConfigs  [][]int `json:"k_configs"`
	N0Configs []int   `json:"n0_configs"`
}

type NonOverlappedSettings struct {
	BaseSettings
	KlastConfigs []int   `json:"klast_configs"`
	NConfigs     [][]int `json:"n_configs"`
}

func SettingsFactory(K []int, n_0 int, l int, m int, tpmType string, learnRule string) (MTPMSettings, error) {

	var stimHandler tpm_stimHandlers.TPMStimulationHandlers
	var ruleHandler tpm_learnRules.TPMLearnRuleHandler

	reverseParameters := false // this is because the no overlap os defined by the stimulus, so K[] is actually N[] and n_0 is actually k_last

	K = copySlice(K)

	switch parsed_tpmType := strings.ToUpper(tpmType); parsed_tpmType {
	case "PARTIALLY_CONNECTED":
		stimHandler = tpm_stimHandlers.PartialConnectionTPM{}
	case "FULLY_CONNECTED":
		stimHandler = tpm_stimHandlers.FullConnectionTPM{}
	case "NO_OVERLAP":
		stimHandler = tpm_stimHandlers.NoOverlapTPM{}
		reverseParameters = true
	}
	if stimHandler == nil {
		return MTPMSettings{}, fmt.Errorf("TPM type is invalid: %s", tpmType)
	}

	switch parsed_learnRule := strings.ToUpper(learnRule); parsed_learnRule {
	case "HEBBIAN":
		ruleHandler = tpm_learnRules.HebbianLearnRule{}
	case "ANTI-HEBBIAN":
		ruleHandler = tpm_learnRules.AntiHebbianLearnRule{}
	case "RANDOM-WALK":
		ruleHandler = tpm_learnRules.RandomWalkLearnRule{}
	}
	if ruleHandler == nil {
		return MTPMSettings{}, fmt.Errorf("TPM rule is invalid: %s", learnRule)
	}

	N := stimHandler.CreateStimulationStructure(K, n_0)
	if reverseParameters {
		aux := N
		N = K
		K = aux
	}

	return MTPMSettings{
		K:                   K,
		N:                   N,
		L:                   l,
		M:                   m,
		H:                   len(K),
		LearnRule:           learnRule,
		LinkType:            tpmType,
		learnRuleHandler:    ruleHandler,
		stimulationHandlers: stimHandler,
	}, nil
}

func copySlice(input []int) []int {
	copied := make([]int, len(input))
	copy(copied, input)
	return copied
}
