package core

import (
	"time"

	"github.com/go-co-op/gocron/v2"
)

var c gocron.Scheduler

func initA() {
	// create a scheduler
	var err error
	c, err = gocron.NewScheduler()
	if err != nil {
		// handle error
		panic(err)
	}
}

func StartCron() {
	c.Start()
}

func AddJob() (string, error) {
	// add a job to the scheduler
	j, err := c.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			func(a string, b int) {
				// do things
			},
			"hello",
			1,
		),
	)
	if err != nil {
		return "", err
	}

	return j.ID().String(), nil
}
