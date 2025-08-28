package main

import (
	"fmt"
	"math/rand"
	"tpm_sync/tpm_core"
)

func testBruteforce(verbose bool, tpmSettings MTPMSettings, attackerCount int, historySize int, localRand *rand.Rand) int {

	state_A := newRandomReducedState(tpmSettings, localRand, historySize)
	state_B := newRandomReducedState(tpmSettings, localRand, historySize)
	groupSession := CreateNewRandomGroupAttackSession(tpmSettings, attackerCount, localRand, historySize)
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
			for i := 0; i < groupSession.AttackerCount; i++ {
				stimulateReducedState(groupSession.Settings, groupSession.Networks[i], input_stimulus) //Only useful when we need to update
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

func CompareGroupAttack(groupSession GroupMTPMAttackSession, state_A, state_B reducedMTPMState) int {
	normal_sync := compareWeightsFromReducedStates(groupSession.Settings, state_A, state_B)
	attacker_sync_count := 0
	for i := 0; i < len(groupSession.Networks); i++ {
		if compareWeightsFromReducedStates(groupSession.Settings, state_A, *groupSession.Networks[i]) {
			attacker_sync_count += 1
		}
	}

	if attacker_sync_count > 0 {
		// fmt.Printf("Attacker success count: %d\n", attacker_sync_count)
		if normal_sync {
			return -2
		} else {
			return -1
		}
	}
	if normal_sync {
		return 1
	}
	return 0
}
