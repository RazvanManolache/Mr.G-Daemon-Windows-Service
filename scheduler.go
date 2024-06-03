package main

import "time"

func scheduler() {
	go checkSubapplications()
	go checkDiskspace()
	go checkSubApplicationUpdates()
}

func checkSubApplicationUpdates() {
	for {
		interval := CurrentConfig.CheckSubApplicationsUpdateInterval
		if interval == 0 {
			interval = 1440
		}
		time.Sleep(time.Duration(interval) * time.Minute)
		checkSubApplicationUpdatesInternal()
	}

}

//run operations at regular interval
func checkSubapplications() {
	for {
		//run every 5 minutes
		interval := CurrentConfig.CheckSubApplicationsInterval
		if interval == 0 {
			interval = 10
		}
		time.Sleep(time.Duration(interval) * time.Minute)
		listApplicationsInternal()

	}
}

func checkDiskspace() {
	for {
		interval := CurrentConfig.CheckDisksInterval
		if interval == 0 {
			interval = 60
		}
		time.Sleep(time.Duration(interval) * time.Minute)
		listDiskSpaceInternal()

	}
}
