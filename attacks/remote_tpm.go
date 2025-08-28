package main

import (
	"errors"
	"math/rand"
	"tpm_sync/tpm_core"
	"tpm_sync/tpm_learnRules"
	"tpm_sync/tpm_stimHandlers"
)

type MTPMSettings struct {
	K                   []int
	N                   []int
	L                   int
	M                   int
	H                   int
	LearnRule           string
	LinkType            string
	stimulationHandlers tpm_stimHandlers.TPMStimulationHandlers
	learnRuleHandler    tpm_learnRules.TPMLearnRuleHandler
}

type LocalMTPMSessionState struct {
	Settings        MTPMSettings
	Weights         [][][]int
	SimIterations   int
	LearnIterations int
	inputBuffer     [][][]int
	outputBuffer    [][]int
	networkOutput   int
}

type reducedMTPMState struct {
	Weights       [][][]int
	SimIterations int
	inputBuffer   [][][]int
	outputBuffer  [][]int
	networkOutput int

	outputHistory []int
	historySize   int
}

type GroupMTPMAttackSession struct {
	AttackerCount   int
	Settings        MTPMSettings
	SimIterations   int
	LearnIterations int
	Networks        []*reducedMTPMState
}

func newRandomState(settings MTPMSettings, localRand *rand.Rand) LocalMTPMSessionState {
	newWeights := make([][][]int, settings.H)
	for layer := 0; layer < settings.H; layer++ {
		newWeights[layer] = tpm_core.CreateRandomLayerWeightsArray(settings.K[layer], settings.N[layer], settings.L, localRand)
	}
	localState := LocalMTPMSessionState{
		Settings:        settings,
		Weights:         newWeights,
		SimIterations:   0,
		LearnIterations: 0,
		inputBuffer:     make([][][]int, settings.H),
		outputBuffer:    make([][]int, settings.H),
		networkOutput:   0,
	}
	return localState
}

func newRandomReducedState(settings MTPMSettings, localRand *rand.Rand, historySize int) reducedMTPMState {
	newWeights := make([][][]int, settings.H)
	for layer := 0; layer < settings.H; layer++ {
		newWeights[layer] = tpm_core.CreateRandomLayerWeightsArray(settings.K[layer], settings.N[layer], settings.L, localRand)
	}
	localState := reducedMTPMState{
		Weights:       newWeights,
		SimIterations: 0,
		inputBuffer:   make([][][]int, settings.H),
		outputBuffer:  make([][]int, settings.H),
		networkOutput: 0,
		historySize:   historySize,
		outputHistory: make([]int, historySize),
	}
	return localState
}

func stimulate(sessionState *LocalMTPMSessionState, firstLayerInput [][]int) {
	settings := sessionState.Settings
	inputs := sessionState.inputBuffer
	inputs[0] = firstLayerInput
	outputs := sessionState.outputBuffer

	//Stimulate all layers, to avoid overflowing the inputs array we do the last layer separately -> (There is no next layer, no more inputs)
	for layer := 0; layer < settings.H-1; layer++ {
		outputs[layer] = tpm_core.StimulateLayer(inputs[layer], sessionState.Weights[layer], settings.K[layer], settings.N[layer])
		inputs[layer+1] = settings.stimulationHandlers.CreateStimulusFromLayerOutput(outputs[layer], settings.K[layer+1], settings.N[layer+1])
	}
	outputs[settings.H-1] = tpm_core.StimulateLayer(inputs[settings.H-1], sessionState.Weights[settings.H-1], settings.K[settings.H-1], settings.N[settings.H-1])
	sessionState.networkOutput = tpm_core.Thau(outputs[settings.H-1], settings.K[settings.H-1])
	sessionState.SimIterations += 1
}

func stimulateReducedState(settings MTPMSettings, mtpmState *reducedMTPMState, firstLayerInput [][]int) {
	inputs := mtpmState.inputBuffer
	inputs[0] = firstLayerInput
	outputs := mtpmState.outputBuffer

	//Stimulate all layers, to avoid overflowing the inputs array we do the last layer separately -> (There is no next layer, no more inputs)
	for layer := 0; layer < settings.H-1; layer++ {
		outputs[layer] = tpm_core.StimulateLayer(inputs[layer], mtpmState.Weights[layer], settings.K[layer], settings.N[layer])
		inputs[layer+1] = settings.stimulationHandlers.CreateStimulusFromLayerOutput(outputs[layer], settings.K[layer+1], settings.N[layer+1])
	}
	outputs[settings.H-1] = tpm_core.StimulateLayer(inputs[settings.H-1], mtpmState.Weights[settings.H-1], settings.K[settings.H-1], settings.N[settings.H-1])
	mtpmState.networkOutput = tpm_core.Thau(outputs[settings.H-1], settings.K[settings.H-1])

	mtpmState.SimIterations += 1
	outputIndex := mtpmState.SimIterations % mtpmState.historySize
	mtpmState.outputHistory[outputIndex] = mtpmState.networkOutput
}

// Learns using the input stored in the buffer.
func learn(sessionState *LocalMTPMSessionState, remoteOutput int) {
	settings := sessionState.Settings

	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, sessionState.Weights[layer], sessionState.inputBuffer[layer], sessionState.outputBuffer[layer], sessionState.networkOutput, remoteOutput)
	}
	sessionState.LearnIterations += 1
}

func learnReducedState(settings MTPMSettings, mtpmState *reducedMTPMState, remoteOutput int) {
	for layer := 0; layer < settings.H; layer++ {
		settings.learnRuleHandler.TPMLearnLayer(settings.K[layer], settings.N[layer], settings.L, mtpmState.Weights[layer], mtpmState.inputBuffer[layer], mtpmState.outputBuffer[layer], mtpmState.networkOutput, remoteOutput)
	}
}

// Convert from 1D to 3D weights array (microcontroller uses 1d array even in multilayer)
func convertWeightsTo3D(oldWeights []int, k, n []int, h int) ([][][]int, error) {
	netDataSize := tpm_core.GetNetworkDataSize(h, k, n)
	if len(oldWeights) != netDataSize {
		return nil, errors.New("The weights length dont match the MTPM structure.")
	}
	output := make([][][]int, h)
	current_index := 0
	for current_index < netDataSize {
		for layer_index := 0; layer_index < h; layer_index++ {
			output[layer_index] = make([][]int, k[layer_index])
			for neuron_index := 0; neuron_index < k[layer_index]; neuron_index++ {
				output[layer_index][neuron_index] = make([]int, n[layer_index])
				for stim_index := 0; stim_index < n[layer_index]; stim_index++ {
					output[layer_index][neuron_index][stim_index] = oldWeights[current_index]
					current_index += 1
				}
			}
		}
	}
	return output, nil
}

func convertInputsTo1D(input [][]int, k_0 int, n_0 int) []int {
	convertedInput := make([]int, k_0*n_0)
	for neuron_index := 0; neuron_index < k_0; neuron_index++ {
		for stim_index := 0; stim_index < n_0; stim_index++ {
			convertedInput[neuron_index*n_0+stim_index] = input[neuron_index][stim_index]
		}
	}

	return convertedInput
}

func compareWeightsFromStates(settings MTPMSettings, state_A LocalMTPMSessionState, state_B LocalMTPMSessionState) bool {
	return tpm_core.CompareWeights(settings.H, settings.K, settings.N, state_A.Weights, state_B.Weights)
}

func compareWeightsFromReducedStates(settings MTPMSettings, state_A reducedMTPMState, state_B reducedMTPMState) bool {
	return tpm_core.CompareWeights(settings.H, settings.K, settings.N, state_A.Weights, state_B.Weights)
}

func CreateNewRandomGroupAttackSession(settings MTPMSettings, attackerCount int, localRand *rand.Rand, historySize int) GroupMTPMAttackSession {
	groupSession := GroupMTPMAttackSession{
		AttackerCount:   attackerCount,
		Settings:        settings,
		SimIterations:   0,
		LearnIterations: 0,
		Networks:        make([]*reducedMTPMState, attackerCount),
	}
	for i := 0; i < attackerCount; i++ {
		newRandomState := newRandomReducedState(settings, localRand, historySize)
		groupSession.Networks[i] = &newRandomState
	}
	return groupSession
}

func CreateNewEmptyGroupAttackSession(settings MTPMSettings, attackerCount int, localRand *rand.Rand, historySize int) GroupMTPMAttackSession {
	groupSession := GroupMTPMAttackSession{
		AttackerCount:   attackerCount,
		Settings:        settings,
		SimIterations:   0,
		LearnIterations: 0,
		Networks:        make([]*reducedMTPMState, 1),
	}
	newRandomState := newRandomReducedState(settings, localRand, historySize)
	groupSession.Networks[0] = &newRandomState
	return groupSession
}

func copyWeights(src [][][]int) [][][]int {
	if src == nil {
		return nil
	}

	dst := make([][][]int, len(src))
	for i := range src {
		dst[i] = make([][]int, len(src[i]))
		for j := range src[i] {
			dst[i][j] = make([]int, len(src[i][j]))
			copy(dst[i][j], src[i][j])
		}
	}
	return dst
}

func copyOutputBuffer(src [][]int) [][]int {
	if src == nil {
		return nil
	}

	dst := make([][]int, len(src))
	for i := range src {
		dst[i] = make([]int, len(src[i]))
		copy(dst[i], src[i])
	}
	return dst
}

func DeepCopyReducedMTPMState(src *reducedMTPMState) *reducedMTPMState {
	if src == nil {
		return nil
	}

	// Deep copy for Weights
	weightsCopy := make([][][]int, len(src.Weights))
	for i := range src.Weights {
		weightsCopy[i] = make([][]int, len(src.Weights[i]))
		for j := range src.Weights[i] {
			weightsCopy[i][j] = make([]int, len(src.Weights[i][j]))
			copy(weightsCopy[i][j], src.Weights[i][j])
		}
	}

	// Deep copy for inputBuffer
	inputBufferCopy := make([][][]int, len(src.inputBuffer))
	for i := range src.inputBuffer {
		inputBufferCopy[i] = make([][]int, len(src.inputBuffer[i]))
		for j := range src.inputBuffer[i] {
			inputBufferCopy[i][j] = make([]int, len(src.inputBuffer[i][j]))
			copy(inputBufferCopy[i][j], src.inputBuffer[i][j])
		}
	}

	// Deep copy for outputBuffer
	outputBufferCopy := make([][]int, len(src.outputBuffer))
	for i := range src.outputBuffer {
		outputBufferCopy[i] = make([]int, len(src.outputBuffer[i]))
		copy(outputBufferCopy[i], src.outputBuffer[i])
	}

	return &reducedMTPMState{
		Weights:       weightsCopy,
		SimIterations: 0,
		inputBuffer:   inputBufferCopy,
		outputBuffer:  outputBufferCopy,
		networkOutput: src.networkOutput,
		historySize:   src.historySize,
		outputHistory: make([]int, src.historySize),
	}
}
