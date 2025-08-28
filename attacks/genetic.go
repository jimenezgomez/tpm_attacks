package main

import (
	"fmt"
	"math"
	"math/rand"
	"tpm_sync/tpm_core"
)

const ARRAY_LIMIT_SIZE = 1000

func testGenetic(verbose bool, tpmSettings MTPMSettings, attackerCountLimit int, historySize int, fitThreshold int, localRand *rand.Rand) int {

	state_A := newRandomReducedState(tpmSettings, localRand, historySize)
	state_B := newRandomReducedState(tpmSettings, localRand, historySize)
	groupSession := CreateNewEmptyGroupAttackSession(tpmSettings, attackerCountLimit, localRand, historySize)
	stopSync := false
	compareWeightsResult := 0
	threshold := 100_000
	for !stopSync {

		if groupSession.SimIterations >= threshold {
			break
		}

		input_stimulus := tpm_core.CreateRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M, localRand)
		stimulateReducedState(tpmSettings, &state_A, input_stimulus)
		stimulateReducedState(tpmSettings, &state_B, input_stimulus)
		groupSession.SimIterations += 1
		fmt.Println(groupSession.SimIterations)
		if state_A.networkOutput == state_B.networkOutput {
			learnReducedState(tpmSettings, &state_B, state_A.networkOutput)
			learnReducedState(tpmSettings, &state_A, state_B.networkOutput)
			previousAttackerCount := (len(groupSession.Networks))
			// fmt.Println(previousAttackerCount)
			for i := 0; i < previousAttackerCount; i++ {
				// fmt.Println("Stimulating attacker: ", i)
				stimulateReducedState(groupSession.Settings, groupSession.Networks[i], input_stimulus) //Stimulating is only useful when we need to update
				if previousAttackerCount > attackerCountLimit {
					//delete all unfit networks
					deleteUnfit(groupSession.Networks, state_A.outputHistory, historySize, fitThreshold)
				}
				mutations := calculateCombinations(tpmSettings, *groupSession.Networks[i], state_A.networkOutput)
				// fmt.Println("Mutations: ", len(mutations))

				currentAttCount := previousAttackerCount
				for _, attacker := range mutations {
					if attacker != nil && currentAttCount < ARRAY_LIMIT_SIZE {
						groupSession.Networks = append(groupSession.Networks, attacker)
						currentAttCount++ //To ease the check in the if statement
					}
				}
			}
			//A second pass, because we also need to update the mutations that have been created
			for i := 0; i < previousAttackerCount; i++ {
				learnReducedState(groupSession.Settings, groupSession.Networks[i], state_A.networkOutput)
			}
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

func calculateCombinations(settings MTPMSettings, localState reducedMTPMState, desiredOutput int) []*reducedMTPMState {
	lastLayerIndex := settings.H - 1
	lastLayerNeuronCount := settings.K[lastLayerIndex]
	var newMutations []*reducedMTPMState
	if localState.networkOutput != desiredOutput {
		newMutations = make([]*reducedMTPMState, lastLayerNeuronCount)
		for i := 0; i < lastLayerNeuronCount; i++ {
			newMutations[i] = DeepCopyReducedMTPMState(&localState)
			newMutations[i].outputBuffer[lastLayerIndex][i] *= -1
		}
	} else {
		newMutations = make([]*reducedMTPMState, int(math.Pow(2, float64(lastLayerNeuronCount-1))))
		for i := 0; i < lastLayerNeuronCount; i++ {
			for j := i + 1; j < lastLayerNeuronCount; j++ {
				newMutations[i] = DeepCopyReducedMTPMState(&localState)
				newMutations[i].outputBuffer[lastLayerIndex][i] *= -1
				newMutations[i].outputBuffer[lastLayerIndex][j] *= -1
			}
		}
	}
	return newMutations
	// fmt.Println(localState.outputBuffer)
	// for _, v := range newMutations {
	// 	if v != nil {
	// 		fmt.Println(v.outputBuffer)
	// 	}
	// }
}

func deleteUnfit(oldAttackers []*reducedMTPMState, referenceOutputHistory []int, historyCount int, matchCount int) []*reducedMTPMState {

	fitAttackers := make([]*reducedMTPMState, 0, historyCount)
	for i := 0; i < len(oldAttackers); i++ {
		if checkFit(oldAttackers[i], referenceOutputHistory, historyCount, matchCount) {
			fitAttackers = append(fitAttackers, oldAttackers[i])
		} else {
			//remove from memory?
		}
	}

	return fitAttackers
}

func checkFit(attacker *reducedMTPMState, referenceOutputHistory []int, historyCount int, minMatchCount int) bool {
	matchCount := 0
	for i := 0; i < historyCount; i++ {
		if attacker.outputHistory[i] == referenceOutputHistory[i] {
			matchCount++
		}
	}
	if matchCount >= minMatchCount {
		return true
	}

	return false
}
