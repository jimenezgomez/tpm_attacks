package main

import (
	"fmt"
	"math"
	"math/rand"

	"tpm_sync/tpm_core"
)

func testGeomAttack(verbose bool, tpmSettings MTPMSettings, localRand *rand.Rand) int {

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
			learnGeomAttack(&state_E, state_A.networkOutput, state_B.networkOutput, localRand)
		}
		compareWeightsResult = CompareWeightsSimpleAttack(tpmSettings, state_A, state_B, state_E)
		stopSync = compareWeightsResult != 0
	}
	if verbose {
		if compareWeightsResult < 0 {
			fmt.Println("[!] Geometric attack was successful!")
			if compareWeightsResult == -2 {
				fmt.Println("[!!] Geometric attack on sync at the same time as B.")
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
func learnGeomAttack(attackerSessionState *LocalMTPMSessionState, output_A, output_B int, localRand *rand.Rand) {
	settings := attackerSessionState.Settings

	if attackerSessionState.networkOutput == output_A {
		for layer := 0; layer < settings.H; layer++ {
			settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, attackerSessionState.Weights[layer], attackerSessionState.inputBuffer[layer], attackerSessionState.outputBuffer[layer], output_A, output_B)
		}
		return
	}

	lastLayerIndex := attackerSessionState.Settings.H - 1
	lastLayer := attackerSessionState.Weights[lastLayerIndex]

	minFieldIndexes := []int{}
	minAbsField := float64(settings.L*settings.N[lastLayerIndex] + 1) //setup with the max possible field sum
	for neuronIndex := 0; neuronIndex < settings.K[lastLayerIndex]; neuronIndex++ {
		//get the last layer local field
		absLocalField := math.Abs(tpm_core.NeuronLocalField(settings.N[lastLayerIndex], lastLayer[neuronIndex], attackerSessionState.inputBuffer[lastLayerIndex][neuronIndex]))
		// localField := NeuronDotProd(settings.N[lastLayerIndex], lastLayer[neuronIndex], sessionState.inputBuffer[lastLayerIndex][neuronIndex])
		if absLocalField < minAbsField {
			minFieldIndexes = []int{neuronIndex}
			minAbsField = absLocalField
			continue
		}
		if absLocalField == minAbsField {
			minFieldIndexes = append(minFieldIndexes, neuronIndex)
		}
	}

	selectedIndex := minFieldIndexes[localRand.Intn(len(minFieldIndexes))]

	attackerSessionState.outputBuffer[lastLayerIndex][selectedIndex] *= -1 //we assume we need to flip the selected index to get closer

	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, attackerSessionState.Weights[layer], attackerSessionState.inputBuffer[layer], attackerSessionState.outputBuffer[layer], output_A, output_B)
	}
	attackerSessionState.LearnIterations += 1
}

func NeuronDotProd(n int, w_k []int, stim_k []int) float64 {
	dot_prod := 0
	for i := 0; i < n; i++ {
		dot_prod += w_k[i] * stim_k[i]
	}
	return float64(dot_prod)
}

func learnGeomAttackReduced(settings MTPMSettings, sessionState *reducedMTPMState, output_A, output_B int, localRand *rand.Rand) {

	if sessionState.networkOutput == output_A {
		for layer := 0; layer < settings.H; layer++ {
			settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, sessionState.Weights[layer], sessionState.inputBuffer[layer], sessionState.outputBuffer[layer], output_A, output_B)
		}
		return
	}

	flipLowestLocalField(settings, sessionState, localRand)

	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, sessionState.Weights[layer], sessionState.inputBuffer[layer], sessionState.outputBuffer[layer], output_A, output_B)
	}
}

func flipLowestLocalField(settings MTPMSettings, sessionState *reducedMTPMState, localRand *rand.Rand) {
	lastLayerIndex := settings.H - 1
	lastLayer := sessionState.Weights[lastLayerIndex]

	minFieldIndexes := []int{}
	minAbsField := float64(settings.L*settings.N[lastLayerIndex] + 1)
	for neuronIndex := 0; neuronIndex < settings.K[lastLayerIndex]; neuronIndex++ {
		absLocalField := math.Abs(tpm_core.NeuronLocalField(settings.N[lastLayerIndex], lastLayer[neuronIndex], sessionState.inputBuffer[lastLayerIndex][neuronIndex]))
		// localField := NeuronDotProd(settings.N[lastLayerIndex], lastLayer[neuronIndex], sessionState.inputBuffer[lastLayerIndex][neuronIndex])
		if absLocalField < minAbsField {
			minFieldIndexes = []int{neuronIndex}
			minAbsField = absLocalField
			continue
		}
		if absLocalField == minAbsField {
			minFieldIndexes = append(minFieldIndexes, neuronIndex)
		}
	}

	selectedIndex := minFieldIndexes[localRand.Intn(len(minFieldIndexes))]

	sessionState.outputBuffer[lastLayerIndex][selectedIndex] *= -1
}
