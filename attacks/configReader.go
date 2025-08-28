package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sourcegraph/conc/pool"
)

var mu sync.Mutex

func ReadFile(filename string) ([]byte, error) {
	// Read file content into a byte slice
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func UnmarshalSettings(data []byte) (BaseSettings, error) {
	var baseSettings BaseSettings
	err := json.Unmarshal(data, &baseSettings)
	if err != nil {
		return BaseSettings{}, err
	}
	return baseSettings, nil
}

// Function to read and deserialize JSON file
func LoadSimulationSettings(filename string) (interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	// Read the entire file into a byte slice
	fileInfo, err := os.Stat(filename)
	if err != nil {
		fmt.Println("Error getting file stats:", err)
		return nil, err
	}
	fileData := make([]byte, fileInfo.Size())
	_, err = file.Read(fileData)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}

	// Decode the file into a map
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(fileData, &rawConfig); err != nil {
		fmt.Println("Error unmarshalling config:", err)
		return nil, err
	}

	return rawConfig, nil
}

func SimulateMultipleFiles(configFileDirectory string) {
	db, err := sql.Open("sqlite3", "file:data.db")
	if err != nil {
		// log.Fatal(err)
		fmt.Println(err)
	}
	defer db.Close()

	// Read init.sql file
	initSQL, err := os.ReadFile("init.sql")
	if err != nil {
		log.Fatalf("Error reading init.sql: %v", err)
	}

	// Execute the SQL script
	_, err = db.Exec(string(initSQL))
	if err != nil {
		log.Fatalf("Error executing init.sql: %v", err)
	}

	fmt.Println("Database initialized successfully.")

	workerPool := pool.New().WithMaxGoroutines(runtime.NumCPU())

	files, err := os.ReadDir(configFileDirectory)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range files {
		fmt.Printf("Reading config file: %s\n", file.Name())

		rawConfig, err := LoadSimulationSettings(filepath.Join(configFileDirectory, file.Name()))
		if err != nil {
			fmt.Printf("Error loading base settings for file %s: %s", file.Name(), err)
			continue
		}
		// Initialize a variable for the base settings
		var baseSettings BaseSettings
		baseSettingsData, _ := json.Marshal(rawConfig)

		if err := json.Unmarshal(baseSettingsData, &baseSettings); err != nil {
			fmt.Printf("Error unmarshalling base settings for file %s: %s", file.Name(), err)
			continue
		}

		fmt.Printf("%s Settings loaded: \n", file.Name())
		fmt.Println(baseSettings)

		for _, rule := range baseSettings.LearnRules {
			for _, m := range baseSettings.MConfigs {
				for _, l := range baseSettings.LConfigs {
					switch strings.ToUpper(baseSettings.TpmType) {
					case "NO_OVERLAP":
						var noOverlapSettings NonOverlappedSettings

						if err := json.Unmarshal(baseSettingsData, &noOverlapSettings); err != nil {
							fmt.Printf("Error unmarshalling noOverlap settings for file %s: %s\n", file.Name(), err)
							continue
						}

						for _, n := range noOverlapSettings.NConfigs {
							for _, k_last := range noOverlapSettings.KlastConfigs {
								tpmInstanceSettings, err := SettingsFactory(n, k_last, l, m, noOverlapSettings.TpmType, rule)
								if err != nil {
									fmt.Printf("Error while creating settings for an instance for file %s: %s \n", file.Name(), err)
									continue
								}
								//HERE IS THE MAGIC
								workerPool.Go(func() { RunAttacks(tpmInstanceSettings, db) })
								// RunAttacks(tpmInstanceSettings, db)
							}
						}
					default:
						var overlapSettings OverlappedSettings

						if err := json.Unmarshal(baseSettingsData, &overlapSettings); err != nil {
							fmt.Printf("Error unmarshalling overlapped settings for file %s: %s\n", file.Name(), err)
						}

						for _, k := range overlapSettings.KConfigs {
							for _, n_0 := range overlapSettings.N0Configs {
								tpmInstanceSettings, err := SettingsFactory(k, n_0, l, m, overlapSettings.TpmType, rule)
								if err != nil {
									fmt.Printf("Error while creating settings for an instance for file %s: %s \n", file.Name(), err)
									continue
								}
								//HERE IS THE MAGIC
								workerPool.Go(func() { RunAttacks(tpmInstanceSettings, db) })
								// RunAttacks(tpmInstanceSettings, db)
							}
						}
					}
				}
			}
		}
		// fmt.Printf("-- All automatic configs finished for file %s --\n", file.Name())
	}
	workerPool.Wait()
	fmt.Printf("-- All automatic configs finished for all files --\n")

}

func RunAttacks(tpmSettings MTPMSettings, db *sql.DB) {
	insertStmt := `
INSERT INTO sessions (
	first_layer_k, first_layer_n, last_layer_k, last_layer_n,
	start_time, end_time, sync_success,
	simple_success, geom_success, brute_success, majority_success,
	total_session_count,
	k, n, l, m, h, learn_rule, scenario
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	sessionObject := CreateSaveSessionObject(tpmSettings, time.Now())
	src := rand.NewSource(time.Now().UnixNano())
	localGoroutineRand := rand.New(src)
	totalSessions := 100
	for i := 0; i < totalSessions; i++ {
		// fmt.Println("attack")
		testResults := testSimpleAttack(false, tpmSettings, localGoroutineRand)
		if testResults < 0 {
			sessionObject.SimpleSuccess += 1
		} else if testResults > 0 {
			sessionObject.SyncSuccess += 1
		}
		sessionObject.TotalSessionCount += 1

		testResults = testGeomAttack(false, tpmSettings, localGoroutineRand)
		if testResults < 0 {
			sessionObject.GeomSuccess += 1
		} else if testResults > 0 {
			sessionObject.SyncSuccess += 1
		}
		sessionObject.TotalSessionCount += 1

		testResults = testBruteforce(false, tpmSettings, 100, 1, localGoroutineRand)
		if testResults < 0 {
			sessionObject.BruteSuccess += 1
		} else if testResults > 0 {
			sessionObject.SyncSuccess += 1
		}
		sessionObject.TotalSessionCount += 1

		testResults = testMajorityFlipping(false, tpmSettings, 100, localGoroutineRand)
		if testResults < 0 {
			sessionObject.MajoritySuccess += 1
		} else if testResults > 0 {
			sessionObject.SyncSuccess += 1
		}
		sessionObject.TotalSessionCount += 1

		// testResults := testGenetic(false, tpmSettings, 100, 20, 10)
	}
	sessionObject.EndTime = time.Now().Format("2006-01-02 15:04:05")
	record := SessionRecord{
		SaveSessionObject: sessionObject, // fill this
		MTPMSettings:      tpmSettings,
	}

	// Serialize K and N
	kJson, _ := json.Marshal(record.K)
	nJson, _ := json.Marshal(record.N)

	// Write to DB (thread-safe)
	mu.Lock()
	_, err := db.Exec(insertStmt,
		record.FirstLayerK,
		record.FirstLayerN,
		record.LastLayerK,
		record.LastLayerN,
		record.StartTime,
		record.EndTime,
		record.SyncSuccess,
		record.SimpleSuccess,
		record.GeomSuccess,
		record.BruteSuccess,
		record.MajoritySuccess,
		record.TotalSessionCount,
		string(kJson),
		string(nJson),
		record.L,
		record.M,
		record.H,
		record.LearnRule,
		record.LinkType,
	)

	mu.Unlock()
	if err != nil {
		log.Println("Insert error:", err)
	}
	fmt.Println("New insert to DB")
}
