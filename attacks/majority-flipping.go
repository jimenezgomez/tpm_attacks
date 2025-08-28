package main

import (
	"fmt"
	"math/rand"
	"strings"
	"tpm_sync/tpm_core"
)

const START_PHASE_THRESHOLD = 100

func testMajorityFlipping(verbose bool, tpmSettings MTPMSettings, attackerCount int, localRand *rand.Rand) int {

	state_A := newRandomReducedState(tpmSettings, localRand, 1)
	state_B := newRandomReducedState(tpmSettings, localRand, 1)
	groupSession := CreateNewRandomGroupAttackSession(tpmSettings, attackerCount, localRand, 1)
	stopSync := false
	compareWeightsResult := 0
	threshold := 100_000
	for !stopSync {

		if groupSession.SimIterations >= threshold {
			break
		}

		// fmt.Println(groupSession.SimIterations)
		input_stimulus := tpm_core.CreateRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M, localRand)
		stimulateReducedState(tpmSettings, &state_A, input_stimulus)
		stimulateReducedState(tpmSettings, &state_B, input_stimulus)
		groupSession.SimIterations += 1
		if state_A.networkOutput == state_B.networkOutput {
			learnReducedState(tpmSettings, &state_B, state_A.networkOutput)
			learnReducedState(tpmSettings, &state_A, state_B.networkOutput)
			learnMajorityFlipping(groupSession, state_A.networkOutput, state_B.networkOutput, input_stimulus, localRand)
			groupSession.LearnIterations += 1
		}
		compareWeightsResult = CompareGroupAttack(groupSession, state_A, state_B)
		stopSync = compareWeightsResult != 0
	}
	if verbose {
		if compareWeightsResult < 0 {
			fmt.Println("[!] brute force attack was successful!")
			if compareWeightsResult == -2 {
				fmt.Println("[!!] brute force attack on sync at the same time as B.")
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
	}

	return compareWeightsResult
}

func learnMajorityFlipping(sessionState GroupMTPMAttackSession, output_A, output_B int, input_stimulus [][]int, localRand *rand.Rand) {
	// During a certain time, perform flipping attack
	// After a certain time, start performing:
	// - Majority-Flipping attack on Even time steps
	// - Flipping attack on Odd time steps

	operationIndex := sessionState.SimIterations % 2

	if sessionState.SimIterations < START_PHASE_THRESHOLD || operationIndex == 1 {
		for i := 0; i < sessionState.AttackerCount; i++ {
			stimulateReducedState(sessionState.Settings, sessionState.Networks[i], input_stimulus)
			learnGeomAttackReduced(sessionState.Settings, sessionState.Networks[i], output_A, output_B, localRand)
		}
	} else {
		referenceCombination := getMostCommonCombination(sessionState.Settings, sessionState.Networks, input_stimulus, output_A, localRand)
		for i := 0; i < sessionState.AttackerCount; i++ {
			learnReducedStateWithReference(sessionState.Settings, sessionState.Networks[i], output_A, *referenceCombination)
		}
	}
}

func keyFromList(list []int) string {
	var sb strings.Builder
	for i, val := range list {
		if i > 0 {
			sb.WriteByte(',') // delimiter
		}
		sb.WriteString(fmt.Sprint(val))
	}
	return sb.String()
}

func encodeAsBits(list []int) uint64 {
	var bits uint64
	for _, v := range list {
		bits <<= 1
		if v == 1 {
			bits |= 1
		}
	}
	return bits
}

type ComboCount struct {
	Combination *[][]int
	Count       int
}

func getMostCommonCombination(tpmSettings MTPMSettings, tpmStates []*reducedMTPMState, input_stimulus [][]int, output_A int, localRand *rand.Rand) *[][]int {
	counts := make(map[uint64]*ComboCount)
	lastLayerIndex := tpmSettings.H - 1
	var maxKey uint64
	maxCount := 0
	for _, tpm := range tpmStates {

		stimulateReducedState(tpmSettings, tpm, input_stimulus)
		if tpm.networkOutput != output_A {
			flipLowestLocalField(tpmSettings, tpm, localRand) //This way all attackers have the same output as Alice
		}

		key := encodeAsBits(tpm.outputBuffer[lastLayerIndex])
		if entry, ok := counts[key]; ok {
			entry.Count++

			if counts[key].Count > maxCount {
				maxCount = counts[key].Count
				maxKey = key
			}

		} else {
			counts[key] = &ComboCount{
				Combination: &tpm.outputBuffer, // original list reference
				Count:       1,
			}
			if maxCount == 0 {
				maxCount = 1
				maxKey = key
			}
		}
	}

	if counts[maxKey] == nil {
		panic("maxKey was never set correctly")
	}
	return counts[maxKey].Combination
}

func learnReducedStateWithReference(settings MTPMSettings, mtpmState *reducedMTPMState, remoteOutput int, referenceOutputBuffer [][]int) {
	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, mtpmState.Weights[layer], mtpmState.inputBuffer[layer], referenceOutputBuffer[layer], mtpmState.networkOutput, remoteOutput)
	}
}
