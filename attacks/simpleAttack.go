package main

import (
	"fmt"
	"math/rand"

	"tpm_sync/tpm_core"
)

func testSimpleAttack(verbose bool, tpmSettings MTPMSettings, localRand *rand.Rand) int {

	// tpmSettings := MTPMSettings{
	// 	K:                   []int{8, 2}, // Example structure, adjust as needed
	// 	N:                   []int{7, 4},
	// 	L:                   3,
	// 	M:                   1,
	// 	H:                   2,
	// 	LearnRule:           "HEBBIAN",
	// 	LinkType:            "NO_OVERLAP",
	// 	stimulationHandlers: tpm_stimHandlers.NoOverlapTPM{},
	// 	learnRuleHandler:    tpm_learnRules.HebbianLearnRule{},
	// }

	state_A := newRandomState(tpmSettings, localRand)
	state_B := newRandomState(tpmSettings, localRand)

	state_E := newRandomState(tpmSettings, localRand)

	stopSync := false
	compareWeightsResult := 0
	threshold := 100_000
	for !stopSync {

		if state_A.SimIterations >= threshold {
			break
		}

		input_stimulus := tpm_core.CreateRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M, localRand)
		stimulate(&state_A, input_stimulus)
		stimulate(&state_B, input_stimulus)
		stimulate(&state_E, input_stimulus)
		if state_A.networkOutput == state_B.networkOutput {
			learn(&state_A, state_B.networkOutput)
			learn(&state_B, state_A.networkOutput)
			learnSimpleAttack(&state_E, state_A.networkOutput, state_B.networkOutput)
		}
		compareWeightsResult = CompareWeightsSimpleAttack(tpmSettings, state_A, state_B, state_E)
		stopSync = compareWeightsResult != 0
	}
	if verbose {
		if compareWeightsResult < 0 {
			fmt.Println("[!] Simple attack was successful!")
			if compareWeightsResult == -2 {
				fmt.Println("[!!] Simple attack on sync at the same time as B.")
			}
		}
		if compareWeightsResult > 0 {
			fmt.Println("A and B on sync succesfully.")
		}
		if compareWeightsResult == 0 {
			fmt.Println("A and B could not sync.")
		}
		fmt.Println("A: ")
		fmt.Println(state_A.Weights)
		fmt.Println("B: ")
		fmt.Println(state_B.Weights)
		fmt.Println("E: ")
		fmt.Println(state_E.Weights)
	}

	return compareWeightsResult
}
func learnSimpleAttack(sessionState *LocalMTPMSessionState, output_A, output_B int) {
	settings := sessionState.Settings

	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, sessionState.Weights[layer], sessionState.inputBuffer[layer], sessionState.outputBuffer[layer], output_A, output_B)
	}
	sessionState.LearnIterations += 1
}

func learnSimpleAttackReducedState(settings MTPMSettings, sessionState *reducedMTPMState, output_A, output_B int) {
	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, sessionState.Weights[layer], sessionState.inputBuffer[layer], sessionState.outputBuffer[layer], output_A, output_B)
	}
}

func CompareWeightsSimpleAttack(tpmSettings MTPMSettings, state_A, state_B, state_E LocalMTPMSessionState) int {
	state := 0
	if compareWeightsFromStates(tpmSettings, state_A, state_E) {
		state = -1
	}
	if compareWeightsFromStates(tpmSettings, state_A, state_B) {
		if state < 0 {
			return -2
		}
		return 1
	}

	return state
}
