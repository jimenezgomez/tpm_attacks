module attacks

go 1.24.2

replace tpm_sync => /home/kwwa/Documents/golang_tpm/src/tpm_sync

require (
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/sourcegraph/conc v0.3.0
	tpm_sync v0.0.0-00010101000000-000000000000
)

require github.com/xconstruct/go-pushbullet v0.0.0-20171206132031-67759df45fbb // indirect
