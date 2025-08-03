package cmd

import (
	"encoding/json"
	"log/slog"
	"os"
	"qga-example/generated/qapi"
	"time"

	"github.com/q-controller/qapi-client/src/client"
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
		monitor, monitorErr := client.NewMonitor()
		if monitorErr != nil {
			return monitorErr
		}

		defer monitor.Close()
		msgCh := monitor.Start()

		for {
			if addErr := monitor.Add("example instance", socketPath); addErr != nil {
				slog.Error("Could not add instance", "error", addErr)
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}

		ep := ShutdownProcessor{}
		for msg := range msgCh {
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
							res := <-ch
							if res.Raw.Error == nil {
								if schemaReq, schemaErr := qapi.PrepareQueryStatusRequest(); schemaErr == nil {
									if statusCh, statusChErr := monitor.Execute("example instance", client.Request(*schemaReq)); statusChErr == nil {
										status := <-statusCh
										if status.Raw.Return != nil {
											var statusInfo qapi.StatusInfo
											if unmarshalErr := json.Unmarshal(status.Raw.Return, &statusInfo); unmarshalErr == nil {
												slog.Info("Retrieved status of the instance", "status", statusInfo.Status)
												if shReq, shReqErr := qapi.PrepareSystemPowerdownRequest(); shReqErr == nil {
													if shutdownCh, shutdownChErr := monitor.Execute("example instance", client.Request(*shReq)); shutdownChErr == nil {
														<-shutdownCh
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
