package main

import "time"

type SaveSessionObject struct {
	FirstLayerK       int
	FirstLayerN       int
	LastLayerK        int
	LastLayerN        int
	StartTime         string
	EndTime           string
	SyncSuccess       int
	SimpleSuccess     int
	GeomSuccess       int
	BruteSuccess      int
	MajoritySuccess   int
	TotalSessionCount int
}

type SessionRecord struct {
	SaveSessionObject
	MTPMSettings
}

func CreateSaveSessionObject(settings MTPMSettings, start time.Time) SaveSessionObject {
	lastLayerIndex := settings.H - 1
	return SaveSessionObject{
		FirstLayerK:       settings.K[0],
		FirstLayerN:       settings.N[0],
		LastLayerK:        settings.K[lastLayerIndex],
		LastLayerN:        settings.N[lastLayerIndex],
		StartTime:         start.Format("2006-01-02 15:04:05"),
		EndTime:           "",
		SyncSuccess:       0,
		SimpleSuccess:     0,
		GeomSuccess:       0,
		BruteSuccess:      0,
		MajoritySuccess:   0,
		TotalSessionCount: 0,
	}
}
