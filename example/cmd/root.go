package cmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"qga-example/generated/qapi"

	"github.com/q-controller/qapi-client/src/client"
	"github.com/q-controller/qapi-client/src/monitor"
	"github.com/spf13/cobra"
)

var socketPath string

type ShutdownProcessor struct {
	qapi.DefaultEventProcessor
	shutdown bool
}

func (ep *ShutdownProcessor) ProcessSHUTDOWN(arg qapi.QObjSHUTDOWNArg) error {
	slog.Info("Processing event", "event", "SHUTDOWN", "data", arg)
	ep.shutdown = true
	return nil
}

type Greeting struct {
	QMP struct {
		Version      qapi.VersionInfo       `json:"version"`
		Capabilities qapi.QMPCapabilityList `json:"capabilities"`
	} `json:"QMP"`
}

var rootCmd = &cobra.Command{
	Use:   "qga-example",
	Short: "A brief description of your application",
	RunE: func(cmd *cobra.Command, args []string) error {
		monitor, monitorErr := monitor.NewMonitor()
		if monitorErr != nil {
			return monitorErr
		}

		defer monitor.Close()
		msgCh := monitor.Messages()

		for {
			addFut := monitor.Add("example instance", socketPath)
			if err, ok := <-addFut; !ok || err != nil {
				slog.Error("Could not add instance", "error", err)
				continue
			}
			break
		}

		ep := ShutdownProcessor{}
		for data := range msgCh {
			msg := data.Message
			if msg == nil {
				continue
			}
			if msg.Event != nil {
				qapi.ProcessEvent(&ep, msg.Event.Event, msg.Event.Data)
				if ep.shutdown {
					break
				}
			}

			if msg.Generic != nil {
				var greeting Greeting
				if err := json.Unmarshal(msg.Generic, &greeting); err == nil {
					if req, reqErr := qapi.PrepareQmpCapabilitiesRequest(qapi.QObjQmpCapabilitiesArg{}); reqErr == nil {
						if ch, chErr := monitor.Execute("example instance", client.Request(*req)); chErr == nil {
							res, ok := ch.Get(context.Background(), -1)
							if !ok {
								continue
							}
							if res.Error == nil {
								if schemaReq, schemaErr := qapi.PrepareQueryStatusRequest(); schemaErr == nil {
									if statusCh, statusChErr := monitor.Execute("example instance", client.Request(*schemaReq)); statusChErr == nil {
										status, ok := statusCh.Get(context.Background(), -1)
										if !ok {
											continue
										}
										if status.Return != nil {
											var statusInfo qapi.StatusInfo
											if unmarshalErr := json.Unmarshal(status.Return, &statusInfo); unmarshalErr == nil {
												slog.Info("Retrieved status of the instance", "status", statusInfo.Status)
												if shReq, shReqErr := qapi.PrepareSystemPowerdownRequest(); shReqErr == nil {
													if shutdownCh, shutdownChErr := monitor.Execute("example instance", client.Request(*shReq)); shutdownChErr == nil {
														_, _ = shutdownCh.Get(context.Background(), -1)
														slog.Info("Shutdown command sent to the instance")
													} else {
														slog.Error("Failed to send shutdown command", "error", shutdownChErr)
													}
												} else {
													slog.Error("Failed to prepare shutdown request", "error", shReqErr)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&socketPath, "socket", "", "Path to QAPI socket")
	rootCmd.MarkFlagRequired("socket")
}
