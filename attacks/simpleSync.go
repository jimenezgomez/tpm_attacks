package main

import (
	"fmt"
	"math/rand"

	"tpm_sync/tpm_core"
	"tpm_sync/tpm_learnRules"
	"tpm_sync/tpm_stimHandlers"
)

func testSimpleSync(localRand *rand.Rand) {

	tpmSettings := MTPMSettings{
		K:                   []int{3}, // Example structure, adjust as needed
		N:                   []int{3},
		L:                   3,
		M:                   1,
		H:                   1,
		LearnRule:           "HEBBIAN",
		LinkType:            "NO_OVERLAP",
		stimulationHandlers: tpm_stimHandlers.NoOverlapTPM{},
		learnRuleHandler:    tpm_learnRules.HebbianLearnRule{},
	}

	state_A := newRandomState(tpmSettings, localRand)
	state_B := newRandomState(tpmSettings, localRand)

	for !compareWeightsFromStates(tpmSettings, state_A, state_B) {
		input_stimulus := tpm_core.CreateRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M, localRand)
		stimulate(&state_A, input_stimulus)
		stimulate(&state_B, input_stimulus)
		if state_A.networkOutput == state_B.networkOutput {
			learn(&state_A, state_B.networkOutput)
			learn(&state_B, state_A.networkOutput)
		}
		fmt.Println(state_A.SimIterations)
	}

}
