package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/krjakbrjak/fakeqemu/generated/qapi"
	"github.com/krjakbrjak/fakeqemu/src/server"
	"github.com/q-controller/qapi-client/src/client"
	"github.com/q-controller/qapi-client/src/monitor"
)

func main() {
	s, err := server.New("/tmp/fakeqemu.sock", server.QMPInitializer)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		return
	}
	defer s.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		time.Sleep(2 * time.Second)
		monitor, monitorErr := monitor.NewMonitor()
		if monitorErr != nil {
			slog.Error("failed to create monitor", "error", monitorErr)
			return
		}
		defer monitor.Close()

		errCh := monitor.Add("example", "/tmp/fakeqemu.sock")
		for err := range errCh {
			if err != nil {
				slog.Error("monitor add error", "error", err)
				return
			}
			slog.Info("monitor add successful")
			break
		}

		for ev := range monitor.Messages() {
			if ev.Message != nil && ev.Message.Generic != nil {
				slog.Info("received QMP greeting", "greeting", string(ev.Message.Generic))
				break
			}
		}

		capReq, capErr := qapi.PrepareQmpCapabilitiesRequest(qapi.QObjQmpCapabilitiesArg{})
		if capErr != nil {
			slog.Error("failed to prepare qmp_capabilities request", "error", capErr)
			return
		}
		capRes, capExecErr := monitor.Execute("example", client.Request(*capReq))
		if capExecErr != nil {
			slog.Error("monitor execute error for qmp_capabilities", "error", capExecErr)
			return
		}
		if d, ok := capRes.Get(ctx, 5*time.Second); !ok || d.Error != nil {
			slog.Error("qmp_capabilities failed", "ok", ok, "result", d)
			return
		}

		request, requestErr := qapi.PrepareBlockdevSnapshotRequest(qapi.BlockdevSnapshot{
			Node: "example-node",
		})
		if requestErr != nil {
			slog.Error("failed to prepare request", "error", requestErr)
			return
		}
		res := client.Request(*request)
		s.When(qapi.BlockdevSnapshotMethod, func(args json.RawMessage) (json.RawMessage, *client.Error) {
			return nil, &client.Error{
				Class:       "lkasjnf",
				Description: "lksndcljn",
			}
		})
		result, resultErr := monitor.Execute("example", res)
		if resultErr != nil {
			slog.Error("monitor execute error", "error", resultErr)
			return
		}
		d, ok := result.Get(context.Background(), 5*time.Second)
		if !ok {
			slog.Error("failed to get result from monitor")
			return
		}
		if d.Error != nil {
			slog.Error("monitor execute returned error", "error", d.Error)
		}
		slog.Info("monitor execute data", "data", d)
		ex, exErr := qapi.PrepareSystemPowerdownRequest()
		if exErr != nil {
			slog.Error("failed to prepare system powerdown request", "error", exErr)
			return
		}
		s.When(qapi.SystemPowerdownMethod, func(args json.RawMessage) (json.RawMessage, *client.Error) {
			return json.RawMessage(`{}`), nil
		})
		w, wErr := monitor.Execute("example", client.Request(*ex))
		if wErr != nil {
			slog.Error("monitor execute error for system powerdown", "error", wErr)
		}
		_, wOk := w.Get(ctx, 5*time.Second)
		if !wOk {
			slog.Error("failed to get result from monitor for system powerdown")
		}
		event, data := qapi.PrepareSHUTDOWNEvent(qapi.QObjSHUTDOWNArg{
			Guest:  true,
			Reason: "Shutdown",
		})
		go func() {
			s.SendEvent(ctx, client.QAPIEvent{
				Event: event,
				Data:  data,
			})
		}()
		for message := range monitor.Messages() {
			if message.Message != nil && message.Message.Event != nil {
				if message.Message.Event.Event == event {
					slog.Info("received shutdown event")
					break
				}
			}
		}
		slog.Info("system powerdown executed successfully")
		stop()
	}()

	if err := s.Listen(ctx); err != nil {
		slog.Error("server listen error", "error", err)
	}
}
