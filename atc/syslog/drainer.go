package syslog

import (
	"context"
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager/lagerctx"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/event"
)

//go:generate counterfeiter . Drainer

type Drainer interface {
	Run(context.Context) error
}

type drainer struct {
	hostname     string
	transport    string `yaml:"transport"`
	address      string `yaml:"address"`
	caCerts      []string
	buildFactory db.BuildFactory
}

func NewDrainer(transport string, address string, hostname string, caCerts []string, buildFactory db.BuildFactory) Drainer {
	return &drainer{
		hostname:     hostname,
		transport:    transport,
		address:      address,
		buildFactory: buildFactory,
		caCerts:      caCerts,
	}
}

func (d *drainer) Run(ctx context.Context) error {
	logger := lagerctx.FromContext(ctx).Session("syslog")

	builds, err := d.buildFactory.GetDrainableBuilds()
	if err != nil {
		logger.Error("Syslog drainer getting drainable builds error.", err)
		return err
	}

	if len(builds) > 0 {
		syslog, err := Dial(d.transport, d.address, d.caCerts)
		if err != nil {
			logger.Error("Syslog drainer connecting to server error.", err)
			return err
		}
		defer syslog.Close()

		for _, build := range builds {
			events, err := build.Events(0)
			if err != nil {
				logger.Error("Syslog drainer getting build events error.", err)
				return err
			}

			for {
				ev, err := events.Next()
				if err != nil {
					if err == db.ErrEndOfBuildEventStream {
						break
					}
					logger.Error("Syslog drainer getting next event error.", err)
					return err
				}

				if ev.Event == "log" {
					var log event.Log

					err := json.Unmarshal(*ev.Data, &log)
					if err != nil {
						logger.Error("Syslog drainer unmarshalling log error.", err)
						return err
					}

					payload := log.Payload
					tag := build.TeamName() + "/" + build.PipelineName() + "/" + build.JobName() + "/" + build.Name() + "/" + string(log.Origin.ID)

					err = syslog.Write(d.hostname, tag, time.Unix(log.Time, 0), payload)
					if err != nil {
						logger.Error("Syslog drainer sending to server error.", err)
						return err
					}
				}
			}

			err = build.SetDrained(true)
			if err != nil {
				logger.Error("Syslog drainer setting drained on build error.", err)
				return err
			}
		}
	}
	return nil
}
