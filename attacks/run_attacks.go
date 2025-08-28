package main

import "github.com/xconstruct/go-pushbullet"

func main() {
	pb := pushbullet.New("o.DokJDY5gFuC3sPf4fLECWT0QOlgkL95E")
	devs, err := pb.Devices()
	if err != nil {
		panic(err)
	}

	err = pb.PushNote(devs[0].Iden, "TPM attacks starting!", "The attacks are starting...")
	if err != nil {
		panic(err)
	}

	SimulateMultipleFiles("configFiles")
	err = pb.PushNote(devs[0].Iden, "TPM attacks finished!", "Time to check the db...")
	if err != nil {
		panic(err)
	}
}
